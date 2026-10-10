package leadaccount

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go_binance_futures/feature/api/binance"
)

var (
	ErrKeyUnavailable         = errors.New("lead credential encryption key is not configured")
	ErrCredentialsUnavailable = errors.New("lead credentials are not configured")
	ErrCredentialsCorrupt     = errors.New("lead credentials cannot be decrypted")
	ErrUnsafeLocation         = errors.New("lead credential storage location is insecure")
)

const (
	MasterKeyEnv      = "BINANCE_LEAD_CREDENTIAL_KEY"
	StorePathEnv      = "BINANCE_LEAD_CREDENTIAL_FILE"
	credentialVersion = "lead-account-v1"
	maxCredentialFile = 16384
)

// Credentials are internal input. Do not JSON serialize, log, or return them
// from any API response. The label is untrusted UI metadata, not identity.
type Credentials struct {
	APIKey         string `json:"-"`
	APISecret      string `json:"-"`
	PortfolioLabel string `json:"portfolio_label"`
}

func (c Credentials) Valid() bool {
	return strings.TrimSpace(c.APIKey) != "" && strings.TrimSpace(c.APISecret) != ""
}
func (c Credentials) MaskedKey() string {
	k := strings.TrimSpace(c.APIKey)
	if len(k) < 4 {
		return "****"
	}
	return "****" + k[len(k)-4:]
}

// Only this unexported payload is JSON encoded before encryption. Public
// Credentials are deliberately non-serializable for key/secret fields.
type sealedCredentialData struct {
	APIKey         string `json:"api_key"`
	APISecret      string `json:"api_secret"`
	PortfolioLabel string `json:"portfolio_label"`
}
type encryptedEnvelope struct {
	Version string `json:"version"`
	Nonce   string `json:"nonce"`
	Payload string `json:"payload"`
}

// EncryptedFileStore has no implicit initialization or file writes. Env master
// key must be independent from app.conf and persisted securely by the user.
// The key is not generated automatically and is never written to disk.
type EncryptedFileStore struct {
	mu   sync.Mutex
	path string
	key  [32]byte
}

func NewEncryptedFileStoreFromEnv() (*EncryptedFileStore, error) {
	raw := strings.TrimSpace(os.Getenv(MasterKeyEnv))
	if raw == "" {
		return nil, ErrKeyUnavailable
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("%s must be standard base64 of exactly 32 random bytes", MasterKeyEnv)
	}
	path := strings.TrimSpace(os.Getenv(StorePathEnv))
	if path == "" {
		return nil, fmt.Errorf("%s must be an absolute path outside the repository", StorePathEnv)
	}
	return NewEncryptedFileStore(path, key)
}
func NewEncryptedFileStore(path string, key []byte) (*EncryptedFileStore, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("lead encryption key must be exactly 32 bytes")
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) == string(os.PathSeparator) || filepath.Base(path) == "." {
		return nil, ErrUnsafeLocation
	}
	clean := filepath.Clean(path)
	// Refuse a symlink target, preventing following secrets into app.conf or
	// elsewhere. Parent directories must already exist, private, and non-symlink.
	if err := verifyCredentialPath(clean); err != nil {
		return nil, err
	}
	s := &EncryptedFileStore{path: clean}
	copy(s.key[:], key)
	return s, nil
}
func verifyCredentialPath(path string) error {
	parent := filepath.Dir(path)
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return ErrUnsafeLocation
	}
	if f, err := os.Lstat(path); err == nil {
		if !f.Mode().IsRegular() || f.Mode().Perm()&0077 != 0 || f.Mode().Perm()&0100 != 0 {
			return ErrUnsafeLocation
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return ErrUnsafeLocation
	}
	return nil
}
func (s *EncryptedFileStore) Load() (Credentials, error) {
	if s == nil {
		return Credentials{}, ErrCredentialsUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := verifyCredentialPath(s.path); err != nil {
		return Credentials{}, err
	}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return Credentials{}, ErrCredentialsUnavailable
	}
	if err != nil {
		return Credentials{}, ErrCredentialsCorrupt
	}
	if len(data) > maxCredentialFile {
		return Credentials{}, ErrCredentialsCorrupt
	}
	var envelope encryptedEnvelope
	if json.Unmarshal(data, &envelope) != nil || envelope.Version != credentialVersion {
		return Credentials{}, ErrCredentialsCorrupt
	}
	nonce, e1 := base64.RawURLEncoding.DecodeString(envelope.Nonce)
	ciphertext, e2 := base64.RawURLEncoding.DecodeString(envelope.Payload)
	block, err := aes.NewCipher(s.key[:])
	if err != nil || e1 != nil || e2 != nil {
		return Credentials{}, ErrCredentialsCorrupt
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(nonce) != gcm.NonceSize() {
		return Credentials{}, ErrCredentialsCorrupt
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, []byte(credentialVersion+":"+string(binance.LeadAccountID)))
	if err != nil {
		return Credentials{}, ErrCredentialsCorrupt
	}
	var raw sealedCredentialData
	if json.Unmarshal(plain, &raw) != nil {
		return Credentials{}, ErrCredentialsCorrupt
	}
	credentials := Credentials{APIKey: raw.APIKey, APISecret: raw.APISecret, PortfolioLabel: raw.PortfolioLabel}
	if !credentials.Valid() {
		return Credentials{}, ErrCredentialsCorrupt
	}
	return credentials, nil
}
func (s *EncryptedFileStore) Save(credentials Credentials) error {
	if s == nil {
		return ErrKeyUnavailable
	}
	if !credentials.Valid() {
		return ErrCredentialsUnavailable
	}
	if len(credentials.APIKey) > 512 || len(credentials.APISecret) > 512 || len(credentials.PortfolioLabel) > 128 {
		return fmt.Errorf("lead credential input exceeds maximum length")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := verifyCredentialPath(s.path); err != nil {
		return err
	}
	raw, err := json.Marshal(sealedCredentialData{
		APIKey: credentials.APIKey, APISecret: credentials.APISecret, PortfolioLabel: credentials.PortfolioLabel,
	})
	if err != nil {
		return fmt.Errorf("encode lead credentials: %w", err)
	}
	block, err := aes.NewCipher(s.key[:])
	if err != nil {
		return fmt.Errorf("create lead encryption cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("create lead encryption mode: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("create lead nonce: %w", err)
	}
	payload := gcm.Seal(nil, nonce, raw, []byte(credentialVersion+":"+string(binance.LeadAccountID)))
	encoded, err := json.Marshal(encryptedEnvelope{
		Version: credentialVersion, Nonce: base64.RawURLEncoding.EncodeToString(nonce), Payload: base64.RawURLEncoding.EncodeToString(payload),
	})
	if err != nil {
		return fmt.Errorf("encode encrypted lead credential: %w", err)
	}
	dir := filepath.Dir(s.path)
	// Atomic update; no transient plaintext or partially-written JSON.
	f, err := os.CreateTemp(dir, ".lead-credential-*.tmp")
	if err != nil {
		return fmt.Errorf("create encrypted lead credential file: %w", err)
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(encoded)
	}
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("persist encrypted lead credential: %w", err)
	}
	if err := verifyCredentialPath(s.path); err != nil {
		return err
	}
	if err := os.Rename(name, s.path); err != nil {
		return fmt.Errorf("commit encrypted lead credential: %w", err)
	}
	// Directory fsync is best-effort on supported filesystems.
	if dirfd, err := os.Open(dir); err == nil {
		_ = dirfd.Sync()
		_ = dirfd.Close()
	}
	return nil
}
