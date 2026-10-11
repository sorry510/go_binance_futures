package leadaccount

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStage41ReviewCredentialPathAndPermissions(t *testing.T) {
	root := t.TempDir()
	private := filepath.Join(root, "private")
	if err := os.Mkdir(private, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "alias")
	if err := os.Symlink(private, link); err != nil {
		t.Fatal(err)
	}
	key := bytes.Repeat([]byte{9}, 32)
	if _, err := NewEncryptedFileStore(filepath.Join(link, "lead.enc"), key); !errors.Is(err, ErrUnsafeLocation) {
		t.Fatalf("symbolic-link parent accepted: %v", err)
	}
	store, err := NewEncryptedFileStore(filepath.Join(private, "lead.enc"), key)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(Credentials{APIKey: "key", APISecret: "secret"}); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(store.path, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); !errors.Is(err, ErrUnsafeLocation) {
		t.Fatalf("executable credential file accepted: %v", err)
	}
}
func TestStage41ReviewRejectedSavePreservesPreviousCredential(t *testing.T) {
	store := newTestEncryptedStore(t)
	orig := Credentials{APIKey: "key", APISecret: "secret"}
	if err := store.Save(orig); err != nil {
		t.Fatal(err)
	}
	cases := []Credentials{
		{APIKey: strings.Repeat("k", 600), APISecret: "secret"},
		{APIKey: "key", APISecret: "secret", PortfolioLabel: strings.Repeat("x", 200)},
		{APIKey: "key"},
	}
	for _, candidate := range cases {
		if err := store.Save(candidate); err == nil {
			t.Fatal("invalid input saved")
		}
		got, err := store.Load()
		if err != nil || got != orig {
			t.Fatalf("previous secret corrupted: %v", err)
		}
	}
	entries, err := os.ReadDir(filepath.Dir(store.path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "lead.enc" {
		t.Fatalf("temporary credential files remain: %v", entries)
	}
}
func TestStage41ReviewEnvelopeVersionFailClosed(t *testing.T) {
	store := newTestEncryptedStore(t)
	if err := store.Save(Credentials{APIKey: "key", APISecret: "secret"}); err != nil {
		t.Fatal(err)
	}
	ciphertext, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}
	var envelope encryptedEnvelope
	if err := json.Unmarshal(ciphertext, &envelope); err != nil {
		t.Fatal(err)
	}
	envelope.Version = "invalid-version"
	modified, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.path, modified, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); !errors.Is(err, ErrCredentialsCorrupt) {
		t.Fatalf("modified envelope accepted: %v", err)
	}
}
