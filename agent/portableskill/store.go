package portableskill

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/models"
)

type Store struct{ Alias string }

func (s Store) orm() orm.Ormer {
	if strings.TrimSpace(s.Alias) != "" {
		return orm.NewOrmUsingDB(s.Alias)
	}
	return orm.NewOrm()
}

func (s Store) GetSkill(ctx context.Context, id int64) (*models.AgentSkill, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	row := &models.AgentSkill{ID: id}
	if err := s.orm().Read(row); err != nil {
		return nil, err
	}
	return row, nil
}
func (s Store) GetSkillByName(ctx context.Context, name string) (*models.AgentSkill, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var row models.AgentSkill
	err := s.orm().QueryTable(new(models.AgentSkill)).Filter("name", strings.TrimSpace(name)).Filter("deleted", 0).One(&row)
	return &row, err
}
func (s Store) VersionByHash(ctx context.Context, hash string) (*models.AgentSkillVersion, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var row models.AgentSkillVersion
	err := s.orm().QueryTable(new(models.AgentSkillVersion)).Filter("package_hash", strings.TrimSpace(hash)).One(&row)
	return &row, err
}
func (s Store) Version(ctx context.Context, id int64) (*models.AgentSkillVersion, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	row := &models.AgentSkillVersion{ID: id}
	if err := s.orm().Read(row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s Store) Install(ctx context.Context, version models.AgentSkillVersion, requested []string, activate bool) (ImportResult, error) {
	if err := ctx.Err(); err != nil {
		return ImportResult{}, err
	}
	var result ImportResult
	err := s.orm().DoTxWithCtx(ctx, func(_ context.Context, tx orm.TxOrmer) error {
		now := time.Now().UTC().UnixMilli()
		var skill models.AgentSkill
		err := tx.QueryTable(new(models.AgentSkill)).Filter("name", version.Name).One(&skill)
		if err != nil && err != orm.ErrNoRows {
			return err
		}
		if err == nil && skill.Type != SkillTypePortable {
			return fmt.Errorf("portable skill %q conflicts with existing %s skill", version.Name, skill.Type)
		}
		if err == nil && skill.Deleted == 1 {
			skill.Deleted = 0
			skill.Enabled = 0
			skill.UpdatedAt = now
			if _, updateErr := tx.Update(&skill, "Deleted", "Enabled", "UpdatedAt"); updateErr != nil {
				return updateErr
			}
		}
		if err == orm.ErrNoRows {
			skill = models.AgentSkill{
				Name: version.Name, DisplayName: version.Name, Description: version.Description,
				Type: SkillTypePortable, Enabled: 0, CreatedAt: now, UpdatedAt: now,
			}
			id, insertErr := tx.Insert(&skill)
			if insertErr != nil {
				return insertErr
			}
			skill.ID = id
		}
		version.SkillID = skill.ID
		id, insertErr := tx.Insert(&version)
		if insertErr != nil {
			return insertErr
		}
		version.ID = id
		seen := map[string]bool{}
		for _, name := range requested {
			name = strings.TrimSpace(name)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			row := models.AgentSkillPermission{
				SkillID: skill.ID, VersionID: version.ID, RequestedName: name,
				Status: "requested", Granted: 0, CreatedAt: now, UpdatedAt: now,
			}
			if _, insertErr := tx.Insert(&row); insertErr != nil {
				return insertErr
			}
		}
		if activate {
			skill.ActiveVersionID = version.ID
			skill.Description = version.Description
			skill.Enabled = 1
			skill.UpdatedAt = now
			if _, updateErr := tx.Update(&skill, "ActiveVersionID", "Description", "Enabled", "UpdatedAt"); updateErr != nil {
				return updateErr
			}
		}
		result = ImportResult{Skill: skill, Version: version}
		return nil
	})
	return result, err
}

func (s Store) ListVersions(ctx context.Context, skillID int64) ([]models.AgentSkillVersion, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows := []models.AgentSkillVersion{}
	_, err := s.orm().QueryTable(new(models.AgentSkillVersion)).Filter("skill_id", skillID).OrderBy("-created_at").All(&rows)
	return rows, err
}
func (s Store) Permissions(ctx context.Context, versionID int64) ([]models.AgentSkillPermission, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows := []models.AgentSkillPermission{}
	_, err := s.orm().QueryTable(new(models.AgentSkillPermission)).Filter("version_id", versionID).OrderBy("requested_name").All(&rows)
	return rows, err
}
func (s Store) Permission(ctx context.Context, id int64) (*models.AgentSkillPermission, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	row := &models.AgentSkillPermission{ID: id}
	if err := s.orm().Read(row); err != nil {
		return nil, err
	}
	return row, nil
}
func (s Store) SavePermissionReview(ctx context.Context, row *models.AgentSkillPermission) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if row == nil || row.ID <= 0 {
		return fmt.Errorf("permission is required")
	}
	row.UpdatedAt = time.Now().UTC().UnixMilli()
	_, err := s.orm().Update(row, "ResolvedName", "Risk", "Status", "Granted", "Reason", "UpdatedAt")
	return err
}
func (s Store) Undelete(ctx context.Context, skillID int64) (*models.AgentSkill, error) {
	skill, err := s.GetSkill(ctx, skillID)
	if err != nil {
		return nil, err
	}
	if skill.Type != SkillTypePortable {
		return nil, fmt.Errorf("skill %q is not portable", skill.Name)
	}
	if skill.Deleted == 0 {
		return skill, nil
	}
	skill.Deleted = 0
	skill.Enabled = 0
	skill.UpdatedAt = time.Now().UTC().UnixMilli()
	_, err = s.orm().Update(skill, "Deleted", "Enabled", "UpdatedAt")
	return skill, err
}

