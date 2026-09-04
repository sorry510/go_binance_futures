package portableskill

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/config"
	_ "go_binance_futures/bootstrap"
	"go_binance_futures/models"
)

const (
	maxArchiveBytes    int64 = 16 << 20
	maxUnpackedBytes   int64 = 32 << 20
	maxSingleFileBytes int64 = 4 << 20
	maxFiles                 = 256
)

type Importer struct{ Store Store }

func DataDir() string {
	value, _ := config.String("agent_skills::data_dir")
	if strings.TrimSpace(value) == "" {
		value = "./data/agent-skills"
	}
	return filepath.Clean(value)
}
func ImportDir() string {
	value, _ := config.String("agent_skills::import_dir")
	if strings.TrimSpace(value) == "" {
		value = "./data/agent-skill-imports"
	}
	return filepath.Clean(value)
}

func (i Importer) ImportFile(ctx context.Context, filename string, reader io.Reader, size int64, activate bool) (ImportResult, error) {
	if size > maxArchiveBytes {
		return ImportResult{}, fmt.Errorf("skill upload exceeds %d bytes", maxArchiveBytes)
	}
	staging, err := newStagingDir()
	if err != nil {
		return ImportResult{}, err
	}
	defer os.RemoveAll(staging)
	base := filepath.Base(strings.TrimSpace(filename))
	var root string
	switch {
	case strings.EqualFold(filepath.Ext(base), ".zip"):
		archivePath := filepath.Join(staging, "upload.zip")
		if err := writeLimited(archivePath, reader, maxArchiveBytes); err != nil {
			return ImportResult{}, err
		}
		root, err = unpackZIP(archivePath, filepath.Join(staging, "unpacked"))
	case base == "SKILL.md":
		raw, readErr := io.ReadAll(io.LimitReader(reader, maxSingleFileBytes+1))
		if readErr != nil {
			return ImportResult{}, readErr
		}
		if int64(len(raw)) > maxSingleFileBytes {
			return ImportResult{}, fmt.Errorf("SKILL.md exceeds %d bytes", maxSingleFileBytes)
		}
		front, _, _, parseErr := parseSkillMD(raw)
		if parseErr != nil {
			return ImportResult{}, parseErr
		}
		if err := validateFrontmatter(front, front.Name); err != nil {
			return ImportResult{}, err
		}
		root = filepath.Join(staging, front.Name)
		if err := os.MkdirAll(root, 0700); err != nil {
			return ImportResult{}, err
		}
		err = os.WriteFile(filepath.Join(root, "SKILL.md"), raw, 0600)
	default:
		return ImportResult{}, fmt.Errorf("only .zip or a single file named SKILL.md can be imported")
	}
	if err != nil {
		return ImportResult{}, err
	}
	return i.install(ctx, root, "upload", base, activate)
}

func (i Importer) ImportDirectory(ctx context.Context, relative string, activate bool) (ImportResult, error) {
	rootBase, err := filepath.Abs(ImportDir())
	if err != nil {
		return ImportResult{}, err
	}
	root, err := safeJoin(rootBase, relative)
	if err != nil {
		return ImportResult{}, err
	}
	if err := rejectSymlinkPath(rootBase, root); err != nil {
		return ImportResult{}, err
	}
	return i.install(ctx, root, "server_directory", filepath.ToSlash(relative), activate)
}

func (i Importer) install(ctx context.Context, root, source, sourceRef string, activate bool) (ImportResult, error) {
	pkg, err := ParsePackage(root)
	if err != nil {
		return ImportResult{}, err
	}
	if existing, err := i.Store.VersionByHash(ctx, pkg.PackageHash); err == nil {
		skill, skillErr := i.Store.GetSkill(ctx, existing.SkillID)
		if skillErr != nil {
			return ImportResult{}, skillErr
		}
		if skill.Deleted == 1 {
			skill, skillErr = i.Store.Undelete(ctx, skill.ID)
			if skillErr != nil {
				return ImportResult{}, skillErr
			}
		}
		if activate && skill.ActiveVersionID != existing.ID {
			if _, err := i.Store.Activate(ctx, existing.ID); err != nil {
				return ImportResult{}, err
			}
			skill, _ = i.Store.GetSkill(ctx, existing.SkillID)
		}
		return ImportResult{Skill: *skill, Version: *existing, Duplicate: true}, nil
	}
	finalRel := filepath.ToSlash(filepath.Join(pkg.Frontmatter.Name, pkg.PackageHash))
	finalPath := filepath.Join(DataDir(), filepath.FromSlash(finalRel))
	if err := copyPackage(pkg.Root, finalPath); err != nil {
		return ImportResult{}, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(finalPath)
		}
	}()
	metadataJSON, _ := json.Marshal(pkg.Frontmatter.Metadata)
	requestedJSON, _ := json.Marshal(pkg.RequestedTools)
	validationJSON, _ := json.Marshal(pkg.Diagnostics)
	version := models.AgentSkillVersion{Name: pkg.Frontmatter.Name, PackageHash: pkg.PackageHash, Version: pkg.Frontmatter.Metadata["version"], Description: pkg.Frontmatter.Description, License: pkg.Frontmatter.License, Compatibility: pkg.Frontmatter.Compatibility, MetadataJSON: string(metadataJSON), RequestedToolsJSON: string(requestedJSON), ValidationStatus: ValidationValid, ValidationJSON: string(validationJSON), Source: source, SourceRef: sourceRef, PackagePath: finalRel, FileCount: pkg.FileCount, TotalBytes: pkg.TotalBytes, CreatedAt: time.Now().UTC().UnixMilli()}
	result, err := i.Store.Install(ctx, version, pkg.RequestedTools, activate)
	if err != nil {
		return ImportResult{}, err
	}
	cleanup = false
	return result, nil
}

