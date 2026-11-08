package feature

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// 01-4: scan all executable Go sources, not only the seven original
// filenames. AST ignores comments and catches renamed imports, multiline
// statements and unanticipated new packages.
// Only the explicit v19 CLI migration file, the archival model definition
// and guarded main sync-db registration are permitted to mention legacy
// orders. Strategy research scripts are intentionally not production code.
var retiredOrderSQLPattern = regexp.MustCompile("(?i)\\b(?:from|into|update|join|delete\\s+from)\\s+[\\x22\\x60]?order[\\x22\\x60]?(?:\\W|$)")

func TestStage014OldSQLTableDetectionIncludesBackticks(t *testing.T) {
	for _, query := range []string{
		"SELECT * FROM `order` WHERE id=1",
		"SELECT * FROM order WHERE id=1",
		"DELETE FROM `order` WHERE id=1",
		"UPDATE order SET side='close'",
		"INSERT INTO `order` (id) VALUES (1)",
		`SELECT * FROM "order"`,
	} {
		if !retiredOrderSQLPattern.MatchString(query) {
			t.Fatalf("legacy table SQL evaded guard: %s", query)
		}
	}
	for _, query := range []string{
		"SELECT * FROM futures_orders", "SELECT * FROM futures_managed_orders",
		"SELECT * FROM order_history", "SELECT * FROM order_line",
	} {
		if retiredOrderSQLPattern.MatchString(query) {
			t.Fatalf("non-legacy table falsely flagged: %s", query)
		}
	}
}

func TestStage014NoLegacyHistoryRuntimeReferences(t *testing.T) {
	root := ".."
	exception := map[string]bool{
		"models/tableStruct.go":              true,
		"command/account_scope_migration.go": true,
	}
	excluded := map[string]bool{
		".git": true, "vendor": true, "node_modules": true,
		"strategy_templates": true, "temp_strategy": true,
		".local": true, "static": true, "dist": true,
	}
	queryLegacy := retiredOrderSQLPattern
	scanned := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if excluded[entry.Name()] {
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
		rel = filepath.ToSlash(rel)
		if exception[rel] {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, src, 0)
		if err != nil {
			return err
		}
		// AST selector identity must track import aliases, not only
		// spellings such as models.Order.
		modelAliases := map[string]bool{}
		for _, spec := range file.Imports {
			path, e := strconv.Unquote(spec.Path.Value)
			if e != nil || path != "go_binance_futures/models" {
				continue
			}
			name := "models"
			if spec.Name != nil {
				name = spec.Name.Name
			}
			if name == "." || name == "_" {
				return fmt.Errorf("%s uses unsafe unqualified models import", rel)
			}
			modelAliases[name] = true
		}
		scanned++
		var violation error
		ast.Inspect(file, func(n ast.Node) bool {
			if violation != nil {
				return false
			}
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if pkg, ok := x.X.(*ast.Ident); ok && modelAliases[pkg.Name] && x.Sel.Name == "Order" {
					if rel == "main.go" {
						return true
					} // exact CLI exception audited separately
					violation = fmt.Errorf("%s uses retired models.Order", rel)
					return false
				}
			case *ast.CallExpr:
				if selector, ok := x.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "QueryTable" && len(x.Args) > 0 {
					if lit, ok := x.Args[0].(*ast.BasicLit); ok {
						str, e := strconv.Unquote(lit.Value)
						if e == nil && strings.EqualFold(strings.TrimSpace(str), "order") {
							violation = fmt.Errorf("%s queries legacy order ORM table", rel)
							return false
						}
					}
				}
			case *ast.BasicLit:
				if x.Kind == token.STRING {
					str, e := strconv.Unquote(x.Value)
					if e == nil && queryLegacy.MatchString(str) {
						violation = fmt.Errorf("%s contains legacy order SQL: %q", rel, str)
						return false
					}
				}
			case *ast.Ident:
				if x.Name == "insertOpenOrder" || x.Name == "insertCloseOrder" || x.Name == "UpdateOrderStatus" ||
					x.Name == "RecordOpen" || x.Name == "RecordClose" {
					violation = fmt.Errorf("%s refers to retired historical function %s", rel, x.Name)
					return false
				}
			}
			return true
		})
		return violation
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanned < 80 {
		t.Fatalf("repository guard unexpectedly scanned only %d production files", scanned)
	}
	if _, err := os.Stat("order.go"); !os.IsNotExist(err) {
		t.Fatalf("retired task still exists: %v", err)
	}
}

// AST gate for the only permitted runtime reference to models.Order:
// registration strictly nested inside an exact sync-db invocation. Go fmt
// output is only used for checking the guard's expression, not comments.
func TestStage014LegacyOrderORMRegistrationOnlyForSyncDB(t *testing.T) {
	source, err := os.ReadFile("../main.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "../main.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	registered := 0
	scoped := 0
	orderModelReferences := 0
	ast.Inspect(file, func(n ast.Node) bool {
		if selector, ok := n.(*ast.SelectorExpr); ok && selector.Sel.Name == "Order" {
			if pkg, ok := selector.X.(*ast.Ident); ok && pkg.Name == "models" {
				orderModelReferences++
			}
		}
		if node, ok := n.(*ast.CallExpr); ok {
			// RegisterModel(new(models.Order)) is the archived CLI-only
			// registration, not any other versioned model.
			var rendered bytes.Buffer
			_ = printer.Fprint(&rendered, fset, node)
			if rendered.String() == "orm.RegisterModel(new(models.Order))" {
				registered++
			}
		}
		if stmt, ok := n.(*ast.IfStmt); ok {
			var condition bytes.Buffer
			_ = printer.Fprint(&condition, fset, stmt.Cond)
			if condition.String() != "isSyncDatabaseCommand(os.Args[1:])" {
				return true
			}
			ast.Inspect(stmt.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				var rendered bytes.Buffer
				_ = printer.Fprint(&rendered, fset, call)
				if rendered.String() == "orm.RegisterModel(new(models.Order))" {
					scoped++
				}
				return true
			})
		}
		return true
	})
	if registered != 1 || scoped != 1 || orderModelReferences != 1 {
		t.Fatalf("archival Order reference must appear only in exact sync-db CLI: registered=%d scoped=%d references=%d", registered, scoped, orderModelReferences)
	}
}

// Exempt CLI code is NOT a broad wildcard: all archived Order SQL must stay
// in the historical v19 migration, never be added to a new command handler.
func TestStage014LegacyCommandExceptionIsRestricted(t *testing.T) {
	entries, err := os.ReadDir("../command")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") ||
			entry.Name() == "account_scope_migration.go" {
			continue
		}
		b, err := os.ReadFile(filepath.Join("../command", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(token.NewFileSet(), entry.Name(), b, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			if lit, ok := node.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				v, e := strconv.Unquote(lit.Value)
				if e == nil && retiredOrderSQLPattern.MatchString(v) {
					t.Errorf("non-migration command %s embeds legacy order SQL", entry.Name())
					return false
				}
			}
			return true
		})
	}
}
