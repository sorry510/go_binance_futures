package portableskill

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var validName = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var markdownLink = regexp.MustCompile(`\]\(([^)]+)\)`)
var allowedFrontmatter = map[string]bool{"name": true, "description": true, "license": true, "compatibility": true, "metadata": true, "allowed-tools": true}

func ParsePackage(root string) (Package, error) {
	return parsePackage(root, filepath.Base(filepath.Clean(root)))
}

func ParseInstalledPackage(root, expectedName string) (Package, error) {
	return parsePackage(root, strings.TrimSpace(expectedName))
}

func parsePackage(root, expectedName string) (Package, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Package{}, err
	}
	skillPath := filepath.Join(root, "SKILL.md")
	raw, err := os.ReadFile(skillPath)
	if err != nil {
		return Package{}, fmt.Errorf("read SKILL.md: %w", err)
	}
	front, body, diagnostics, err := parseSkillMD(raw)
	if err != nil {
		return Package{}, err
	}
	if err := validateFrontmatter(front, expectedName); err != nil {
		return Package{}, err
	}
	files, count, total, scanDiagnostics, err := scanPackageFiles(root)
	if err != nil {
		return Package{}, err
	}
	diagnostics = append(diagnostics, scanDiagnostics...)
	if err := validateFileReferences(root, body); err != nil {
		return Package{}, err
	}
	hash, err := hashPackage(root, files)
	if err != nil {
		return Package{}, err
	}
	requested := strings.Fields(strings.TrimSpace(front.AllowedTools))
	return Package{Root: root, Frontmatter: front, Body: body, PackageHash: hash, Files: files, FileCount: count, TotalBytes: total, RequestedTools: requested, Diagnostics: diagnostics}, nil
}

func parseSkillMD(raw []byte) (Frontmatter, string, []Diagnostic, error) {
	normalized := bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
	if !bytes.HasPrefix(normalized, []byte("---\n")) {
		return Frontmatter{}, "", nil, fmt.Errorf("SKILL.md must start with YAML frontmatter delimiter ---")
	}
	end := bytes.Index(normalized[4:], []byte("\n---\n"))
	if end < 0 {
		return Frontmatter{}, "", nil, fmt.Errorf("SKILL.md frontmatter closing delimiter --- is missing")
	}
	frontRaw := normalized[4 : 4+end]
	body := strings.TrimSpace(string(normalized[4+end+5:]))
	var node yaml.Node
	if err := yaml.Unmarshal(frontRaw, &node); err != nil {
		return Frontmatter{}, "", nil, fmt.Errorf("parse SKILL.md frontmatter: %w", err)
	}
	if len(node.Content) == 0 || len(node.Content[0].Content)%2 != 0 {
		return Frontmatter{}, "", nil, fmt.Errorf("SKILL.md frontmatter must be a YAML mapping")
	}
	mapping := node.Content[0]
	for i := 0; i < len(mapping.Content); i += 2 {
		key := mapping.Content[i].Value
		if !allowedFrontmatter[key] {
			if key == "version" || key == "trusted" {
				return Frontmatter{}, "", nil, fmt.Errorf("unsupported top-level frontmatter field %q; migrate it to metadata.%s", key, key)
			}
			return Frontmatter{}, "", nil, fmt.Errorf("unsupported Agent Skills frontmatter field %q", key)
		}
	}
	var front Frontmatter
	if err := mapping.Decode(&front); err != nil {
		return Frontmatter{}, "", nil, fmt.Errorf("decode SKILL.md frontmatter: %w", err)
	}
	return front, body, nil, nil
}

func validateFrontmatter(front Frontmatter, parent string) error {
	if len(front.Name) < 1 || len(front.Name) > 64 || !validName.MatchString(front.Name) || strings.Contains(front.Name, "--") {
		return fmt.Errorf("invalid skill name %q; use 1-64 lowercase letters, digits and single hyphens", front.Name)
	}
	if parent != front.Name {
		return fmt.Errorf("skill name %q must match parent directory %q", front.Name, parent)
	}
	if len(strings.TrimSpace(front.Description)) < 1 || len(front.Description) > 1024 {
		return fmt.Errorf("description must be 1-1024 characters")
	}
	if front.Compatibility != "" && (len(front.Compatibility) < 1 || len(front.Compatibility) > 500) {
		return fmt.Errorf("compatibility must be 1-500 characters when provided")
	}
	return nil
}

func scanPackageFiles(root string) ([]string, int, int64, []Diagnostic, error) {
	files := []string{}
	var total int64
	diagnostics := []Diagnostic{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || (!info.Mode().IsRegular() && !info.IsDir()) {
			return fmt.Errorf("unsupported special file %s", path)
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if len(files) >= maxFiles {
			return fmt.Errorf("skill package exceeds %d files", maxFiles)
		}
		if info.Size() > maxSingleFileBytes {
			return fmt.Errorf("skill file %s exceeds %d bytes", rel, maxSingleFileBytes)
		}
		total += info.Size()
		if total > maxUnpackedBytes {
			return fmt.Errorf("skill package exceeds %d unpacked bytes", maxUnpackedBytes)
		}
		files = append(files, rel)
		if strings.HasPrefix(rel, "scripts/") {
			diagnostics = append(diagnostics, Diagnostic{Level: "warning", Code: "script_execution_disabled", Message: "scripts are indexed/readable but execution is disabled", Path: rel})
		}
		return nil
	})
	if err != nil {
		return nil, 0, 0, nil, err
	}
	sort.Strings(files)
	return files, len(files), total, diagnostics, nil
}

func validateFileReferences(root, body string) error {
	for _, match := range markdownLink.FindAllStringSubmatch(body, -1) {
		ref := strings.TrimSpace(strings.Split(match[1], "#")[0])
		if ref == "" || strings.Contains(ref, "://") || strings.HasPrefix(ref, "mailto:") {
			continue
		}
		if filepath.IsAbs(ref) {
			return fmt.Errorf("SKILL.md reference %q must be relative", ref)
		}
		resolved, err := safeJoin(root, ref)
		if err != nil {
			return fmt.Errorf("invalid SKILL.md reference %q: %w", ref, err)
		}
		if _, err := os.Stat(resolved); err != nil {
			return fmt.Errorf("SKILL.md reference %q does not exist", ref)
		}
	}
	return nil
}

func hashPackage(root string, files []string) (string, error) {
	h := sha256.New()
	for _, rel := range files {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return "", err
		}
		h.Write([]byte(rel))
		h.Write([]byte{0})
		h.Write(raw)
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func safeJoin(root, rel string) (string, error) {
	rel = filepath.Clean(filepath.FromSlash(strings.ReplaceAll(rel, "\\", "/")))
	if rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path traversal is not allowed")
	}
	joined := filepath.Join(root, rel)
	base, _ := filepath.Abs(root)
	full, _ := filepath.Abs(joined)
	if full != base && !strings.HasPrefix(full, base+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes skill package")
	}
	return full, nil
}
