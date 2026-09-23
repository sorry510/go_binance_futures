package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	agentapp "go_binance_futures/agent/app"
	"go_binance_futures/agent/portableskill"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/server/web"
)

type AgentSkillDraftController struct{ web.Controller }

type skillDraftCreateRequest struct {
	Name            string `json:"name"`
	SourceVersionID int64  `json:"source_version_id"`
}

type skillDraftFileRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type skillDraftPublishRequest struct {
	Activate    bool `json:"activate"`
	DeleteDraft bool `json:"delete_draft"`
}

func (ctrl *AgentSkillDraftController) List() {
	rows, err := (portableskill.DraftStore{}).List(ctrl.Ctx.Request.Context())
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": rows, "msg": "success"})
}

func (ctrl *AgentSkillDraftController) Create() {
	var request skillDraftCreateRequest
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	store := portableskill.DraftStore{}
	var (
		detail portableskill.DraftDetail
		err    error
	)
	if request.SourceVersionID > 0 {
		detail, err = store.CloneVersion(ctrl.Ctx.Request.Context(), request.SourceVersionID)
	} else {
		detail, err = store.Create(ctrl.Ctx.Request.Context(), request.Name)
	}
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": detail, "msg": "success"})
}

func (ctrl *AgentSkillDraftController) Detail() {
	detail, err := (portableskill.DraftStore{}).Detail(ctrl.Ctx.Request.Context(), ctrl.draftID())
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(404, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": detail, "msg": "success"})
}

func (ctrl *AgentSkillDraftController) Delete() {
	if err := (portableskill.DraftStore{}).Delete(ctrl.Ctx.Request.Context(), ctrl.draftID()); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "msg": "success"})
}

func (ctrl *AgentSkillDraftController) ReadFile() {
	path := strings.TrimSpace(ctrl.GetString("path"))
	content, err := (portableskill.DraftStore{}).ReadFile(ctrl.Ctx.Request.Context(), ctrl.draftID(), path)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": map[string]string{"path": path, "content": content}, "msg": "success"})
}

func (ctrl *AgentSkillDraftController) WriteFile() {
	var request skillDraftFileRequest
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	if err := (portableskill.DraftStore{}).WriteFile(ctrl.Ctx.Request.Context(), ctrl.draftID(), request.Path, []byte(request.Content)); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	detail, _ := (portableskill.DraftStore{}).Detail(ctrl.Ctx.Request.Context(), ctrl.draftID())
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": detail, "msg": "success"})
}

func (ctrl *AgentSkillDraftController) DeleteFile() {
	path := strings.TrimSpace(ctrl.GetString("path"))
	if err := (portableskill.DraftStore{}).DeleteFile(ctrl.Ctx.Request.Context(), ctrl.draftID(), path); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	detail, _ := (portableskill.DraftStore{}).Detail(ctrl.Ctx.Request.Context(), ctrl.draftID())
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": detail, "msg": "success"})
}

func (ctrl *AgentSkillDraftController) UploadFile() {
	if ctrl.Ctx.Request.ContentLength > portableskill.MaxSingleFileBytes()+1024*1024 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "Draft 文件过大"))
		return
	}
	file, header, err := ctrl.GetFile("file")
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请选择文件: "+err.Error()))
		return
	}
	defer file.Close()
	path := strings.TrimSpace(ctrl.GetString("path"))
	if path == "" {
		path, err = draftUploadDefaultPath(header.Filename)
		if err != nil {
			ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
			return
		}
	}
	raw, err := io.ReadAll(io.LimitReader(file, portableskill.MaxSingleFileBytes()+1))
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	if int64(len(raw)) > portableskill.MaxSingleFileBytes() {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, fmt.Sprintf("Draft 文件超过 %d bytes", portableskill.MaxSingleFileBytes())))
		return
	}
	if err := (portableskill.DraftStore{}).WriteFile(ctrl.Ctx.Request.Context(), ctrl.draftID(), path, raw); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	detail, _ := (portableskill.DraftStore{}).Detail(ctrl.Ctx.Request.Context(), ctrl.draftID())
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": detail, "msg": "success"})
}

func (ctrl *AgentSkillDraftController) Validate() {
	result, err := (portableskill.DraftStore{}).Validate(ctrl.Ctx.Request.Context(), ctrl.draftID())
	if err != nil {
		result.Valid = false
		if result.Error == "" {
			result.Error = err.Error()
		}
		ctrl.Ctx.Resp(map[string]any{"code": 200, "data": result, "msg": "validation failed"})
		return
	}
	ctrl.Ctx.Resp(map[string]any{"code": 200, "data": result, "msg": "success"})
}

func (ctrl *AgentSkillDraftController) Publish() {
	var request skillDraftPublishRequest
	if len(ctrl.Ctx.Input.RequestBody) > 0 {
		if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
			ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
			return
		}
	}
	ctx := ctrl.Ctx.Request.Context()
	drafts := portableskill.DraftStore{}
	root, _, err := drafts.PackageRoot(ctx, ctrl.draftID())
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	if _, err := drafts.Validate(ctx, ctrl.draftID()); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "Draft 校验失败: "+err.Error()))
		return
	}
	result, err := (portableskill.Importer{}).ImportDraft(ctx, root, ctrl.draftID())
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	if err := agentapp.ReviewPortableSkillPermissions(ctx, result.Version.ID); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "Skill 已发布但权限解析失败: "+err.Error()))
		return
	}
	if request.Activate {
		activated, activateErr := (portableskill.Store{}).Activate(ctx, result.Version.ID)
		if activateErr != nil {
			ctrl.Ctx.Resp(utils.ResJson(400, nil, "Skill 已发布但激活失败: "+activateErr.Error()))
			return
		}
		if activated != nil {
			result.Skill = *activated
		}
		if err := agentapp.SyncDefaultPortableSkills(ctx); err != nil {
			ctrl.Ctx.Resp(utils.ResJson(500, nil, "Skill 已激活但运行时同步失败: "+err.Error()))
			return
		}
	}
	detail, err := (portableskill.Store{}).Detail(ctx, result.Version.ID)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, "Skill 已发布但读取版本详情失败: "+err.Error()))
		return
	}
	if request.DeleteDraft {
		_ = drafts.Delete(ctx, ctrl.draftID())
	}
	ctrl.Ctx.Resp(map[string]any{
		"code": 200,
		"data": map[string]any{
			"skill": result.Skill, "version": detail.Version, "permissions": detail.Permissions,
			"files": detail.Files, "duplicate": result.Duplicate,
		},
		"msg": "success",
	})
}

func draftUploadDefaultPath(filename string) (string, error) {
	filename = strings.TrimSpace(strings.ReplaceAll(filename, "\\", "/"))
	filename = filepath.Base(filename)
	if filename == "" || filename == "." || filename == ".." || strings.ContainsRune(filename, 0) {
		return "", fmt.Errorf("上传文件名无效")
	}
	return filename, nil
}

func (ctrl *AgentSkillDraftController) draftID() string {
	return strings.TrimSpace(ctrl.Ctx.Input.Param(":draftId"))
}
