package portableskill

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const defaultDraftRoot = "./data/agent-skill-drafts"

var (
	draftIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)
	draftMu        sync.Mutex
)

type DraftStore struct {
	Root  string
	Store Store
}

type Draft struct {
	ID              string `json:"id"`
	SkillName       string `json:"skill_name"`
	SourceVersionID int64  `json:"source_version_id,omitempty"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

type DraftDetail struct {
	Draft
	Files []string `json:"files"`
}

type DraftValidation struct {
	Valid          bool         `json:"valid"`
	Error          string       `json:"error,omitempty"`
	Name           string       `json:"name,omitempty"`
	Description    string       `json:"description,omitempty"`
	Version        string       `json:"version,omitempty"`
	PackageHash    string       `json:"package_hash,omitempty"`
	RequestedTools []string     `json:"requested_tools"`
	Diagnostics    []Diagnostic `json:"diagnostics"`
	Files          []string     `json:"files"`
	FileCount      int          `json:"file_count"`
	TotalBytes     int64        `json:"total_bytes"`
}

func DraftRoot() string { return filepath.Clean(defaultDraftRoot) }

func MaxSingleFileBytes() int64 { return maxSingleFileBytes }

func (s DraftStore) root() string {
	if strings.TrimSpace(s.Root) != "" {
		return filepath.Clean(s.Root)
	}
	return DraftRoot()
}

func (s DraftStore) store() Store {
	return s.Store
}

func (s DraftStore) Create(ctx context.Context, name string) (DraftDetail, error) {
	if err := ctx.Err(); err != nil {
		return DraftDetail{}, err
	}
	name = strings.TrimSpace(name)
	if err := validateDraftSkillName(name); err != nil {
		return DraftDetail{}, err
	}

	draftMu.Lock()
	defer draftMu.Unlock()

	id, err := newDraftID()
	if err != nil {
		return DraftDetail{}, err
	}
	now := time.Now().UTC().UnixMilli()
	meta := Draft{ID: id, SkillName: name, CreatedAt: now, UpdatedAt: now}
	root, err := s.packageRootFor(meta)
	if err != nil {
		return DraftDetail{}, err
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return DraftDetail{}, err
	}
	if err := os.WriteFile(filepath.Join(root, "SKILL.md"), []byte(defaultDraftSkillMD(name)), 0600); err != nil {
		_ = os.RemoveAll(s.mustDraftDir(id))
		return DraftDetail{}, err
	}
	if err := s.writeMeta(meta); err != nil {
		_ = os.RemoveAll(s.mustDraftDir(id))
		return DraftDetail{}, err
	}
	return s.detailFromMeta(meta)
}

func (s DraftStore) CloneVersion(ctx context.Context, versionID int64) (DraftDetail, error) {
	if err := ctx.Err(); err != nil {
		return DraftDetail{}, err
	}
	if versionID <= 0 {
		return DraftDetail{}, fmt.Errorf("source version id is required")
	}
	store := s.store()
	version, err := store.Version(ctx, versionID)
	if err != nil {
		return DraftDetail{}, err
	}
	skill, err := store.GetSkill(ctx, version.SkillID)
	if err != nil {
		return DraftDetail{}, err
	}
	if skill.Type != SkillTypePortable {
		return DraftDetail{}, fmt.Errorf("skill %q is not portable", skill.Name)
	}

	draftMu.Lock()
	defer draftMu.Unlock()

	id, err := newDraftID()
	if err != nil {
		return DraftDetail{}, err
	}
	now := time.Now().UTC().UnixMilli()
	meta := Draft{ID: id, SkillName: version.Name, SourceVersionID: version.ID, CreatedAt: now, UpdatedAt: now}
	dst, err := s.packageRootFor(meta)
	if err != nil {
		return DraftDetail{}, err
	}
	src, err := PackageRoot(*version)
	if err != nil {
		return DraftDetail{}, err
	}
	if err := copyPackage(src, dst); err != nil {
		_ = os.RemoveAll(s.mustDraftDir(id))
		return DraftDetail{}, err
	}
	if err := s.writeMeta(meta); err != nil {
		_ = os.RemoveAll(s.mustDraftDir(id))
		return DraftDetail{}, err
	}
	return s.detailFromMeta(meta)
}
func (s DraftStore) List(ctx context.Context) ([]Draft, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root := s.root()
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return []Draft{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := make([]Draft, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || !draftIDPattern.MatchString(entry.Name()) {
			continue
		}
		meta, err := s.readMeta(entry.Name())
		if err != nil {
			continue
		}
		result = append(result, meta)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].UpdatedAt == result[j].UpdatedAt {
			return result[i].ID > result[j].ID
		}
		return result[i].UpdatedAt > result[j].UpdatedAt
	})
	return result, nil
}

func (s DraftStore) Detail(ctx context.Context, id string) (DraftDetail, error) {
	if err := ctx.Err(); err != nil {
		return DraftDetail{}, err
	}
	meta, err := s.readMeta(id)
	if err != nil {
		return DraftDetail{}, err
	}
	return s.detailFromMeta(meta)
}

func (s DraftStore) PackageRoot(ctx context.Context, id string) (string, Draft, error) {
	if err := ctx.Err(); err != nil {
		return "", Draft{}, err
	}
	meta, err := s.readMeta(id)
	if err != nil {
		return "", Draft{}, err
	}
	root, err := s.packageRootFor(meta)
	return root, meta, err
}

func (s DraftStore) ReadFile(ctx context.Context, id, relative string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	root, _, err := s.PackageRoot(ctx, id)
	if err != nil {
		return "", err
	}
	path, err := draftFilePath(root, relative)
	if err != nil {
		return "", err
	}
	if err := ensureNoSymlinkComponents(root, path); err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("resource is not a regular file")
	}
	if info.Size() > maxSingleFileBytes {
		return "", fmt.Errorf("resource exceeds %d bytes", maxSingleFileBytes)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if strings.IndexByte(string(raw), 0) >= 0 {
		return "", fmt.Errorf("binary resources are not exposed as text")
	}
	return string(raw), nil
}

func (s DraftStore) WriteFile(ctx context.Context, id, relative string, raw []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if int64(len(raw)) > maxSingleFileBytes {
		return fmt.Errorf("skill file exceeds %d bytes", maxSingleFileBytes)
	}

	draftMu.Lock()
	defer draftMu.Unlock()

	root, meta, err := s.PackageRoot(ctx, id)
	if err != nil {
		return err
	}
	path, err := draftFilePath(root, relative)
	if err != nil {
		return err
	}
	if err := ensureNoSymlinkComponents(root, filepath.Dir(path)); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if err := ensureNoSymlinkComponents(root, filepath.Dir(path)); err != nil {
		return err
	}
	var (
		previousRaw []byte
		existed     bool
	)
	if info, statErr := os.Lstat(path); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink file is not allowed")
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("draft path is not a regular file")
		}
		previousRaw, err = os.ReadFile(path)
		if err != nil {
			return err
		}
		existed = true
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		return err
	}
	if _, _, _, _, err := scanPackageFiles(root); err != nil {
		if existed {
			_ = os.WriteFile(path, previousRaw, 0600)
		} else {
			_ = os.Remove(path)
			removeEmptyDraftParents(root, filepath.Dir(path))
		}
		return err
	}
	return s.touch(meta)
}

func (s DraftStore) DeleteFile(ctx context.Context, id, relative string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	relative = normalizeDraftRelative(relative)
	if relative == "SKILL.md" {
		return fmt.Errorf("SKILL.md cannot be deleted from a draft")
	}

	draftMu.Lock()
	defer draftMu.Unlock()

	root, meta, err := s.PackageRoot(ctx, id)
	if err != nil {
		return err
	}
	path, err := draftFilePath(root, relative)
	if err != nil {
		return err
	}
	if err := ensureNoSymlinkComponents(root, path); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("only regular draft files can be deleted")
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	removeEmptyDraftParents(root, filepath.Dir(path))
	return s.touch(meta)
}

func (s DraftStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	draftMu.Lock()
	defer draftMu.Unlock()
	dir, err := s.draftDir(id)
	if err != nil {
		return err
	}
	if _, err := s.readMeta(id); err != nil {
		return err
	}
	return os.RemoveAll(dir)
}
func (s DraftStore) Validate(ctx context.Context, id string) (DraftValidation, error) {
	root, _, err := s.PackageRoot(ctx, id)
	if err != nil {
		return DraftValidation{}, err
	}
	pkg, err := ParsePackage(root)
	if err != nil {
		return DraftValidation{Valid: false, Error: err.Error(), RequestedTools: []string{}, Diagnostics: []Diagnostic{}, Files: []string{}}, err
	}
	return validationFromPackage(pkg), nil
}

func validationFromPackage(pkg Package) DraftValidation {
	version := ""
	if pkg.Frontmatter.Metadata != nil {
		version = pkg.Frontmatter.Metadata["version"]
	}
	return DraftValidation{
		Valid:          true,
		Name:           pkg.Frontmatter.Name,
		Description:    pkg.Frontmatter.Description,
		Version:        version,
		PackageHash:    pkg.PackageHash,
		RequestedTools: append([]string(nil), pkg.RequestedTools...),
		Diagnostics:    append([]Diagnostic(nil), pkg.Diagnostics...),
		Files:          append([]string(nil), pkg.Files...),
		FileCount:      pkg.FileCount,
		TotalBytes:     pkg.TotalBytes,
	}
}

func (s DraftStore) detailFromMeta(meta Draft) (DraftDetail, error) {
	root, err := s.packageRootFor(meta)
	if err != nil {
		return DraftDetail{}, err
	}
	files, _, _, _, err := scanPackageFiles(root)
	if err != nil {
		return DraftDetail{}, err
	}
	return DraftDetail{Draft: meta, Files: files}, nil
}

func (s DraftStore) readMeta(id string) (Draft, error) {
	dir, err := s.draftDir(id)
	if err != nil {
		return Draft{}, err
	}
	if err := ensureNoSymlinkComponents(s.root(), dir); err != nil {
		return Draft{}, err
	}
	raw, err := os.ReadFile(filepath.Join(dir, "draft.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return Draft{}, fmt.Errorf("skill draft %q not found", id)
		}
		return Draft{}, err
	}
	var meta Draft
	if err := json.Unmarshal(raw, &meta); err != nil {
		return Draft{}, fmt.Errorf("decode draft metadata: %w", err)
	}
	if meta.ID != id || !draftIDPattern.MatchString(meta.ID) {
		return Draft{}, fmt.Errorf("invalid draft metadata")
	}
	if err := validateDraftSkillName(meta.SkillName); err != nil {
		return Draft{}, fmt.Errorf("invalid draft metadata: %w", err)
	}
	return meta, nil
}

func (s DraftStore) writeMeta(meta Draft) error {
	dir, err := s.draftDir(meta.ID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "draft.json"), raw, 0600)
}

func (s DraftStore) touch(meta Draft) error {
	meta.UpdatedAt = time.Now().UTC().UnixMilli()
	return s.writeMeta(meta)
}

func (s DraftStore) draftDir(id string) (string, error) {
	id = strings.TrimSpace(id)
	if !draftIDPattern.MatchString(id) {
		return "", fmt.Errorf("invalid draft id")
	}
	return safeJoin(s.root(), id)
}

func (s DraftStore) mustDraftDir(id string) string {
	dir, _ := s.draftDir(id)
	return dir
}

func (s DraftStore) packageRootFor(meta Draft) (string, error) {
	dir, err := s.draftDir(meta.ID)
	if err != nil {
		return "", err
	}
	return safeJoin(dir, filepath.ToSlash(filepath.Join("package", meta.SkillName)))
}

func validateDraftSkillName(name string) error {
	if len(name) < 1 || len(name) > 64 || !validName.MatchString(name) || strings.Contains(name, "--") {
		return fmt.Errorf("invalid skill name %q; use 1-64 lowercase letters, digits and single hyphens", name)
	}
	return nil
}

func newDraftID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
func draftFilePath(root, relative string) (string, error) {
	relative = normalizeDraftRelative(relative)
	if relative == "" || relative == "." {
		return "", fmt.Errorf("file path is required")
	}
	return safeJoin(root, relative)
}

func normalizeDraftRelative(relative string) string {
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.ReplaceAll(strings.TrimSpace(relative), "\\", "/"))))
}

func ensureNoSymlinkComponents(root, target string) error {
	base, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	full, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(base, full)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path escapes skill draft")
	}
	if info, err := os.Lstat(base); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("draft root symlink is not allowed")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	current := base
	if rel == "." {
		return nil
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink path component %q is not allowed", part)
		}
	}
	return nil
}

func removeEmptyDraftParents(root, dir string) {
	base, _ := filepath.Abs(root)
	current, _ := filepath.Abs(dir)
	for current != base && strings.HasPrefix(current, base+string(filepath.Separator)) {
		if err := os.Remove(current); err != nil {
			return
		}
		current = filepath.Dir(current)
	}
}

func defaultDraftSkillMD(name string) string {
	return fmt.Sprintf(`---
name: %s
description: Describe what this Skill does.
metadata:
  version: 0.1.0
---

# Instructions

Describe when this Skill should be used and how it should perform the task.
`, name)
}

func (i Importer) ImportDraft(ctx context.Context, root, draftID string) (ImportResult, error) {
	draftID = strings.TrimSpace(draftID)
	if !draftIDPattern.MatchString(draftID) {
		return ImportResult{}, fmt.Errorf("invalid draft id")
	}
	return i.install(ctx, root, "web_draft", draftID, false)
}
