package llm

import (
	"context"
	"strings"
	"sync"
	"testing"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

var llmStoreTestOnce sync.Once
var llmStoreTestErr error

func setupLLMStoreTest(t *testing.T) Store {
	t.Helper()
	llmStoreTestOnce.Do(func() {
		llmStoreTestErr = orm.RegisterDriver("sqlite3", orm.DRSqlite)
		if llmStoreTestErr != nil {
			return
		}
		orm.RegisterModel(new(models.LLMConfig))
		llmStoreTestErr = orm.RegisterDataBase("default", "sqlite3", "file:llm_config_test?mode=memory&cache=shared")
		if llmStoreTestErr == nil {
			llmStoreTestErr = orm.RunSyncdb("default", true, false)
		}
	})
	if llmStoreTestErr != nil {
		t.Fatal(llmStoreTestErr)
	}
	return Store{}
}
func TestStoreMasksKeyAndSwitchesActiveConfig(t *testing.T) {
	store := setupLLMStoreTest(t)
	ctx := context.Background()
	first, err := store.Create(ctx, ConfigInput{
		Name: "deepseek-main", Provider: "deepseek", APIKey: "secret-deepseek", Model: "deepseek-test",
		TimeoutSeconds: 60, Temperature: 0.2, Enabled: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !first.HasAPIKey || strings.Contains(first.APIKeyMasked, "secret-deepseek") {
		t.Fatalf("api key leaked in public config: %+v", first)
	}
	second, err := store.Create(ctx, ConfigInput{
		Name: "ollama-local", Provider: "ollama", Model: "qwen3:8b",
		TimeoutSeconds: 30, Temperature: 0.1, Enabled: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Enabled != 1 {
		t.Fatalf("second config not enabled: %+v", second)
	}
	active, err := store.ActiveConfig()
	if err != nil {
		t.Fatal(err)
	}
	if active.ID != second.ID {
		t.Fatalf("active config identity lost: got %d want %d", active.ID, second.ID)
	}
	client, err := NewClient(active)
	if err != nil {
		t.Fatal(err)
	}
	if ConfigID(client) != second.ID {
		t.Fatalf("client config identity = %d, want %d", ConfigID(client), second.ID)
	}
	items, err := store.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.ID == first.ID && item.Enabled != 0 {
			t.Fatalf("previous active config was not disabled: %+v", item)
		}
	}
}
func TestStoreUpdatePreservesBlankAPIKey(t *testing.T) {
	store := setupLLMStoreTest(t)
	ctx := context.Background()
	item, err := store.Create(ctx, ConfigInput{
		Name: "gemini-main", Provider: "gemini", APIKey: "gemini-secret", Model: "gemini-test",
		TimeoutSeconds: 60, Temperature: 0.3, Enabled: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.Update(ctx, item.ID, ConfigInput{
		Name: "gemini-main", Provider: "gemini", APIKey: "", Model: "gemini-test-2",
		TimeoutSeconds: 60, Temperature: 0.4, Enabled: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !updated.HasAPIKey {
		t.Fatalf("blank update removed existing key: %+v", updated)
	}
	row, err := store.Get(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if row.APIKey != "gemini-secret" || row.Model != "gemini-test-2" {
		t.Fatalf("unexpected stored config: %+v", row)
	}
}
func TestSupportedHTTPProviderConfigs(t *testing.T) {
	providers := []string{"openai", "anthropic", "deepseek", "glm", "moonshot", "ollama", "gemini", "openai_compatible"}
	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			input := ConfigInput{Name: provider, Provider: provider, Model: "test-model", APIKey: "test-key", TimeoutSeconds: 30, Temperature: 0.2}
			if provider == "ollama" || provider == "openai_compatible" {
				input.APIKey = ""
			}
			if provider == "openai_compatible" {
				input.APIURL = "http://127.0.0.1:9999/v1/chat/completions"
			}
			cfg, err := BuildConfig(input, "")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := NewClient(cfg); err != nil {
				t.Fatalf("provider %s is not supported: %v", provider, err)
			}
		})
	}
}

func TestDeletingActiveConfigLeavesRuntimeUnconfigured(t *testing.T) {
	store := setupLLMStoreTest(t)
	ctx := context.Background()
	item, err := store.Create(ctx, ConfigInput{Name: "delete-active", Provider: "ollama", Model: "qwen", TimeoutSeconds: 30, Temperature: 0.2, Enabled: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ActiveConfig(); err == nil || !strings.Contains(err.Error(), "no enabled LLM configuration") {
		t.Fatalf("expected unconfigured runtime, got %v", err)
	}
}

func TestNewFromConfigIDUsesRequestedConfiguration(t *testing.T) {
	store := setupLLMStoreTest(t)
	ctx := context.Background()
	item, err := store.Create(ctx, ConfigInput{
		Name: "resume-frozen", Provider: "ollama", Model: "qwen-resume", TimeoutSeconds: 30, Temperature: 0.2, Enabled: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewFromConfigID(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ConfigID(client) != item.ID {
		t.Fatalf("client config id = %d, want %d", ConfigID(client), item.ID)
	}
}
