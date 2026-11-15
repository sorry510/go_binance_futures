package applog

import (
	"os"
	"path/filepath"

	"github.com/beego/beego/v2/core/logs"
	"strings"
	"testing"
)

func TestReadRecentFileFiltersAndLimits(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "app.log")
	body := "2026 [I] initial\n2026 [E] failed BTCUSDT api_key=example\n2026 [W] warned\n2026 [I] latest\n"
	if err := os.WriteFile(filename, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := readRecentFile(filename, 2, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Lines) != 2 || !strings.Contains(result.Lines[0], "warned") || !strings.Contains(result.Lines[1], "latest") {
		t.Fatalf("wrong tail: %+v", result)
	}
	result, err = readRecentFile(filename, 200, "ERROR", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Lines) != 1 || strings.Contains(result.Lines[0], "example") || !strings.Contains(result.Lines[0], "api_key=***") {
		t.Fatalf("filtered log disclosed secret: %+v", result)
	}
	result, err = readRecentFile(filename, 200, "WARN", "")
	if err != nil || len(result.Lines) != 1 {
		t.Fatalf("warning filter: %+v %v", result, err)
	}
}

func TestReadRecentFileBoundedAndPartialLine(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "app.log")
	payload := strings.Repeat("X", int(maxReadBytes)+150) + "\n2026 [I] last line\n"
	if err := os.WriteFile(filename, []byte(payload), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := readRecentFile(filename, 200, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasEarlier || result.ScannedBytes != maxReadBytes || len(result.Lines) != 1 || result.Lines[0] != "2026 [I] last line" {
		t.Fatalf("bounded tail failed: %+v", result)
	}
}

func TestMaskSecrets(t *testing.T) {
	line := "api_secret=abc123 password:xyz Authorization: Bearer token123 https://name:pwd@host/test"
	out := maskSecrets(line)
	for _, secret := range []string{"abc123", "xyz", "token123", "name:pwd"} {
		if strings.Contains(out, secret) {
			t.Fatalf("leaked %q: %s", secret, out)
		}
	}
}

func TestReadMissingLogFile(t *testing.T) {
	if _, err := readRecentFile(filepath.Join(t.TempDir(), "missing"), 200, "", ""); err == nil {
		t.Fatal("missing file must report an error")
	}
}

// Verify the actual Beego adapter writes a private, readable file while
// retaining its console adapter for PM2.
func TestInitWritesPrivateLogFile(t *testing.T) {
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(original) }()
	if err := Init(); err != nil {
		t.Fatal(err)
	}
	logs.Info("APPLICATION_FILE_LOG_TEST")
	logs.GetBeeLogger().Flush()
	content, err := os.ReadFile(FilePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "APPLICATION_FILE_LOG_TEST") ||
		!strings.Contains(string(content), "[I]") {
		t.Fatalf("Beego log or severity marker not persisted: %q", content)
	}
	stat, err := os.Stat(FilePath)
	if err != nil {
		t.Fatal(err)
	}
	if stat.Mode().Perm() != 0600 {
		t.Fatalf("log mode = %o, expected 0600", stat.Mode().Perm())
	}
}
