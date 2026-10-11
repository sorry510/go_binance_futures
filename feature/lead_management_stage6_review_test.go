package feature

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// Only this reviewed Stage 6 controller is allowed to import the Lead
// account service from outside its own package. This works even when an
// attacker/accidental change renames the import alias or launches the live
// supervisor via an intermediate wrapper from a different directory.
const stage6LeadImport = "go_binance_futures/service/leadaccount"

func stage6AuditLeadFile(filePath, relative string) error {
	node, err := parser.ParseFile(token.NewFileSet(), filePath, nil, 0)
	if err != nil {
		return err
	}
	aliases := map[string]bool{}
	for _, imp := range node.Imports {
		path, e := strconv.Unquote(imp.Path.Value)
		if e != nil {
			return e
		}
		if path != stage6LeadImport {
			continue
		}
		if relative != "controllers/lead_account.go" {
			return fmt.Errorf("unreviewed production Lead import in %s", relative)
		}
		name := "leadaccount"
		if imp.Name != nil {
			name = imp.Name.Name
		}
		if name == "." || name == "_" {
			return fmt.Errorf("opaque Lead import forbidden: %s", relative)
		}
		aliases[name] = true
	}
	// The controller is the sole approved management ingress. Every direct
	// exported Lead symbol must be reviewed and allowlisted. This catches
	// supervisor constructor calls even in aliased imports.
	allowed := map[string]bool{
		"NewEncryptedFileStoreFromEnv": true, "NewManager": true,
		"LeadAuditStoreFromEnv": true, "LeadConfigStoreFromEnv": true,
		"DefaultLeadSettings": true, "ValidateLeadSettings": true,
		"ListLeadManaged": true, "LeadUsageSnapshot": true,
		"Credentials": true, "LeadSettings": true,
		"EncryptedFileStore": true, "Manager": true,
		"MasterKeyEnv": true, "StorePathEnv": true,
		"LatchLeadEmergencyPause": true, "LeadEmergencyPaused": true,
	}
	bannedCalls := map[string]bool{
		"NewLeadSupervisor": true, "NewLeadUserStream": true,
		"NewLeadExecutionAdapter": true, "NewLeadRecoveryAuditor": true,
		"ExecuteOpen": true, "ExecuteManagedClose": true,
		"Submit": true, "Cancel": true, "Run": true, "Reconcile": true,
		"DialLeadUserStream": true, "GetListenKey": true,
		"completeAuthoritativeRecovery": true,
	}
	var violation error
	ast.Inspect(node, func(n ast.Node) bool {
		if violation != nil {
			return false
		}
		sel, ok := n.(*ast.SelectorExpr)
		if ok {
			if ident, isIdent := sel.X.(*ast.Ident); isIdent && aliases[ident.Name] && !allowed[sel.Sel.Name] {
				violation = fmt.Errorf("unreviewed Lead symbol %s in %s", sel.Sel.Name, relative)
				return false
			}
		}
		call, ok := n.(*ast.CallExpr)
		if !ok || relative != "controllers/lead_account.go" {
			return true
		}
		var method string
		switch fn := call.Fun.(type) {
		case *ast.SelectorExpr:
			method = fn.Sel.Name
		case *ast.Ident:
			method = fn.Name
		}
		if bannedCalls[method] {
			violation = fmt.Errorf("Lead management ingress called prohibited %s", method)
			return false
		}
		return true
	})
	return violation
}

func TestStage6ReviewProductionLeadIngressImportAllowlist(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("missing test file location")
	}
	root := filepath.Dir(filepath.Dir(here))
	seen := 0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "vendor", "node_modules", "static", "dist", "testdata",
				"strategy_templates", ".agents":
				return filepath.SkipDir
			}
			// service/leadaccount is the reviewed implementation itself, not an
			// external production entrypoint. All other production Go is scanned.
			if path == filepath.Join(root, "service", "leadaccount") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		seen++
		return stage6AuditLeadFile(path, filepath.ToSlash(rel))
	})
	if err != nil {
		t.Fatal(err)
	}
	if seen < 40 {
		t.Fatalf("incomplete source scan: %d Go files", seen)
	}
}

func TestStage6ReviewLeadIngressGuardCatchesAliasAndLiveStart(t *testing.T) {
	cases := []struct {
		name, relative, source string
		ok                     bool
	}{
		{"safe_controller", "controllers/lead_account.go", `package controllers
    import account "go_binance_futures/service/leadaccount"
    func f(){account.LatchLeadEmergencyPause()}`, true},
		{"other_package_alias", "main.go", `package main
    import account "go_binance_futures/service/leadaccount"
    func main(){_ = account.NewLeadSupervisor}`, false},
		{"controller_supervisor", "controllers/lead_account.go", `package controllers
    import l "go_binance_futures/service/leadaccount"
    func f(){_ = l.NewLeadSupervisor}`, false},
		{"controller_run", "controllers/lead_account.go", `package controllers
    func f(v interface{Run() error}) { _ = v.Run() }`, false},
		{"controller_dot_import", "controllers/lead_account.go", `package controllers
    import . "go_binance_futures/service/leadaccount"
    func f(){_ = NewLeadSupervisor}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "sample.go")
			if err := os.WriteFile(path, []byte(tc.source), 0600); err != nil {
				t.Fatal(err)
			}
			err := stage6AuditLeadFile(path, tc.relative)
			if (err == nil) != tc.ok {
				t.Fatalf("guard outcome err=%v expected allowed=%v", err, tc.ok)
			}
		})
	}
}
