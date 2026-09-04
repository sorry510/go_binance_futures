package portableskill

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
	_ "github.com/mattn/go-sqlite3"
	"go_binance_futures/models"
)

func TestParseStandardFixtures(t *testing.T) {
	pkg, err := ParsePackage("testdata/valid-minimal")
	if err != nil {
		t.Fatal(err)
	}
	if pkg.Frontmatter.Name != "valid-minimal" || pkg.Frontmatter.Metadata["version"] != "1.0.0" || len(pkg.RequestedTools) != 1 || len(pkg.PackageHash) != 64 {
		t.Fatalf("unexpected package: %+v", pkg)
	}
	withResources, err := ParsePackage("testdata/valid-resources")
	if err != nil {
		t.Fatal(err)
	}
	foundWarning := false
	for _, d := range withResources.Diagnostics {
		if d.Code == "script_execution_disabled" {
			foundWarning = true
		}
	}
	if !foundWarning {
		t.Fatal("scripts/ must be indexed with execution-disabled warning")
	}
}
func TestStrictFrontmatterRejectsPrivateTopLevelFields(t *testing.T) {
	for _, field := range []string{"version", "trusted", "unknown"} {
		raw := []byte("---\nname: test-skill\ndescription: valid description\n" + field + ": x\n---\nbody\n")
		_, _, _, err := parseSkillMD(raw)
		if err == nil {
			t.Fatalf("field %s must be rejected", field)
		}
		if (field == "version" || field == "trusted") && !strings.Contains(err.Error(), "metadata."+field) {
			t.Fatalf("missing migration hint: %v", err)
		}
	}
}
func TestParseRejectsNameDirectoryMismatch(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: another-name\ndescription: valid description\n---\nbody\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ParsePackage(dir); err == nil || !strings.Contains(err.Error(), "parent directory") {
		t.Fatalf("expected directory mismatch, got %v", err)
	}
}

func zipBytes(t *testing.T, entries map[string]string, symlink string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, body := range entries {
		var f *zip.FileHeader
		if name == symlink {
			f = &zip.FileHeader{Name: name, Method: zip.Store}
			f.SetMode(os.ModeSymlink | 0777)
		} else {
			f = &zip.FileHeader{Name: name, Method: zip.Deflate}
			f.SetMode(0600)
		}
		out, err := w.CreateHeader(f)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := out.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
func TestZIPSecurityRejectsTraversalAndSymlink(t *testing.T) {
	imp := Importer{}
	cases := []struct {
		name string
		data []byte
	}{
		{"traversal", zipBytes(t, map[string]string{"../evil": "x"}, "")},
		{"windows-drive", zipBytes(t, map[string]string{"C:/portable/SKILL.md": "---\nname: portable\ndescription: valid portable skill\n---\nbody\n"}, "")},
		{"symlink", zipBytes(t, map[string]string{"portable/SKILL.md": "---\nname: portable\ndescription: valid portable skill\n---\nbody\n", "portable/link": "target"}, "portable/link")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := imp.ImportFile(context.Background(), "skill.zip", bytes.NewReader(tc.data), int64(len(tc.data)), false); err == nil {
				t.Fatal("unsafe zip must fail")
			}
		})
	}
}
func TestZIPSecurityRejectsTooManyEntries(t *testing.T) {
	entries := map[string]string{"many/SKILL.md": "---\nname: many\ndescription: valid many skill\n---\nbody\n"}
	for i := 0; i < maxFiles; i++ {
		entries[filepath.ToSlash(filepath.Join("many", "references", strings.Repeat("x", 4)+string(rune('a'+i%26))+fmt.Sprint(i)+".md"))] = "x"
	}
	data := zipBytes(t, entries, "")
	if _, err := (Importer{}).ImportFile(context.Background(), "many.zip", bytes.NewReader(data), int64(len(data)), false); err == nil || !strings.Contains(err.Error(), "entries") {
		t.Fatalf("expected entry limit, got %v", err)
	}
}

func setupPortableDB(t *testing.T) Store {
	t.Helper()
	alias := "default"
	_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
	orm.RegisterModel(new(models.AgentSkill), new(models.AgentSkillVersion), new(models.AgentSkillPermission))
	if err := orm.RegisterDataBase(alias, "sqlite3", "file:portable_test?mode=memory&cache=shared"); err != nil {
		t.Fatal(err)
	}
	if err := orm.RunSyncdb(alias, true, false); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(t.TempDir(), "skills")
	_ = config.Set("agent_skills::data_dir", data)
	return Store{Alias: alias}
}
func TestInstallRevisionRollbackAndAllowedToolsRequireReview(t *testing.T) {
	store := setupPortableDB(t)
	src := filepath.Join(t.TempDir(), "portable-test")
	if err := os.MkdirAll(src, 0700); err != nil {
		t.Fatal(err)
	}
	write := func(version, body string) {
		content := "---\nname: portable-test\ndescription: Portable test skill for revision checks.\nmetadata:\n  version: \"" + version + "\"\nallowed-tools: get_market_condition\n---\n" + body + "\n"
		if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("1.0.0", "first")
	pkg, err := ParsePackage(src)
	if err != nil {
		t.Fatal(err)
	}
	metadata, _ := json.Marshal(pkg.Frontmatter.Metadata)
	requested, _ := json.Marshal(pkg.RequestedTools)
	v := models.AgentSkillVersion{Name: pkg.Frontmatter.Name, PackageHash: pkg.PackageHash, Version: "1.0.0", Description: pkg.Frontmatter.Description, MetadataJSON: string(metadata), RequestedToolsJSON: string(requested), ValidationStatus: ValidationValid, PackagePath: "portable-test/" + pkg.PackageHash}
	if err := copyPackage(src, filepath.Join(DataDir(), filepath.FromSlash(v.PackagePath))); err != nil {
		t.Fatal(err)
	}
	r1, err := store.Install(context.Background(), v, pkg.RequestedTools, true)
	if err != nil {
		t.Fatal(err)
	}
	perms, _ := store.Permissions(context.Background(), r1.Version.ID)
	if len(perms) != 1 || perms[0].Granted != 0 {
		t.Fatalf("allowed-tools must not auto grant: %+v", perms)
	}
	write("2.0.0", "second")
	pkg2, _ := ParsePackage(src)
	v.PackageHash = pkg2.PackageHash
	v.Version = "2.0.0"
	v.PackagePath = "portable-test/" + pkg2.PackageHash
	if err := copyPackage(src, filepath.Join(DataDir(), filepath.FromSlash(v.PackagePath))); err != nil {
		t.Fatal(err)
	}
	r2, err := store.Install(context.Background(), v, pkg2.RequestedTools, true)
	if err != nil {
		t.Fatal(err)
	}
	if r2.Skill.ActiveVersionID != r2.Version.ID {
		t.Fatal("new revision not activated")
	}
	rolled, err := store.Activate(context.Background(), r1.Version.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rolled.ActiveVersionID != r1.Version.ID {
		t.Fatal("rollback failed")
	}
}
