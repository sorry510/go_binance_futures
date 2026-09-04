package toolruntime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"go_binance_futures/agent/contextengine"
	"go_binance_futures/agent/permission"
	"go_binance_futures/agent/security"
	"go_binance_futures/agent/tools"
)

type Runtime struct {
	registry              *tools.Registry
	policy                permission.Policy
	contextEngine         *contextengine.Engine
	cache                 *memoryCache
	schemas               *schemaValidator
	defaultMaxResultBytes int
	now                   func() time.Time
}

type Config struct {
	Registry              *tools.Registry
	Policy                permission.Policy
	ContextEngine         *contextengine.Engine
	DefaultMaxResultBytes int
	Now                   func() time.Time
}

func New(config Config) (*Runtime, error) {
	if config.Registry == nil {
		return nil, fmt.Errorf("tool runtime requires registry")
	}
	if config.Policy == nil {
		policy := permission.AllowReadOnly()
		config.Policy = policy
	}
	if config.ContextEngine == nil {
		config.ContextEngine = contextengine.New()
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	return &Runtime{
		registry: config.Registry, policy: config.Policy, contextEngine: config.ContextEngine,
		cache: newMemoryCache(), schemas: newSchemaValidator(), defaultMaxResultBytes: config.DefaultMaxResultBytes, now: config.Now,
	}, nil
}

func (runtime *Runtime) Descriptor(name string) (ToolDescriptor, bool) {
	if runtime == nil {
		return ToolDescriptor{}, false
	}
	selected, ok := runtime.registry.Get(strings.TrimSpace(name))
	if !ok {
		return ToolDescriptor{}, false
	}
	return descriptorFromTool(selected), true
}

func descriptorFromTool(selected tools.Tool) ToolDescriptor {
	metadata := selected.Metadata()
	sourceType := SourceType(strings.TrimSpace(metadata.SourceType))
	if sourceType == "" {
		sourceType = SourceNative
	}
	return ToolDescriptor{
		CanonicalName: strings.TrimSpace(selected.Name()), SourceType: sourceType,
		Description: strings.TrimSpace(selected.Description()), InputSchema: append(json.RawMessage(nil), metadata.InputSchema...),
		OutputSchema: append(json.RawMessage(nil), metadata.OutputSchema...), Risk: selected.Risk(), Idempotent: metadata.Idempotent,
		TimeoutMs: metadata.Timeout.Milliseconds(), CachePolicy: NewCachePolicy(metadata.CacheTTL), ProviderRef: strings.TrimSpace(metadata.ProviderRef),
		ProtocolVersion: metadata.ProtocolVersion, CatalogHash: metadata.CatalogHash, SchemaHash: metadata.SchemaHash,
		MaxResultBytes: metadata.MaxResultBytes,
	}
}

func (runtime *Runtime) Check(request ExecuteRequest) (ToolDescriptor, error) {
	if runtime == nil {
		return ToolDescriptor{}, newError(ErrorInternal, request.ToolName, nil, "tool runtime is nil")
	}
	selected, exists := runtime.registry.Get(strings.TrimSpace(request.ToolName))
	if !exists {
		return ToolDescriptor{}, newError(ErrorNotFound, request.ToolName, nil, "tool %q is not registered", request.ToolName)
	}
	descriptor := descriptorFromTool(selected)
	if request.AllowedTools != nil && !request.AllowedTools[descriptor.CanonicalName] {
		return descriptor, newError(ErrorPermission, descriptor.CanonicalName, nil, "skill %q does not allow tool %q", request.SkillName, descriptor.CanonicalName)
	}
	if err := runtime.policy.Allow(request.SkillName, descriptor.CanonicalName, descriptor.Risk); err != nil {
		return descriptor, newError(ErrorPermission, descriptor.CanonicalName, err, "%s", err.Error())
	}
	return descriptor, nil
}

func (runtime *Runtime) Execute(ctx context.Context, request ExecuteRequest) (ExecuteResult, error) {
	startedAt := runtime.now().UTC()
	descriptor, checkErr := runtime.Check(request)
	if checkErr != nil {
		return ExecuteResult{Descriptor: descriptor}, checkErr
	}
	selected, _ := runtime.registry.Get(descriptor.CanonicalName)
	arguments := normalizeArguments(request.Arguments)
	argumentsHash, err := canonicalHash(arguments)
	trace := newTrace(descriptor, argumentsHash, request)
	if err != nil {
		classified := newError(ErrorInvalidInput, descriptor.CanonicalName, err, "invalid tool arguments: %s", err.Error())
		return runtime.errorResult(descriptor, trace, startedAt, classified), nil
	}
	if err := runtime.schemas.validate(descriptor.InputSchema, arguments); err != nil {
		classified := newError(ErrorInvalidInput, descriptor.CanonicalName, err, "tool %q arguments do not match input_schema: %s", descriptor.CanonicalName, err.Error())
		return runtime.errorResult(descriptor, trace, startedAt, classified), nil
	}

	cacheKey := ""
	cacheTTL := time.Duration(descriptor.CachePolicy.TTLms) * time.Millisecond
	if descriptor.Risk == permission.RiskRead && descriptor.Idempotent && cacheTTL > 0 {
		cacheKey = descriptor.CanonicalName + ":" + argumentsHash
		if cachedRaw, hit := runtime.cache.get(cacheKey, startedAt); hit {
			value, restoreErr := restoreValue(selected, cachedRaw)
			if restoreErr == nil {
				return runtime.successResult(descriptor, trace, startedAt, value, cachedRaw, request.MaxResultBytes, true)
			}
		}
	}

	toolCtx := ctx
	cancel := func() {}
	if descriptor.TimeoutMs > 0 {
		toolCtx, cancel = context.WithTimeout(ctx, time.Duration(descriptor.TimeoutMs)*time.Millisecond)
	}
	value, toolErr := selected.Execute(toolCtx, arguments)
	cancel()
	if toolErr != nil {
		classified := newError(classifyToolError(toolErr), descriptor.CanonicalName, toolErr, "%s", toolErr.Error())
		return runtime.errorResult(descriptor, trace, startedAt, classified), nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return ExecuteResult{Descriptor: descriptor, Trace: trace}, newError(ErrorInternal, descriptor.CanonicalName, err, "encode tool %q result: %s", descriptor.CanonicalName, err.Error())
	}
	if err := runtime.schemas.validate(descriptor.OutputSchema, raw); err != nil {
		classified := newError(ErrorInternal, descriptor.CanonicalName, err, "tool %q result does not match output_schema: %s", descriptor.CanonicalName, err.Error())
		return runtime.errorResult(descriptor, trace, startedAt, classified), nil
	}
	if cacheKey != "" {
		runtime.cache.set(cacheKey, raw, cacheTTL, runtime.now().UTC())
	}
	return runtime.successResult(descriptor, trace, startedAt, value, raw, request.MaxResultBytes, false)
}

func (runtime *Runtime) successResult(descriptor ToolDescriptor, trace Trace, startedAt time.Time, value any, raw json.RawMessage, requestMaxBytes int, cacheHit bool) (ExecuteResult, error) {
	observedAt := runtime.now().UTC()
	safeRaw := []byte(security.RedactPayload(string(raw)))
	var safeValue any
	if err := json.Unmarshal(safeRaw, &safeValue); err != nil {
		return ExecuteResult{Descriptor: descriptor, Trace: trace}, newError(ErrorInternal, descriptor.CanonicalName, err, "decode redacted tool result: %s", err.Error())
	}
	conversion, err := runtime.contextEngine.ConvertToolResult(descriptor.CanonicalName, safeValue, observedAt)
	if err != nil {
		return ExecuteResult{Descriptor: descriptor, Trace: trace}, newError(ErrorInternal, descriptor.CanonicalName, err, "%s", err.Error())
	}
	maxBytes := effectiveResultLimit(runtime.defaultMaxResultBytes, descriptor.MaxResultBytes, requestMaxBytes)
	data, warnings, truncated := trimResult(safeRaw, maxBytes)
	partial := truncated
	for _, evidence := range conversion.Evidence {
		if len(evidence.DataMissing) > 0 {
			partial = true
			warnings = append(warnings, "source reports data_missing")
		}
	}
	freshnessStale := false
	asOf := ""
	for _, evidence := range conversion.Evidence {
		if asOf == "" && evidence.AsOf != "" {
			asOf = evidence.AsOf
		}
		if evidence.Freshness == contextengine.FreshnessStale || evidence.Freshness == contextengine.FreshnessMissing {
			freshnessStale = true
		}
	}
	errorType := ErrorType("")
	if freshnessStale {
		errorType = ErrorStale
		warnings = append(warnings, "source freshness policy is not satisfied")
	} else if partial {
		errorType = ErrorPartial
	}
	contentHash := contextengine.ContentHash(string(safeRaw))
	duration := runtime.now().UTC().Sub(startedAt)
	envelope := ToolResultEnvelope{Data: data, Source: descriptor.CanonicalName, AsOf: asOf, DurationMs: duration.Milliseconds(), CacheHit: cacheHit, Partial: partial, Warnings: uniqueStrings(warnings), ErrorType: errorType, RawSize: len(raw), ContentHash: contentHash}
	trace.DurationMs, trace.CacheHit, trace.Partial, trace.ErrorType, trace.RawSize, trace.ContentHash = envelope.DurationMs, cacheHit, partial, errorType, len(raw), contentHash
	trace.AsOf, trace.Warnings = envelope.AsOf, append([]string(nil), envelope.Warnings...)
	conversion.Block.ContentHash = contentHash
	return ExecuteResult{Descriptor: descriptor, Envelope: envelope, Trace: trace, Value: value, Raw: append(json.RawMessage(nil), raw...), Evidence: conversion.Evidence, ContextBlock: conversion.Block}, nil
}

func (runtime *Runtime) errorResult(descriptor ToolDescriptor, trace Trace, startedAt time.Time, toolErr *Error) ExecuteResult {
	duration := runtime.now().UTC().Sub(startedAt)
	envelope := ToolResultEnvelope{Source: descriptor.CanonicalName, DurationMs: duration.Milliseconds(), CacheHit: false, Partial: false, Warnings: []string{security.RedactText(toolErr.Error())}, ErrorType: toolErr.Type}
	trace.DurationMs, trace.ErrorType = envelope.DurationMs, toolErr.Type
	trace.Warnings = append([]string(nil), envelope.Warnings...)
	return ExecuteResult{Descriptor: descriptor, Envelope: envelope, Trace: trace, ToolError: toolErr}
}

func newTrace(descriptor ToolDescriptor, argumentsHash string, request ExecuteRequest) Trace {
	return Trace{
		CanonicalName: descriptor.CanonicalName, SourceType: descriptor.SourceType, Risk: descriptor.Risk, Idempotent: descriptor.Idempotent,
		TimeoutMs: descriptor.TimeoutMs, CacheTTLms: descriptor.CachePolicy.TTLms, ArgumentsHash: argumentsHash, CallIndex: request.CallIndex, CallBudget: request.CallBudget,
		ProviderRef: descriptor.ProviderRef, ProtocolVersion: descriptor.ProtocolVersion, CatalogHash: descriptor.CatalogHash, SchemaHash: descriptor.SchemaHash,
	}
}

func normalizeArguments(raw json.RawMessage) json.RawMessage {
	if len(bytes.TrimSpace(raw)) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return json.RawMessage(`{}`)
	}
	return append(json.RawMessage(nil), raw...)
}

func canonicalHash(raw json.RawMessage) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "", err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(canonical)
	return hex.EncodeToString(hash[:]), nil
}

func effectiveResultLimit(values ...int) int {
	limit := 0
	for _, value := range values {
		if value > 0 && (limit == 0 || value < limit) {
			limit = value
		}
	}
	return limit
}

func trimResult(raw []byte, maxBytes int) (any, []string, bool) {
	if maxBytes <= 0 || len(raw) <= maxBytes {
		var value any
		_ = json.Unmarshal(raw, &value)
		return value, nil, false
	}
	previewLimit := maxBytes / 2
	if previewLimit < 16 {
		previewLimit = 16
	}
	preview := safePrefix(string(raw), previewLimit)
	return map[string]any{"truncated": true, "preview": preview}, []string{fmt.Sprintf("result trimmed from %d bytes to fit %d-byte policy", len(raw), maxBytes)}, true
}

func safePrefix(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	end := maxBytes
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end]
}

func restoreValue(selected tools.Tool, raw json.RawMessage) (any, error) {
	if codec, ok := selected.(tools.CheckpointCodec); ok {
		return codec.RestoreCheckpoint(raw)
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}
