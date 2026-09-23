package portableskill

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDraftCreateEditValidate(t *testing.T) {
	store := DraftStore{Root: t.TempDir()}
	draft, err := store.Create(context.Background(), "web-skill")
	if err != nil {
		t.Fatal(err)
	}
	if draft.SkillName != "web-skill" || len(draft.Files) != 1 || draft.Files[0] != "SKILL.md" {
		t.Fatalf("unexpected draft: %+v", draft)
	}

	skillMD := `---
name: web-skill
description: Web authored portable skill
metadata:
  version: 1.0.0
allowed-tools: get_symbol_snapshot
---

# Instructions

Read [the reference](references/guide.md).
`
	if err := store.WriteFile(context.Background(), draft.ID, "SKILL.md", []byte(skillMD)); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteFile(context.Background(), draft.ID, "references/guide.md", []byte("# Guide\nUse deterministic data.")); err != nil {
		t.Fatal(err)
	}
	result, err := store.Validate(context.Background(), draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid || result.Name != "web-skill" || result.Version != "1.0.0" {
		t.Fatalf("unexpected validation: %+v", result)
	}
	if len(result.RequestedTools) != 1 || result.RequestedTools[0] != "get_symbol_snapshot" {
		t.Fatalf("requested tools = %+v", result.RequestedTools)
	}
}

func TestDraftValidationRejectsMissingReferenceAndNameMismatch(t *testing.T) {
	store := DraftStore{Root: t.TempDir()}
	draft, err := store.Create(context.Background(), "draft-check")
	if err != nil {
		t.Fatal(err)
	}
	missing := `---
name: draft-check
description: Check invalid references
---

[missing](references/not-found.md)
`
	if err := store.WriteFile(context.Background(), draft.ID, "SKILL.md", []byte(missing)); err != nil {
		t.Fatal(err)
	}
	result, err := store.Validate(context.Background(), draft.ID)
	if err == nil || result.Valid || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected missing reference rejection: result=%+v err=%v", result, err)
	}

	mismatch := strings.Replace(missing, "name: draft-check", "name: other-skill", 1)
	if err := store.WriteFile(context.Background(), draft.ID, "SKILL.md", []byte(mismatch)); err != nil {
		t.Fatal(err)
	}
	_, err = store.Validate(context.Background(), draft.ID)
	if err == nil || !strings.Contains(err.Error(), "must match parent directory") {
		t.Fatalf("expected name-directory mismatch, got %v", err)
	}
}

func TestDraftPathTraversalAndSkillDeletionRejected(t *testing.T) {
	store := DraftStore{Root: t.TempDir()}
	draft, err := store.Create(context.Background(), "safe-draft")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WriteFile(context.Background(), draft.ID, "../escape.txt", []byte("bad")); err == nil {
		t.Fatal("path traversal must be rejected")
	}
	if _, err := store.ReadFile(context.Background(), draft.ID, "/etc/passwd"); err == nil {
		t.Fatal("absolute path must be rejected")
	}
	if err := store.DeleteFile(context.Background(), draft.ID, "SKILL.md"); err == nil {
		t.Fatal("SKILL.md deletion must be rejected")
	}
}

