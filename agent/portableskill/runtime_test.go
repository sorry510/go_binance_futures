package portableskill

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/beego/beego/v2/core/config"
	"go_binance_futures/agent/permission"
	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/task"
	agenttools "go_binance_futures/agent/tools"
	"go_binance_futures/llm"
	"go_binance_futures/models"
)

type portableFakeClient struct {
	mu       sync.Mutex
	items    []*llm.Response
	requests []llm.Request
}

func (c *portableFakeClient) Provider() llm.Provider { return llm.ProviderOpenAICompatible }
func (c *portableFakeClient) Generate(_ context.Context, req llm.Request) (*llm.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requests = append(c.requests, req)
	idx := len(c.requests) - 1
	if idx >= len(c.items) {
		return nil, fmt.Errorf("unexpected LLM request %d", idx+1)
	}
	return c.items[idx], nil
}

func TestPortableSkillRunsThroughUnifiedRuntimeAndReadsResource(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "agent-skills")
	oldDataDir, _ := config.String("agent_skills::data_dir")
	if err := config.Set("agent_skills::data_dir", dataDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = config.Set("agent_skills::data_dir", oldDataDir) })

	source := filepath.Join("testdata", "valid-resources")
	pkg, err := ParsePackage(source)
	if err != nil {
		t.Fatal(err)
	}
	packagePath := filepath.ToSlash(filepath.Join(pkg.Frontmatter.Name, pkg.PackageHash))
	if err := copyPackage(source, filepath.Join(dataDir, filepath.FromSlash(packagePath))); err != nil {
		t.Fatal(err)
	}
	version := models.AgentSkillVersion{
		ID: 41, SkillID: 9, Name: pkg.Frontmatter.Name, PackageHash: pkg.PackageHash,
		Version: "2.0.0", Description: pkg.Frontmatter.Description, PackagePath: packagePath,
		ValidationStatus: ValidationValid,
	}
	adapter, err := LoadAdapter(version)
	if err != nil {
		t.Fatal(err)
	}
	if adapter.PackageHash() != pkg.PackageHash {
		t.Fatalf("package identity drift: %s", adapter.PackageHash())
	}

	skills := skill.NewRegistry()
	if err := skills.Register(adapter); err != nil {
		t.Fatal(err)
	}
	tools := agenttools.NewRegistry()
	if err := tools.Register(adapter.ResourceTool()); err != nil {
		t.Fatal(err)
	}
	client := &portableFakeClient{items: []*llm.Response{
		{Content: `{"action":"tool","tool":"skill.valid-resources.read-resource","arguments":{"path":"references/POLICY.md"}}`, Usage: llm.Usage{TotalTokens: 100}},
		{Content: `{"action":"final","summary":"done","result":{"policy":"reference policy"}}`, Usage: llm.Usage{TotalTokens: 100}},
	}}
	store := task.NewMemoryStore()
	runner, err := agentruntime.NewRunner(agentruntime.Config{
		Client: client, Skills: skills, Tools: tools, Tasks: store, Policy: permission.AllowReadOnly(),
		Timeout: 3 * time.Second, Retry: agentruntime.RetryPolicy{MaxAttempts: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := runner.Run(context.Background(), agentruntime.Request{Skill: adapter.Name(), Input: `{"topic":"policy"}`})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || !strings.Contains(string(result.Raw), "reference policy") {
		t.Fatalf("unexpected portable result: %+v", result)
	}
	items, err := store.List(context.Background(), task.ListOptions{})
	if err != nil || len(items.List) != 1 {
		t.Fatalf("load task: items=%+v err=%v", items, err)
	}
	item := items.List[0]
	if item.SkillSource != SkillTypePortable || item.SkillPackageHash != pkg.PackageHash || item.SkillSourceVersion != pkg.PackageHash {
		t.Fatalf("portable task identity not frozen: source=%q sourceVersion=%q hash=%q", item.SkillSource, item.SkillSourceVersion, item.SkillPackageHash)
	}
	if len(client.requests) != 2 {
		t.Fatalf("expected two LLM rounds, got %d", len(client.requests))
	}
	second := client.requests[1]
	encoded, _ := json.Marshal(second.Messages)
	if !strings.Contains(string(encoded), "reference policy") {
		t.Fatalf("resource tool result was not returned to the LLM: %s", string(encoded))
	}
}

func TestPortableResourceToolCannotEscapePackageOrExecuteScript(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "agent-skills")
	oldDataDir, _ := config.String("agent_skills::data_dir")
	_ = config.Set("agent_skills::data_dir", dataDir)
	t.Cleanup(func() { _ = config.Set("agent_skills::data_dir", oldDataDir) })
	pkg, err := ParsePackage(filepath.Join("testdata", "valid-resources"))
	if err != nil {
		t.Fatal(err)
	}
	packagePath := filepath.ToSlash(filepath.Join(pkg.Frontmatter.Name, pkg.PackageHash))
	if err := copyPackage(filepath.Join("testdata", "valid-resources"), filepath.Join(dataDir, filepath.FromSlash(packagePath))); err != nil {
		t.Fatal(err)
	}
	adapter, err := LoadAdapter(models.AgentSkillVersion{ID: 1, Name: pkg.Frontmatter.Name, PackageHash: pkg.PackageHash, PackagePath: packagePath})
	if err != nil {
		t.Fatal(err)
	}
	tool := adapter.ResourceTool()
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"../outside.txt"}`)); err == nil {
		t.Fatal("resource tool must reject path traversal")
	}
	value, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"scripts/check.sh"}`))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(value)
	if !strings.Contains(string(raw), "echo never-execute") {
		t.Fatalf("script must only be returned as text: %s", raw)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "executed-marker")); !os.IsNotExist(err) {
		t.Fatal("portable script unexpectedly executed")
	}
}
