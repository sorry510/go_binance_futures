// Package bootstrap loads the application configuration before any other
// package reads it.
//
// beego's core/config package initialises its global instance from the
// relative path "conf/app.conf" (see core/config/ini.go). That only resolves
// when the process runs from the project root. Running "go test ./..." starts
// each test binary in its own package directory, so the lookup fails and the
// global instance ends up holding a nil container. Every later config read
// then dereferences a nil pointer and panics during package initialisation.
//
// Packages that read configuration from package-level variables or from init
// functions blank import this package to guarantee the global instance is
// ready first:
//
//	import _ "go_binance_futures/bootstrap"
package bootstrap

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/beego/beego/v2/core/config"
)

// configCandidates are probed in order inside every directory visited while
// walking up from the working directory. app.conf is gitignored, so
// app.conf.example is accepted as a fallback to keep a fresh clone testable.
var configCandidates = []string{
	filepath.Join("conf", "app.conf"),
	filepath.Join("conf", "app.conf.example"),
}

var (
	projectRootOnce sync.Once
	projectRootPath string
)

func init() {
	if path, ok := findConfigFile(); ok {
		// Reloading a file beego already parsed is harmless, so the behaviour
		// stays identical when the process does run from the project root.
		_ = config.InitGlobalInstance("ini", path)
	}
}

// ProjectRoot returns the directory that contains the conf folder, falling
// back to the working directory when it cannot be located. Use it to resolve
// bundled resources such as lang files so they do not depend on the working
// directory either.
func ProjectRoot() string {
	projectRootOnce.Do(func() {
		if path, ok := findConfigFile(); ok {
			// path is <root>/conf/<file>, so strip two levels.
			projectRootPath = filepath.Dir(filepath.Dir(path))
			return
		}
		projectRootPath, _ = os.Getwd()
	})
	return projectRootPath
}

// findConfigFile walks up from the working directory until one of
// configCandidates exists, so it resolves both from the project root and from
// any package directory during "go test".
func findConfigFile() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for {
		for _, candidate := range configCandidates {
			path := filepath.Join(dir, candidate)
			if info, statErr := os.Stat(path); statErr == nil && !info.IsDir() {
				return path, true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
