package toolruntime

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v5"
)

type schemaValidator struct {
	mu    sync.Mutex
	items map[string]*jsonschema.Schema
}

func newSchemaValidator() *schemaValidator {
	return &schemaValidator{items: map[string]*jsonschema.Schema{}}
}

func (validator *schemaValidator) validate(schemaRaw, payload json.RawMessage) error {
	if len(bytes.TrimSpace(schemaRaw)) == 0 {
		return nil
	}
	schema, err := validator.compile(schemaRaw)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("decode JSON payload: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("JSON payload contains multiple values")
		}
		return fmt.Errorf("decode trailing JSON: %w", err)
	}
	if err := schema.Validate(value); err != nil {
		return err
	}
	return nil
}

func (validator *schemaValidator) compile(raw json.RawMessage) (*jsonschema.Schema, error) {
	hash := sha256.Sum256(raw)
	key := hex.EncodeToString(hash[:])
	validator.mu.Lock()
	defer validator.mu.Unlock()
	if schema := validator.items[key]; schema != nil {
		return schema, nil
	}
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft2020
	uri := "mem://tool-schema/" + key + ".json"
	if err := compiler.AddResource(uri, strings.NewReader(string(raw))); err != nil {
		return nil, fmt.Errorf("load JSON schema: %w", err)
	}
	schema, err := compiler.Compile(uri)
	if err != nil {
		return nil, fmt.Errorf("compile JSON schema: %w", err)
	}
	validator.items[key] = schema
	return schema, nil
}
