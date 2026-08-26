package bootstrap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/beego/beego/v2/core/config"
)

// TestConfigReadableFromPackageDirectory is the regression test for the panic
// this package exists to prevent. "go test" runs with the working directory
// set to this package directory, which is exactly the situation where beego's
// relative "conf/app.conf" lookup fails.
func TestConfigReadableFromPackageDirectory(t *testing.T) {
	appname, err := config.String("appname")
	if err != nil {
		t.Fatalf("read appname from global config: %v", err)
	}
	if appname == "" {
		t.Fatal("appname is empty, the global config instance was not loaded")
	}
}

func TestProjectRootContainsConfDirectory(t *testing.T) {
	root := ProjectRoot()
	if root == "" {
		t.Fatal("ProjectRoot returned an empty path")
	}
	info, err := os.Stat(filepath.Join(root, "conf"))
	if err != nil {
		t.Fatalf("conf directory not found under project root %q: %v", root, err)
	}
	if !info.IsDir() {
		t.Fatalf("%q is not a directory", filepath.Join(root, "conf"))
	}
}

func TestFindConfigFileWalksUpFromWorkingDirectory(t *testing.T) {
	path, ok := findConfigFile()
	if !ok {
		t.Fatal("findConfigFile did not locate a config file")
	}
	if !filepath.IsAbs(path) {
		t.Fatalf("expected an absolute path, got %q", path)
	}
	base := filepath.Base(path)
	if base != "app.conf" && base != "app.conf.example" {
		t.Fatalf("unexpected config file %q", base)
	}
}

func TestFindConfigFileReturnsFalseOutsideProject(t *testing.T) {
	dir := t.TempDir()
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir to temp dir: %v", err)
	}
	if path, ok := findConfigFile(); ok {
		t.Fatalf("expected no config file outside the project, got %q", path)
	}
}
