package leadaccount

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestEncryptedStore(t *testing.T) *EncryptedFileStore {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	store, err := NewEncryptedFileStore(filepath.Join(dir, "lead.enc"), bytes.Repeat([]byte{3}, 32))
	if err != nil {
		t.Fatal(err)
	}
	return store
}
func TestStage41EncryptedCredentialStoreRoundTripAndRotation(t *testing.T) {
	store := newTestEncryptedStore(t)
	if _, err := store.Load(); !errors.Is(err, ErrCredentialsUnavailable) {
		t.Fatalf("missing=%v", err)
	}
	first := Credentials{APIKey: "lead-account-key-first", APISecret: "lead-account-secret-first", PortfolioLabel: "Portfolio 1"}
	if err := store.Save(first); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte(first.APIKey)) || bytes.Contains(data, []byte(first.APISecret)) || bytes.Contains(data, []byte(first.PortfolioLabel)) {
		t.Fatal("encrypted file contains unencrypted credentials")
	}
	info, err := os.Stat(store.path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("file permission=%v", info.Mode())
	}
	loaded, err := store.Load()
	if err != nil || loaded != first {
		t.Fatalf("decryption failed: %v", err)
	}
	rotated := Credentials{APIKey: "lead-account-key-second", APISecret: "lead-account-secret-second", PortfolioLabel: "Portfolio 2"}
	if err := store.Save(rotated); err != nil {
		t.Fatal(err)
	}
	loaded, err = store.Load()
	if err != nil || loaded != rotated {
		t.Fatalf("rotation failed: %v", err)
	}
	if strings.Contains(first.MaskedKey(), first.APIKey) || strings.Contains(rotated.MaskedKey(), rotated.APIKey) {
		t.Fatal("masked key leaked")
	}
	other, err := NewEncryptedFileStore(store.path, bytes.Repeat([]byte{4}, 32))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.Load(); !errors.Is(err, ErrCredentialsCorrupt) {
		t.Fatalf("wrong key accepted: %v", err)
	}
}
func TestStage41EncryptedCredentialStoreRejectsUnsafeInputs(t *testing.T) {
	t.Setenv(MasterKeyEnv, "")
	t.Setenv(StorePathEnv, "")
	if _, err := NewEncryptedFileStoreFromEnv(); !errors.Is(err, ErrKeyUnavailable) {
		t.Fatalf("missing key=%v", err)
	}
	t.Setenv(MasterKeyEnv, base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32)))
	if _, err := NewEncryptedFileStoreFromEnv(); err == nil {
		t.Fatal("missing path accepted")
	}
	if _, err := NewEncryptedFileStore("relative/lead.enc", bytes.Repeat([]byte{3}, 32)); !errors.Is(err, ErrUnsafeLocation) {
		t.Fatalf("relative path=%v", err)
	}
	d := t.TempDir()
	unsafe := filepath.Join(d, "unsafe")
	if err := os.Mkdir(unsafe, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := NewEncryptedFileStore(filepath.Join(unsafe, "lead.enc"), bytes.Repeat([]byte{3}, 32)); !errors.Is(err, ErrUnsafeLocation) {
		t.Fatalf("unsafe dir=%v", err)
	}
	real := filepath.Join(d, "actual")
	if err := os.WriteFile(real, []byte("not secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(d, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := NewEncryptedFileStore(filepath.Join(d, "link"), bytes.Repeat([]byte{3}, 32)); !errors.Is(err, ErrUnsafeLocation) {
		t.Fatalf("symlink=%v", err)
	}
	store := newTestEncryptedStore(t)
	if err := store.Save(Credentials{APIKey: "", APISecret: "secret"}); !errors.Is(err, ErrCredentialsUnavailable) {
		t.Fatalf("empty key accepted: %v", err)
	}
	if err := store.Save(Credentials{APIKey: "key", APISecret: "secret"}); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(store.path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); !errors.Is(err, ErrUnsafeLocation) {
		t.Fatalf("world-readable store allowed: %v", err)
	}
}
func TestStage41EncryptedCredentialStoreTamperFailsClosed(t *testing.T) {
	store := newTestEncryptedStore(t)
	if err := store.Save(Credentials{APIKey: "lead-key", APISecret: "lead-secret"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.path, []byte(`{"version":"lead-account-v1","nonce":"xyz","payload":"xyz"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); !errors.Is(err, ErrCredentialsCorrupt) {
		t.Fatalf("tamper accepted: %v", err)
	}
}

func TestStage41CredentialsNeverJSONExposeKeyOrSecret(t *testing.T) {
	credentials := Credentials{APIKey: "private-key", APISecret: "private-secret", PortfolioLabel: "display-only"}
	value, err := json.Marshal(credentials)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(value, []byte("private-key")) || bytes.Contains(value, []byte("private-secret")) {
		t.Fatalf("public credentials JSON leaks secret fields")
	}
}
