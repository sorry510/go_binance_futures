package controllers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	agentapp "go_binance_futures/agent/app"
	"go_binance_futures/agent/observability"
	"go_binance_futures/agent/portableskill"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/server/web"
)

type AgentPortableSkillController struct{ web.Controller }

func portableID(value string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("id 无效")
	}
	return id, nil
}
func parseActivate(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "1" || value == "true" || value == "yes"
}

func (ctrl *AgentPortableSkillController) Import() {
	if ctrl.Ctx.Request.ContentLength > (16<<20)+1024*1024 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "Skill 上传文件过大"))
		return
	}
	file, header, err := ctrl.GetFile("file")
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请选择 ZIP 或 SKILL.md 文件: "+err.Error()))
		return
	}
	defer file.Close()
	activate := parseActivate(ctrl.GetString("activate"))
	result, err := (portableskill.Importer{}).ImportFile(ctrl.Ctx.Request.Context(), header.Filename, file, header.Size, false)
	if err != nil {
		observability.RecordChange(ctrl.Ctx.Request.Context(), observability.ChangeEvent{Category: "skill", EntityType: "skill_import", EntityName: header.Filename, ChangeType: "import_validation_failed", Status: "error", Detail: map[string]any{"error": err.Error()}})
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	if err := agentapp.ReviewPortableSkillPermissions(ctrl.Ctx.Request.Context(), result.Version.ID); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "Skill 已导入但权限解析失败: "+err.Error()))
		return
	}
	if activate {
		activatedSkill, activateErr := (portableskill.Store{}).Activate(ctrl.Ctx.Request.Context(), result.Version.ID)
		err = activateErr
		if err == nil && activatedSkill != nil {
			result.Skill = *activatedSkill
		}
		if err != nil {
			ctrl.Ctx.Resp(utils.ResJson(400, nil, "Skill 已导入但激活失败: "+err.Error()))
			return
		}
	}
	if result.Skill.ActiveVersionID == result.Version.ID {
		if err := agentapp.SyncDefaultPortableSkills(ctrl.Ctx.Request.Context()); err != nil {
			ctrl.Ctx.Resp(utils.ResJson(500, nil, "Skill 已导入但运行时同步失败: "+err.Error()))
			return
		}
	}
	detail, _ := (portableskill.Store{}).Detail(ctrl.Ctx.Request.Context(), result.Version.ID)
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": map[string]any{"skill": result.Skill, "version": detail.Version, "permissions": detail.Permissions, "files": detail.Files, "duplicate": result.Duplicate}, "msg": "success"})
}

type portableDirectoryRequest struct {
	Path     string `json:"path"`
	Activate bool   `json:"activate"`
}

func (ctrl *AgentPortableSkillController) ImportDirectory() {
	var request portableDirectoryRequest
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	result, err := (portableskill.Importer{}).ImportDirectory(ctrl.Ctx.Request.Context(), request.Path, false)
	if err != nil {
		observability.RecordChange(ctrl.Ctx.Request.Context(), observability.ChangeEvent{Category: "skill", EntityType: "skill_import", EntityName: request.Path, ChangeType: "import_validation_failed", Status: "error", Detail: map[string]any{"error": err.Error()}})
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	if err := agentapp.ReviewPortableSkillPermissions(ctrl.Ctx.Request.Context(), result.Version.ID); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	if request.Activate {
		activatedSkill, activateErr := (portableskill.Store{}).Activate(ctrl.Ctx.Request.Context(), result.Version.ID)
		err = activateErr
		if err == nil && activatedSkill != nil {
			result.Skill = *activatedSkill
		}
		if err != nil {
			ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
			return
		}
	}
	if result.Skill.ActiveVersionID == result.Version.ID {
		if err := agentapp.SyncDefaultPortableSkills(ctrl.Ctx.Request.Context()); err != nil {
			ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
			return
		}
	}
	detail, _ := (portableskill.Store{}).Detail(ctrl.Ctx.Request.Context(), result.Version.ID)
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": map[string]any{"skill": result.Skill, "version": detail.Version, "permissions": detail.Permissions, "files": detail.Files, "duplicate": result.Duplicate}, "msg": "success"})
}

func (ctrl *AgentPortableSkillController) Versions() {
	skillID, err := portableID(ctrl.Ctx.Input.Param(":id"))
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	rows, err := (portableskill.Store{}).ListVersions(ctrl.Ctx.Request.Context(), skillID)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": rows, "msg": "success"})
}
func (ctrl *AgentPortableSkillController) VersionDetail() {
	versionID, err := portableID(ctrl.Ctx.Input.Param(":versionId"))
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	detail, err := (portableskill.Store{}).Detail(ctrl.Ctx.Request.Context(), versionID)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(404, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": detail, "msg": "success"})
}
func (ctrl *AgentPortableSkillController) ReadVersionFile() {
	versionID, err := portableID(ctrl.Ctx.Input.Param(":versionId"))
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	content, err := (portableskill.Store{}).ReadFile(ctrl.Ctx.Request.Context(), versionID, ctrl.GetString("path"))
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": map[string]string{"path": ctrl.GetString("path"), "content": content}, "msg": "success"})
}
func (ctrl *AgentPortableSkillController) Activate() {
	versionID, err := portableID(ctrl.Ctx.Input.Param(":versionId"))
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	if err := agentapp.ReviewPortableSkillPermissions(ctrl.Ctx.Request.Context(), versionID); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	skillRow, err := (portableskill.Store{}).Activate(ctrl.Ctx.Request.Context(), versionID)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	if err := agentapp.SyncDefaultPortableSkills(ctrl.Ctx.Request.Context()); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": skillRow, "msg": "success"})
}

type portablePermissionRequest struct {
	Granted int `json:"granted"`
}

func (ctrl *AgentPortableSkillController) UpdatePermission() {
	id, err := portableID(ctrl.Ctx.Input.Param(":id"))
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	var request portablePermissionRequest
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	row, err := agentapp.SetPortableSkillPermission(ctrl.Ctx.Request.Context(), id, request.Granted)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": row, "msg": "success"})
}
