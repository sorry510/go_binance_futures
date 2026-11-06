package strategy

import (
	"strings"
	"testing"
)

func TestStrategySnapshotHashUsesStoredSnapshotAndTrimsOuterWhitespace(t *testing.T) {
	left := StrategySnapshotHash(` {"b":2,"a":1} `, ` [{"name":"n","type":"long"}] `)
	right := StrategySnapshotHash(`{"b":2,"a":1}`, `[{"name":"n","type":"long"}]`)
	if left == "" || left != right {
		t.Fatalf("snapshot hash should ignore only outer whitespace: left=%q right=%q", left, right)
	}
	if left == StrategySnapshotHash(`{"a":1,"b":2}`, `[{"name":"n","type":"long"}]`) {
		t.Fatal("different stored JSON snapshots must have different version hashes")
	}
}

func TestRuleHashIgnoresOuterWhitespace(t *testing.T) {
	if RuleHash("  ROI > 2  ") != RuleHash("ROI > 2") {
		t.Fatal("rule hash should ignore outer whitespace")
	}
}

func TestResolveRuleIdentityFindsNameAndType(t *testing.T) {
	strategyJSON := `[
		{"name":"same code wrong side","enable":true,"code":"Price > 1","type":"short"},
		{"name":"long entry","enable":true,"code":"Price > 1","type":"long"}
	]`
	identity := ResolveRuleIdentity(strategyJSON, "Price > 1", "long")
	if identity.Name != "long entry" || identity.Type != "long" || identity.Hash == "" {
		t.Fatalf("unexpected rule identity: %+v", identity)
	}
}

func TestPersistedStrategyNameTruncatesUnicodeSafely(t *testing.T) {
	for _, tc := range []struct {
		name string
		want string
	}{
		{"  name  ", "name"},
		{strings.Repeat("a", 128), strings.Repeat("a", 128)},
		{strings.Repeat("a", 150), strings.Repeat("a", 128)},
		{strings.Repeat("策略", 70), strings.Repeat("策略", 64)},
	} {
		if got := PersistedStrategyName(tc.name); got != tc.want {
			t.Fatalf("PersistedStrategyName input_chars=%d got=%q want=%q", len([]rune(tc.name)), got, tc.want)
		}
	}
}
