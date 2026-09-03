package toolruntime

import (
	"testing"
	"time"

	"go_binance_futures/agent/permission"
	"go_binance_futures/agent/tools"
)

func TestCatalogHashIsDeterministicAndTracksMetadata(t *testing.T) {
	makeRuntime := func(timeout time.Duration) *Runtime {
		registry := tools.NewRegistry()
		_ = registry.Register(tools.Func{ToolName: "a", ToolRisk: permission.RiskRead, ToolMetadata: tools.Metadata{Idempotent: true, Timeout: timeout}})
		_ = registry.Register(tools.Func{ToolName: "b", ToolRisk: permission.RiskRead, ToolMetadata: tools.Metadata{Idempotent: true}})
		runtime, err := New(Config{Registry: registry, Policy: permission.AllowReadOnly()})
		if err != nil {
			t.Fatal(err)
		}
		return runtime
	}
	first := makeRuntime(time.Second)
	left := first.CatalogHash([]string{"b", "a", "a"})
	right := first.CatalogHash([]string{"a", "b"})
	if len(left) != 64 || left != right {
		t.Fatalf("catalog hash is not deterministic: %q %q", left, right)
	}
	changed := makeRuntime(2 * time.Second).CatalogHash([]string{"a", "b"})
	if changed == left {
		t.Fatal("catalog metadata change did not change hash")
	}
	missing := first.CatalogHash([]string{"a", "missing"})
	if missing == left {
		t.Fatal("missing tool identity did not change hash")
	}
}