func (s Store) SetPermissionGrant(ctx context.Context, id int64, granted int) (*models.AgentSkillPermission, error) {
	if granted != 0 && granted != 1 {
		return nil, fmt.Errorf("granted must be 0 or 1")
	}
	row, err := s.Permission(ctx, id)
	if err != nil {
		return nil, err
	}
	if granted == 1 && strings.TrimSpace(row.ResolvedName) == "" {
		return nil, fmt.Errorf("requested tool %q is unresolved", row.RequestedName)
	}
	row.Granted = granted
	if granted == 1 {
		row.Status = "granted"
	} else if row.Status == "granted" {
		row.Status = "review_required"
	}
	row.UpdatedAt = time.Now().UTC().UnixMilli()
	_, err = s.orm().Update(row, "Granted", "Status", "UpdatedAt")
	return row, err
}
func (s Store) Activate(ctx context.Context, versionID int64) (*models.AgentSkill, error) {
	version, err := s.Version(ctx, versionID)
	if err != nil {
		return nil, err
	}
	skill, err := s.GetSkill(ctx, version.SkillID)
	if err != nil {
		return nil, err
	}
	if skill.Type != SkillTypePortable {
		return nil, fmt.Errorf("skill %q is not portable", skill.Name)
	}
	skill.ActiveVersionID = version.ID
	skill.Description = version.Description
	skill.Enabled = 1
	skill.Deleted = 0
	skill.UpdatedAt = time.Now().UTC().UnixMilli()
	_, err = s.orm().Update(skill, "ActiveVersionID", "Description", "Enabled", "Deleted", "UpdatedAt")
	return skill, err
}

func (s Store) ActiveVersions(ctx context.Context) ([]models.AgentSkillVersion, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var skills []models.AgentSkill
	if _, err := s.orm().QueryTable(new(models.AgentSkill)).Filter("type", SkillTypePortable).Filter("enabled", 1).Filter("deleted", 0).Filter("active_version_id__gt", 0).All(&skills); err != nil {
		return nil, err
	}
	if len(skills) == 0 {
		return []models.AgentSkillVersion{}, nil
	}
	ids := make([]interface{}, 0, len(skills))
	for _, skill := range skills {
		ids = append(ids, skill.ActiveVersionID)
	}
	var versions []models.AgentSkillVersion
	_, err := s.orm().QueryTable(new(models.AgentSkillVersion)).Filter("id__in", ids...).All(&versions)
	return versions, err
}
func (s Store) GrantedToolNames(ctx context.Context, skillName string) ([]string, error) {
	skill, err := s.GetSkillByName(ctx, skillName)
	if err == orm.ErrNoRows {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	if skill.Type != SkillTypePortable || skill.ActiveVersionID <= 0 {
		return []string{}, nil
	}
	var rows []models.AgentSkillPermission
	if _, err := s.orm().QueryTable(new(models.AgentSkillPermission)).Filter("version_id", skill.ActiveVersionID).Filter("granted", 1).All(&rows); err != nil {
		return nil, err
	}
	out := []string{}
	seen := map[string]bool{}
	for _, row := range rows {
		name := strings.TrimSpace(row.ResolvedName)
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out, nil
}

func PackageRoot(version models.AgentSkillVersion) (string, error) {
	return safeJoin(DataDir(), version.PackagePath)
}
func (s Store) Files(ctx context.Context, versionID int64) ([]string, error) {
	version, err := s.Version(ctx, versionID)
	if err != nil {
		return nil, err
	}
	root, err := PackageRoot(*version)
	if err != nil {
		return nil, err
	}
	pkg, err := ParseInstalledPackage(root, version.Name)
	if err != nil {
		return nil, err
	}
	return pkg.Files, nil
}
func (s Store) ReadFile(ctx context.Context, versionID int64, relative string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	version, err := s.Version(ctx, versionID)
	if err != nil {
		return "", err
	}
	root, err := PackageRoot(*version)
	if err != nil {
		return "", err
	}
	path, err := safeJoin(root, relative)
	if err != nil {
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
func (s Store) Detail(ctx context.Context, versionID int64) (VersionDetail, error) {
	v, err := s.Version(ctx, versionID)
	if err != nil {
		return VersionDetail{}, err
	}
	p, err := s.Permissions(ctx, versionID)
	if err != nil {
		return VersionDetail{}, err
	}
	f, err := s.Files(ctx, versionID)
	if err != nil {
		return VersionDetail{}, err
	}
	return VersionDetail{Version: *v, Permissions: p, Files: f}, nil
}

func decodeRequested(raw string) []string {
	var out []string
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}
func VersionPath(version models.AgentSkillVersion) string {
	return filepath.Join(DataDir(), filepath.FromSlash(version.PackagePath))
}
