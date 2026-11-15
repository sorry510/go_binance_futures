// Package applog provides a bounded, read-only view of the app's own log file.
// PM2 continues supervising the executable but is not involved in log queries.
package applog

import (
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/beego/beego/v2/core/logs"
)

const (
	FilePath           = "logs/app.log"
	maxReadBytes int64 = 2 << 20
	maxLines           = 1000
)

// Init adds a rotating file adapter and preserves console output for PM2.
func Init() error {
	if err := os.MkdirAll("logs", 0700); err != nil {
		return fmt.Errorf("create application log directory: %w", err)
	}
	cfg := "{\"filename\":\"logs/app.log\",\"maxsize\":20971520,\"maxfiles\":100,\"daily\":true,\"maxdays\":7,\"rotate\":true,\"perm\":\"0600\",\"dirperm\":\"0700\",\"rotateperm\":\"0600\"}"
	// If another package emitted during init(), it may have already
	// activated the default console output. Normalize to one copy.
	_ = logs.GetBeeLogger().DelLogger(logs.AdapterConsole)
	if err := logs.SetLogger(logs.AdapterFile, cfg); err != nil {
		// Beego's first SetLogger clears the implicit default console adapter.
		_ = logs.SetLogger(logs.AdapterConsole)
		return fmt.Errorf("configure application file logger: %w", err)
	}
	if err := logs.SetLogger(logs.AdapterConsole); err != nil {
		return fmt.Errorf("configure application console logger: %w", err)
	}
	// The small number of standard-library log.Print calls (notably Lead
	// safety notices) should also appear in the same application log.
	log.SetOutput(standardLogBridge{})
	log.SetFlags(0)
	return nil
}

type standardLogBridge struct{}

func (standardLogBridge) Write(p []byte) (int, error) {
	logs.Warning("%s", strings.TrimRight(string(p), "\r\n"))
	return len(p), nil
}

type Result struct {
	Lines        []string
	ScannedBytes int64
	HasEarlier   bool
}

var (
	sensitiveValue   = regexp.MustCompile("(?i)\\b((?:api[_-]?key|api[_-]?secret|password|token|secret|authorization)\\s*[:=]\\s*)(\\\"[^\\\"]*\\\"|'[^']*'|[^\\s&,;]+)")
	bearerValue      = regexp.MustCompile("(?i)(\\bBearer\\s+)[A-Za-z0-9._~+/=-]+")
	proxyCredentials = regexp.MustCompile("(?i)(://)[^/\\s@]+@")
)

func maskSecrets(line string) string {
	line = bearerValue.ReplaceAllString(line, "$"+"{1}***")
	line = sensitiveValue.ReplaceAllString(line, "$"+"{1}***")
	return proxyCredentials.ReplaceAllString(line, "$"+"{1}***@")
}

func severityMatches(line, level string) bool {
	switch strings.ToUpper(level) {
	case "", "ALL":
		return true
	case "ERROR":
		return strings.Contains(line, "[E]") || strings.Contains(line, "[C]") || strings.Contains(line, "[A]") || strings.Contains(line, "[M]")
	case "WARN":
		return strings.Contains(line, "[W]")
	case "INFO":
		return strings.Contains(line, "[I]") || strings.Contains(line, "[N]")
	case "DEBUG":
		return strings.Contains(line, "[D]")
	default:
		return false
	}
}

// Recent reads only the bounded tail of the fixed application file.
// Filtering happens after bounded reads, with no arbitrary filename from HTTP.
func Recent(limit int, level, keyword string) (Result, error) {
	return readRecentFile(FilePath, limit, level, keyword)
}

func readRecentFile(filename string, limit int, level, keyword string) (Result, error) {
	result := Result{Lines: []string{}}
	if limit <= 0 {
		limit = 200
	}
	if limit > maxLines {
		limit = maxLines
	}
	if len(keyword) > 128 {
		keyword = keyword[:128]
	}
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	f, err := os.Open(filename)
	if err != nil {
		return result, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return result, err
	}
	if !info.Mode().IsRegular() {
		return result, fmt.Errorf("application log is not a regular file")
	}
	start := info.Size() - maxReadBytes
	if start < 0 {
		start = 0
	}
	result.HasEarlier = start > 0
	count := info.Size() - start
	result.ScannedBytes = count
	b := make([]byte, count)
	_, err = io.ReadFull(io.NewSectionReader(f, start, count), b)
	if err != nil {
		return result, err
	}
	// Discard the partial line when reading from the middle of a file.
	if start > 0 {
		n := strings.IndexByte(string(b), '\n')
		if n < 0 {
			return result, nil
		}
		b = b[n+1:]
	}
	lines := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	for i := len(lines) - 1; i >= 0 && len(result.Lines) < limit; i-- {
		line := strings.TrimSuffix(lines[i], "\r")
		if line == "" || !severityMatches(line, level) {
			continue
		}
		line = maskSecrets(line)
		if keyword != "" && !strings.Contains(strings.ToLower(line), keyword) {
			continue
		}
		result.Lines = append(result.Lines, line)
	}
	for left, right := 0, len(result.Lines)-1; left < right; left, right = left+1, right-1 {
		result.Lines[left], result.Lines[right] = result.Lines[right], result.Lines[left]
	}
	return result, nil
}
