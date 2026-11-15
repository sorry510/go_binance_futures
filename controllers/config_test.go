package controllers

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestSaveIfChangedNoopPreservesMtime(t *testing.T) {
	p := filepath.Join(t.TempDir(), "app.conf")
	content := []byte("appname = test\n[web]\nport = 3333\n")
	if err := os.WriteFile(p, content, 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	changed, err := saveIfChanged(p, content)
	if err != nil || changed {
		t.Fatalf("unchanged content must be noop: changed=%v err=%v", changed, err)
	}
	after, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if !before.ModTime().Equal(after.ModTime()) {
		t.Fatalf("unchanged file was rewritten: before=%v after=%v", before.ModTime(), after.ModTime())
	}
}

func TestSaveIfChangedWritesValidConfigPrivately(t *testing.T) {
	p := filepath.Join(t.TempDir(), "app.conf")
	if err := os.WriteFile(p, []byte("appname = old\n"), 0644); err != nil {
		t.Fatal(err)
	}
	newData := []byte("appname = new\n[web]\nport = 3334\n")
	changed, err := saveIfChanged(p, newData)
	if err != nil || !changed {
		t.Fatalf("expected update: changed=%v err=%v", changed, err)
	}
	after, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(newData) {
		t.Fatal("saved config differs")
	}
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("config file mode = %o, want 0600", info.Mode().Perm())
	}
	files, err := filepath.Glob(filepath.Join(filepath.Dir(p), ".app.conf-*.tmp"))
	if err != nil || len(files) != 0 {
		t.Fatalf("temp files left behind: %v, %v", files, err)
	}
}

func TestSaveIfChangedRejectsInvalidWithoutReplacingFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "app.conf")
	initial := []byte("appname = old\n")
	if err := os.WriteFile(p, initial, 0600); err != nil {
		t.Fatal(err)
	}
	for _, data := range [][]byte{nil, []byte("[web]\nport=3333\n")} {
		changed, err := saveIfChanged(p, data)
		if err == nil || changed {
			t.Fatalf("bad config accepted: %q changed=%v err=%v", data, changed, err)
		}
		got, err := os.ReadFile(p)
		if err != nil || string(got) != string(initial) {
			t.Fatalf("bad config replaced original: %q err=%v", got, err)
		}
	}
}

func TestSaveIfChangedConcurrentSaveOnlyOneWrites(t *testing.T) {
	p := filepath.Join(t.TempDir(), "app.conf")
	if err := os.WriteFile(p, []byte("appname = old\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	changedCount := make(chan bool, 10)
	errs := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			changed, err := saveIfChanged(p, []byte("appname = new\n"))
			changedCount <- changed
			errs <- err
		}()
	}
	wg.Wait()
	close(changedCount)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	count := 0
	for changed := range changedCount {
		if changed {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one write, got %d", count)
	}
}