func newStagingDir() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	base := filepath.Join(DataDir(), ".staging")
	if err := os.MkdirAll(base, 0700); err != nil {
		return "", err
	}
	return os.MkdirTemp(base, hex.EncodeToString(b[:])+"-")
}
func writeLimited(path string, r io.Reader, max int64) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	n, err := io.Copy(f, io.LimitReader(r, max+1))
	if err != nil {
		return err
	}
	if n > max {
		return fmt.Errorf("skill upload exceeds %d bytes", max)
	}
	return nil
}

func unpackZIP(path, destination string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("open skill zip: %w", err)
	}
	defer r.Close()
	if len(r.File) > maxFiles {
		return "", fmt.Errorf("skill zip exceeds %d entries", maxFiles)
	}
	if err := os.MkdirAll(destination, 0700); err != nil {
		return "", err
	}
	var total int64
	for _, f := range r.File {
		name := strings.ReplaceAll(f.Name, "\\", "/")
		firstSegment := strings.SplitN(name, "/", 2)[0]
		if strings.HasPrefix(name, "/") || strings.Contains(firstSegment, ":") || strings.Contains(name, "../") || name == ".." || filepath.IsAbs(name) {
			return "", fmt.Errorf("unsafe zip path %q", f.Name)
		}
		if f.Mode()&os.ModeSymlink != 0 || (!f.Mode().IsRegular() && !f.FileInfo().IsDir()) {
			return "", fmt.Errorf("zip entry %q is not a regular file/directory", f.Name)
		}
		target, err := safeJoin(destination, name)
		if err != nil {
			return "", err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0700); err != nil {
				return "", err
			}
			continue
		}
		if int64(f.UncompressedSize64) > maxSingleFileBytes {
			return "", fmt.Errorf("zip entry %q exceeds %d bytes", f.Name, maxSingleFileBytes)
		}
		total += int64(f.UncompressedSize64)
		if total > maxUnpackedBytes {
			return "", fmt.Errorf("skill zip exceeds %d unpacked bytes", maxUnpackedBytes)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return "", err
		}
		src, err := f.Open()
		if err != nil {
			return "", err
		}
		err = writeLimitedExisting(target, src, maxSingleFileBytes)
		src.Close()
		if err != nil {
			return "", err
		}
	}
	if _, err := os.Stat(filepath.Join(destination, "SKILL.md")); err == nil {
		return normalizeFlatRoot(destination)
	}
	entries, err := os.ReadDir(destination)
	if err != nil {
		return "", err
	}
	dirs := []string{}
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		} else {
			return "", fmt.Errorf("zip must contain SKILL.md at root or one skill directory")
		}
	}
	if len(dirs) != 1 {
		return "", fmt.Errorf("zip must contain exactly one skill package")
	}
	return normalizeRootSkill(filepath.Join(destination, dirs[0]))
}
func normalizeRootSkill(root string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(root, "SKILL.md"))
	if err != nil {
		return "", fmt.Errorf("SKILL.md not found in skill package")
	}
	front, _, _, err := parseSkillMD(raw)
	if err != nil {
		return "", err
	}
	if err := validateFrontmatter(front, filepath.Base(root)); err != nil {
		return "", err
	}
	return root, nil
}
func normalizeFlatRoot(root string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(root, "SKILL.md"))
	if err != nil {
		return "", err
	}
	front, _, _, err := parseSkillMD(raw)
	if err != nil {
		return "", err
	}
	if err := validateFrontmatter(front, front.Name); err != nil {
		return "", err
	}
	target := filepath.Join(filepath.Dir(root), front.Name)
	if err := os.MkdirAll(target, 0700); err != nil {
		return "", err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if err := os.Rename(filepath.Join(root, entry.Name()), filepath.Join(target, entry.Name())); err != nil {
			return "", err
		}
	}
	if err := os.Remove(root); err != nil {
		return "", err
	}
	return target, nil
}
func writeLimitedExisting(path string, r io.Reader, max int64) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	n, err := io.Copy(f, io.LimitReader(r, max+1))
	if err != nil {
		return err
	}
	if n > max {
		return fmt.Errorf("file exceeds %d bytes", max)
	}
	return nil
}
func copyPackage(src, dst string) error {
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	return filepath.WalkDir(src, func(path string, e os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := e.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink %s is not allowed", rel)
		}
		if e.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}
func rejectSymlinkPath(base, target string) error {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return err
	}
	current := base
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink path component %q is not allowed", part)
		}
	}
	return nil
}