func TestDraftScriptsAreStoredButValidationWarns(t *testing.T) {
	store := DraftStore{Root: t.TempDir()}
	draft, err := store.Create(context.Background(), "script-draft")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WriteFile(context.Background(), draft.ID, "scripts/helper.sh", []byte("#!/bin/sh\necho disabled\n")); err != nil {
		t.Fatal(err)
	}
	result, err := store.Validate(context.Background(), draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range result.Diagnostics {
		if item.Code == "script_execution_disabled" && item.Path == "scripts/helper.sh" {
			found = true
		}
	}
	if !found {
		t.Fatalf("script warning missing: %+v", result.Diagnostics)
	}
}

func TestDraftFilesAccumulateAndDelete(t *testing.T) {
	store := DraftStore{Root: t.TempDir()}
	draft, err := store.Create(context.Background(), "files-draft")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"references/a.md", "references/b.md", "assets/readme.txt"} {
		if err := store.WriteFile(context.Background(), draft.ID, path, []byte(path)); err != nil {
			t.Fatal(err)
		}
	}
	detail, err := store.Detail(context.Background(), draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Files) != 4 {
		t.Fatalf("files = %+v", detail.Files)
	}
	if err := store.DeleteFile(context.Background(), draft.ID, "references/a.md"); err != nil {
		t.Fatal(err)
	}
	detail, err = store.Detail(context.Background(), draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range detail.Files {
		if path == "references/a.md" {
			t.Fatalf("deleted file still present: %+v", detail.Files)
		}
	}
}

func TestDraftClonePublishPreservesImmutableRevision(t *testing.T) {
	store := setupPortableDB(t)
	ctx := context.Background()
	v1 := []byte(`---
name: studio-revision
description: Studio immutable revision test.
metadata:
  version: 1.0.0
allowed-tools: get_market_condition
---

Return V1.
`)
	r1, err := (Importer{Store: store}).ImportFile(ctx, "SKILL.md", bytes.NewReader(v1), int64(len(v1)), false)
	if err != nil {
		t.Fatal(err)
	}
	original, err := store.ReadFile(ctx, r1.Version.ID, "SKILL.md")
	if err != nil {
		t.Fatal(err)
	}

	drafts := DraftStore{Root: t.TempDir(), Store: store}
	draft, err := drafts.CloneVersion(ctx, r1.Version.ID)
	if err != nil {
		t.Fatal(err)
	}
	modified := strings.Replace(original, "version: 1.0.0", "version: 2.0.0", 1)
	modified = strings.Replace(modified, "Return V1.", "Return V2.", 1)
	if err := drafts.WriteFile(ctx, draft.ID, "SKILL.md", []byte(modified)); err != nil {
		t.Fatal(err)
	}
	validation, err := drafts.Validate(ctx, draft.ID)
	if err != nil || !validation.Valid || validation.Version != "2.0.0" {
		t.Fatalf("draft validation=%+v err=%v", validation, err)
	}
	root, _, err := drafts.PackageRoot(ctx, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := (Importer{Store: store}).ImportDraft(ctx, root, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if r2.Version.ID == r1.Version.ID || r2.Version.PackageHash == r1.Version.PackageHash {
		t.Fatalf("publish must create a new immutable revision: v1=%+v v2=%+v", r1.Version, r2.Version)
	}
	after, err := store.ReadFile(ctx, r1.Version.ID, "SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if after != original {
		t.Fatalf("old revision was modified:\n%s", after)
	}
	newContent, err := store.ReadFile(ctx, r2.Version.ID, "SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(newContent, "version: 2.0.0") || !strings.Contains(newContent, "Return V2.") {
		t.Fatalf("new revision content mismatch:\n%s", newContent)
	}
	versions, err := store.ListVersions(ctx, r1.Skill.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 {
		t.Fatalf("versions=%d want=2", len(versions))
	}
}

func TestDraftValidationRejectsBrokenYAML(t *testing.T) {
	store := DraftStore{Root: t.TempDir()}
	draft, err := store.Create(context.Background(), "broken-yaml")
	if err != nil {
		t.Fatal(err)
	}
	broken := []byte("---\nname: [broken\ndescription: nope\n---\nbody\n")
	if err := store.WriteFile(context.Background(), draft.ID, "SKILL.md", broken); err != nil {
		t.Fatal(err)
	}
	result, err := store.Validate(context.Background(), draft.ID)
	if err == nil || result.Valid || !strings.Contains(err.Error(), "parse SKILL.md frontmatter") {
		t.Fatalf("expected broken YAML rejection: result=%+v err=%v", result, err)
	}
}

func TestDraftFileCountLimitRollsBackWrite(t *testing.T) {
	store := DraftStore{Root: t.TempDir()}
	draft, err := store.Create(context.Background(), "file-limit")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for i := 0; i < maxFiles-1; i++ {
		path := fmt.Sprintf("references/%03d.md", i)
		if err := store.WriteFile(ctx, draft.ID, path, []byte("ok")); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	overflow := "references/overflow.md"
	if err := store.WriteFile(ctx, draft.ID, overflow, []byte("must rollback")); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected file-count limit error, got %v", err)
	}
	root, _, err := store.PackageRoot(ctx, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(overflow))); !os.IsNotExist(err) {
		t.Fatalf("overflow file must not remain, stat err=%v", err)
	}
	detail, err := store.Detail(ctx, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Files) != maxFiles {
		t.Fatalf("files=%d want=%d", len(detail.Files), maxFiles)
	}
	if result, err := store.Validate(ctx, draft.ID); err != nil || !result.Valid {
		t.Fatalf("draft must remain valid after rollback: result=%+v err=%v", result, err)
	}
}

func TestDraftTotalSizeLimitOverwriteRestoresPreviousContent(t *testing.T) {
	store := DraftStore{Root: t.TempDir()}
	draft, err := store.Create(context.Background(), "size-rollback")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	root, _, err := store.PackageRoot(ctx, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	skillInfo, err := os.Stat(filepath.Join(root, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	full := bytes.Repeat([]byte{'a'}, int(maxSingleFileBytes))
	for i := 0; i < 7; i++ {
		path := fmt.Sprintf("assets/full-%d.bin", i)
		if err := store.WriteFile(ctx, draft.ID, path, full); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	remaining := maxUnpackedBytes - skillInfo.Size() - 7*maxSingleFileBytes - 128
	if remaining <= 0 || remaining >= maxSingleFileBytes {
		t.Fatalf("unexpected remaining test size %d", remaining)
	}
	targetPath := "assets/target.bin"
	original := bytes.Repeat([]byte{'b'}, int(remaining))
	if err := store.WriteFile(ctx, draft.ID, targetPath, original); err != nil {
		t.Fatal(err)
	}
	replacement := bytes.Repeat([]byte{'c'}, int(remaining+256))
	if int64(len(replacement)) > maxSingleFileBytes {
		t.Fatalf("replacement exceeds single-file limit: %d", len(replacement))
	}
	if err := store.WriteFile(ctx, draft.ID, targetPath, replacement); err == nil || !strings.Contains(err.Error(), "package exceeds") {
		t.Fatalf("expected total-size limit error, got %v", err)
	}
	after, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(targetPath)))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, original) {
		t.Fatal("existing file content was not restored after rejected overwrite")
	}
}
