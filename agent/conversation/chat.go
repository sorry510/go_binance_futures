package conversation

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"go_binance_futures/agent/contextengine"
	"go_binance_futures/agent/security"
	"go_binance_futures/agent/task"
	"go_binance_futures/llm"
	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

var appendOnceMu sync.Mutex

const (
	ChatSkill    = "chat"
	DefaultTitle = "新对话"
)

type ListOptions struct {
	Page  int
	Limit int
}

type ListResult struct {
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
	Total int64          `json:"total"`
	List  []Conversation `json:"list"`
}
type MessageView struct {
	ID             int64  `json:"id"`
	ConversationID string `json:"conversation_id"`
	TaskID         string `json:"task_id,omitempty"`
	Skill          string `json:"skill,omitempty"`
	Sequence       int    `json:"sequence"`
	Role           string `json:"role"`
	Content        string `json:"content"`
	CreatedAt      int64  `json:"created_at"`
	TaskStatus     string `json:"task_status,omitempty"`
	TaskStage      string `json:"task_stage,omitempty"`
	TaskError      string `json:"task_error,omitempty"`
}

func (store *ORMStore) ListChats(ctx context.Context, options ListOptions) (ListResult, error) {
	if err := ctx.Err(); err != nil {
		return ListResult{}, err
	}
	page, limit := normalizeChatList(options.Page, options.Limit)
	query := store.orm().QueryTable(new(models.AgentConversation)).Filter("skill", ChatSkill)
	total, err := query.Count()
	if err != nil {
		return ListResult{}, err
	}
	var rows []models.AgentConversation
	if _, err := query.OrderBy("-updated_at").Limit(limit, (page-1)*limit).All(&rows); err != nil {
		return ListResult{}, err
	}
	items := make([]Conversation, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromModel(row))
	}
	return ListResult{Page: page, Limit: limit, Total: total, List: items}, nil
}

func (store *ORMStore) SetTitle(ctx context.Context, id, title string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	title = truncateTitle(title)
	if title == "" {
		title = DefaultTitle
	}
	_, err := store.orm().QueryTable(new(models.AgentConversation)).Filter("id", strings.TrimSpace(id)).Update(orm.Params{
		"title": title, "updated_at": time.Now().UTC().UnixMilli(),
	})
	return err
}

func (store *ORMStore) DeleteChat(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("conversation id is required")
	}
	o := store.orm()
	var row models.AgentConversation
	if err := o.QueryTable(new(models.AgentConversation)).Filter("id", id).One(&row); err != nil {
		if err == orm.ErrNoRows {
			return fmt.Errorf("chat conversation %q not found", id)
		}
		return err
	}
	if row.Skill != ChatSkill {
		return fmt.Errorf("conversation %q is not a chat conversation", id)
	}
	runningStatuses := []interface{}{
		string(task.StatusQueued), string(task.StatusRunning), string(task.StatusWaitingLLM),
		string(task.StatusWaitingTool), string(task.StatusValidating),
	}
	if o.QueryTable(new(models.AgentTask)).Filter("conversation_id", id).Filter("status__in", runningStatuses...).Exist() {
		return fmt.Errorf("chat conversation has a running task")
	}
	tx, err := o.Begin()
	if err != nil {
		return fmt.Errorf("begin delete chat conversation transaction: %w", err)
	}
	if _, err := tx.QueryTable(new(models.AgentConversationMessage)).Filter("conversation_id", id).Delete(); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("delete chat conversation messages: %w", err)
	}
	deleted, err := tx.QueryTable(new(models.AgentConversation)).Filter("id", id).Filter("skill", ChatSkill).Delete()
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("delete chat conversation: %w", err)
	}
	if deleted == 0 {
		_ = tx.Rollback()
		return fmt.Errorf("chat conversation %q not found", id)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete chat conversation: %w", err)
	}
	return nil
}

