package conversation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"go_binance_futures/agent/security"
	"go_binance_futures/llm"
	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

const (
	StatusActive = "active"
	StatusClosed = "closed"
)

type Conversation struct {
	ID        string    `json:"id"`
	Skill     string    `json:"skill"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ClosedAt  time.Time `json:"closed_at,omitempty"`
}

type Store interface {
	Create(context.Context, string) (Conversation, error)
	Get(context.Context, string) (Conversation, error)
	Messages(context.Context, string) ([]llm.Message, error)
	Append(context.Context, string, string, llm.Message) error
	Close(context.Context, string) error
}

type ORMStore struct{ Alias string }

func NewORMStore() *ORMStore { return &ORMStore{} }

func (store *ORMStore) Create(ctx context.Context, skill string) (Conversation, error) {
	if err := ctx.Err(); err != nil {
		return Conversation{}, err
	}
	skill = strings.TrimSpace(skill)
	if skill == "" {
		return Conversation{}, fmt.Errorf("conversation skill is required")
	}
	now := time.Now().UTC()
	title := ""
	if skill == ChatSkill {
		title = DefaultTitle
	}
	row := models.AgentConversation{ID: newID(), Skill: skill, Title: title, Status: StatusActive, CreatedAt: now.UnixMilli(), UpdatedAt: now.UnixMilli()}
	if _, err := store.orm().Insert(&row); err != nil {
		return Conversation{}, fmt.Errorf("insert agent conversation: %w", err)
	}
	return fromModel(row), nil
}

func (store *ORMStore) Get(ctx context.Context, id string) (Conversation, error) {
	if err := ctx.Err(); err != nil {
		return Conversation{}, err
	}
	id = strings.TrimSpace(id)
	var row models.AgentConversation
	if err := store.orm().QueryTable(new(models.AgentConversation)).Filter("id", id).One(&row); err != nil {
		if err == orm.ErrNoRows {
			return Conversation{}, fmt.Errorf("conversation %q not found", id)
		}
		return Conversation{}, err
	}
	return fromModel(row), nil
}

func (store *ORMStore) Messages(ctx context.Context, id string) ([]llm.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var rows []models.AgentConversationMessage
	if _, err := store.orm().QueryTable(new(models.AgentConversationMessage)).Filter("conversation_id", strings.TrimSpace(id)).OrderBy("sequence").All(&rows); err != nil {
		return nil, fmt.Errorf("load conversation messages: %w", err)
	}
	result := make([]llm.Message, 0, len(rows))
	for _, row := range rows {
		result = append(result, llm.Message{Role: row.Role, Content: row.Content})
	}
	return result, nil
}

func (store *ORMStore) Append(ctx context.Context, conversationID, taskID string, message llm.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" || strings.TrimSpace(message.Role) == "" || strings.TrimSpace(message.Content) == "" {
		return nil
	}
	o := store.orm()
	count, err := o.QueryTable(new(models.AgentConversationMessage)).Filter("conversation_id", conversationID).Count()
	if err != nil {
		return fmt.Errorf("count conversation messages: %w", err)
	}
	now := time.Now().UTC()
	row := models.AgentConversationMessage{
		ConversationID: conversationID, TaskID: strings.TrimSpace(taskID), Sequence: int(count) + 1,
		Role: strings.TrimSpace(message.Role), Content: security.RedactText(message.Content), CreatedAt: now.UnixMilli(),
	}
	if _, err := o.Insert(&row); err != nil {
		return fmt.Errorf("insert conversation message: %w", err)
	}
	_, err = o.QueryTable(new(models.AgentConversation)).Filter("id", conversationID).Update(orm.Params{"updated_at": now.UnixMilli()})
	return err
}

func (store *ORMStore) Close(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	now := time.Now().UTC().UnixMilli()
	updated, err := store.orm().QueryTable(new(models.AgentConversation)).Filter("id", strings.TrimSpace(id)).Update(orm.Params{
		"status": StatusClosed, "updated_at": now, "closed_at": now,
	})
	if err != nil {
		return err
	}
	if updated == 0 {
		return fmt.Errorf("conversation %q not found", id)
	}
	return nil
}

func (store *ORMStore) orm() orm.Ormer {
	if strings.TrimSpace(store.Alias) != "" {
		return orm.NewOrmUsingDB(store.Alias)
	}
	return orm.NewOrm()
}

type memoryConversation struct {
	Conversation
	messages []llm.Message
}

type MemoryStore struct {
	mu    sync.Mutex
	items map[string]*memoryConversation
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{items: map[string]*memoryConversation{}} }

func (store *MemoryStore) Create(ctx context.Context, skill string) (Conversation, error) {
	if err := ctx.Err(); err != nil {
		return Conversation{}, err
	}
	now := time.Now().UTC()
	cleanSkill := strings.TrimSpace(skill)
	title := ""
	if cleanSkill == ChatSkill {
		title = DefaultTitle
	}
	item := &memoryConversation{Conversation: Conversation{ID: newID(), Skill: cleanSkill, Title: title, Status: StatusActive, CreatedAt: now, UpdatedAt: now}}
	store.mu.Lock()
	store.items[item.ID] = item
	store.mu.Unlock()
	return item.Conversation, nil
}

func (store *MemoryStore) Get(ctx context.Context, id string) (Conversation, error) {
	if err := ctx.Err(); err != nil {
		return Conversation{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	item := store.items[strings.TrimSpace(id)]
	if item == nil {
		return Conversation{}, fmt.Errorf("conversation %q not found", id)
	}
	return item.Conversation, nil
}

func (store *MemoryStore) Messages(ctx context.Context, id string) ([]llm.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	item := store.items[strings.TrimSpace(id)]
	if item == nil {
		return nil, fmt.Errorf("conversation %q not found", id)
	}
	return append([]llm.Message(nil), item.messages...), nil
}

func (store *MemoryStore) Append(ctx context.Context, conversationID, taskID string, message llm.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	item := store.items[strings.TrimSpace(conversationID)]
	if item == nil {
		return fmt.Errorf("conversation %q not found", conversationID)
	}
	message.Content = security.RedactText(message.Content)
	item.messages = append(item.messages, message)
	item.UpdatedAt = time.Now().UTC()
	return nil
}

func (store *MemoryStore) Close(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	item := store.items[strings.TrimSpace(id)]
	if item == nil {
		return fmt.Errorf("conversation %q not found", id)
	}
	item.Status = StatusClosed
	item.UpdatedAt = time.Now().UTC()
	item.ClosedAt = item.UpdatedAt
	return nil
}

func fromModel(row models.AgentConversation) Conversation {
	item := Conversation{ID: row.ID, Skill: row.Skill, Title: row.Title, Status: row.Status, CreatedAt: time.UnixMilli(row.CreatedAt).UTC(), UpdatedAt: time.UnixMilli(row.UpdatedAt).UTC()}
	if row.ClosedAt > 0 {
		item.ClosedAt = time.UnixMilli(row.ClosedAt).UTC()
	}
	return item
}

func newID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err == nil {
		return "conv_" + hex.EncodeToString(value)
	}
	return fmt.Sprintf("conv_%d", time.Now().UnixNano())
}