func (store *ORMStore) SetTitleFromFirstMessage(ctx context.Context, id, content string) error {
	conversation, err := store.Get(ctx, id)
	if err != nil {
		return err
	}
	if strings.TrimSpace(conversation.Title) != "" && conversation.Title != DefaultTitle {
		return nil
	}
	return store.SetTitle(ctx, id, content)
}
func (store *ORMStore) AppendOnce(ctx context.Context, conversationID, taskID, skillName string, message llm.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	appendOnceMu.Lock()
	defer appendOnceMu.Unlock()
	conversationID = strings.TrimSpace(conversationID)
	taskID = strings.TrimSpace(taskID)
	role := strings.TrimSpace(message.Role)
	content := strings.TrimSpace(message.Content)
	if conversationID == "" || taskID == "" || role == "" || content == "" {
		return fmt.Errorf("conversation_id, task_id, role and content are required")
	}
	o := store.orm()
	if o.QueryTable(new(models.AgentConversationMessage)).Filter("conversation_id", conversationID).Filter("task_id", taskID).Filter("role", role).Exist() {
		return nil
	}
	count, err := o.QueryTable(new(models.AgentConversationMessage)).Filter("conversation_id", conversationID).Count()
	if err != nil {
		return err
	}
	now := time.Now().UTC().UnixMilli()
	row := models.AgentConversationMessage{ConversationID: conversationID, TaskID: taskID, Skill: strings.TrimSpace(skillName), Sequence: int(count) + 1, Role: role, Content: security.RedactText(content), CreatedAt: now}
	if _, err := o.Insert(&row); err != nil {
		if o.QueryTable(new(models.AgentConversationMessage)).Filter("conversation_id", conversationID).Filter("task_id", taskID).Filter("role", role).Exist() {
			return nil
		}
		return err
	}
	_, err = o.QueryTable(new(models.AgentConversation)).Filter("id", conversationID).Update(orm.Params{"updated_at": now})
	return err
}
func (store *ORMStore) MessagesDetailed(ctx context.Context, id string) ([]MessageView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var rows []models.AgentConversationMessage
	if _, err := store.orm().QueryTable(new(models.AgentConversationMessage)).Filter("conversation_id", strings.TrimSpace(id)).OrderBy("sequence").All(&rows); err != nil {
		return nil, err
	}
	taskIDs := make([]interface{}, 0, len(rows))
	seen := map[string]bool{}
	for _, row := range rows {
		if row.TaskID != "" && !seen[row.TaskID] {
			seen[row.TaskID] = true
			taskIDs = append(taskIDs, row.TaskID)
		}
	}
	tasks := map[string]models.AgentTask{}
	if len(taskIDs) > 0 {
		var taskRows []models.AgentTask
		if _, err := store.orm().QueryTable(new(models.AgentTask)).Filter("id__in", taskIDs...).All(&taskRows); err != nil {
			return nil, err
		}
		for _, row := range taskRows {
			tasks[row.ID] = row
		}
	}
	result := make([]MessageView, 0, len(rows))
	for _, row := range rows {
		item := MessageView{ID: row.ID, ConversationID: row.ConversationID, TaskID: row.TaskID, Skill: row.Skill, Sequence: row.Sequence, Role: row.Role, Content: row.Content, CreatedAt: row.CreatedAt}
		if linked, ok := tasks[row.TaskID]; ok {
			item.TaskStatus, item.TaskStage, item.TaskError = linked.Status, linked.Stage, linked.Error
		}
		result = append(result, item)
	}
	return result, nil
}

func (store *ORMStore) SuccessfulHistory(ctx context.Context, conversationID, currentTaskID string, limit int) ([]contextengine.ContextBlock, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 30 {
		limit = 30
	}
	var succeeded []models.AgentTask
	query := store.orm().QueryTable(new(models.AgentTask)).Filter("conversation_id", strings.TrimSpace(conversationID)).Filter("status", string(task.StatusSucceeded))
	if currentTaskID = strings.TrimSpace(currentTaskID); currentTaskID != "" {
		query = query.Exclude("id", currentTaskID)
	}
	if _, err := query.OrderBy("-created_at").Limit(30).All(&succeeded); err != nil {
		return nil, err
	}
	if len(succeeded) == 0 {
		return []contextengine.ContextBlock{}, nil
	}
	ids := make([]interface{}, 0, len(succeeded))
	for _, row := range succeeded {
		ids = append(ids, row.ID)
	}
	var messages []models.AgentConversationMessage
	if _, err := store.orm().QueryTable(new(models.AgentConversationMessage)).Filter("conversation_id", strings.TrimSpace(conversationID)).Filter("task_id__in", ids...).OrderBy("-sequence").Limit(limit).All(&messages); err != nil {
		return nil, err
	}
	sort.Slice(messages, func(i, j int) bool { return messages[i].Sequence < messages[j].Sequence })
	blocks := make([]contextengine.ContextBlock, 0, len(messages))
	for _, row := range messages {
		blocks = append(blocks, contextengine.ContextBlock{
			ID: fmt.Sprintf("conversation-%s-%06d", conversationID, row.Sequence), Type: contextengine.BlockHistory,
			Source: "conversation:" + strings.TrimSpace(conversationID), Role: row.Role,
			Priority: contextengine.DefaultPriority(contextengine.BlockHistory), Freshness: contextengine.FreshnessUnknown,
			Content: row.Content,
		})
	}
	return blocks, nil
}

func normalizeChatList(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

func truncateTitle(value string) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	runes := []rune(value)
	if len(runes) > 40 {
		runes = runes[:40]
	}
	return string(runes)
}
