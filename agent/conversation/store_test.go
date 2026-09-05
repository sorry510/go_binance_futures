package conversation

import (
	"context"
	"strings"
	"sync"
	"testing"

	"go_binance_futures/llm"
	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

var conversationStoreTestOnce sync.Once
var conversationStoreTestErr error

func setupConversationStoreTest(t *testing.T) *ORMStore {
	t.Helper()
	conversationStoreTestOnce.Do(func() {
		conversationStoreTestErr = orm.RegisterDriver("sqlite3", orm.DRSqlite)
		if conversationStoreTestErr != nil {
			return
		}
		orm.RegisterModel(new(models.AgentConversation), new(models.AgentConversationMessage), new(models.AgentTask))
		conversationStoreTestErr = orm.RegisterDataBase("agent_conversation_test", "sqlite3", "file:agent_conversation_test?mode=memory&cache=shared")
		if conversationStoreTestErr != nil {
			return
		}
		conversationStoreTestErr = orm.RunSyncdb("agent_conversation_test", true, false)
	})
	if conversationStoreTestErr != nil {
		t.Fatal(conversationStoreTestErr)
	}
	return &ORMStore{Alias: "agent_conversation_test"}
}

func TestORMStorePersistsConversationAcrossStoreInstances(t *testing.T) {
	store := setupConversationStoreTest(t)
	ctx := context.Background()
	conversation, err := store.Create(ctx, "strategy_builder")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Append(ctx, conversation.ID, "task-1", llm.Message{Role: llm.RoleUser, Content: "api_key=secret-value first prompt"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(ctx, conversation.ID, "task-1", llm.Message{Role: llm.RoleAssistant, Content: "first answer"}); err != nil {
		t.Fatal(err)
	}

	reloaded := &ORMStore{Alias: "agent_conversation_test"}
	got, err := reloaded.Get(ctx, conversation.ID)
	if err != nil {
		t.Fatal(err)
	}
	messages, err := reloaded.Messages(ctx, conversation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != conversation.ID || got.Status != StatusActive || len(messages) != 2 {
		t.Fatalf("unexpected reloaded conversation: %+v messages=%+v", got, messages)
	}
	if strings.Contains(messages[0].Content, "secret-value") {
		t.Fatalf("sensitive conversation content was persisted: %q", messages[0].Content)
	}
}

func TestORMStoreClosesConversation(t *testing.T) {
	store := setupConversationStoreTest(t)
	ctx := context.Background()
	conversation, err := store.Create(ctx, "strategy_builder")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(ctx, conversation.ID); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, conversation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusClosed || got.ClosedAt.IsZero() {
		t.Fatalf("conversation was not closed: %+v", got)
	}
}

func TestChatAppendOnceAndSuccessfulHistory(t *testing.T) {
	store := setupConversationStoreTest(t)
	ctx := context.Background()
	conv, err := store.Create(ctx, ChatSkill)
	if err != nil {
		t.Fatal(err)
	}
	if conv.Title != DefaultTitle {
		t.Fatalf("unexpected chat title %q", conv.Title)
	}
	o := orm.NewOrmUsingDB("agent_conversation_test")
	now := int64(1700000000000)
	for _, row := range []models.AgentTask{
		{ID: "chat-ok", Skill: "portable", ConversationID: conv.ID, Status: "succeeded", CreatedAt: now, UpdatedAt: now},
		{ID: "chat-failed", Skill: "portable", ConversationID: conv.ID, Status: "failed", CreatedAt: now + 1, UpdatedAt: now + 1},
		{ID: "chat-current", Skill: "portable", ConversationID: conv.ID, Status: "succeeded", CreatedAt: now + 2, UpdatedAt: now + 2},
	} {
		if _, err := o.Insert(&row); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.AppendOnce(ctx, conv.ID, "chat-ok", "portable", llm.Message{Role: llm.RoleUser, Content: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendOnce(ctx, conv.ID, "chat-ok", "portable", llm.Message{Role: llm.RoleAssistant, Content: "answer"}); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendOnce(ctx, conv.ID, "chat-ok", "portable", llm.Message{Role: llm.RoleAssistant, Content: "duplicate"}); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendOnce(ctx, conv.ID, "chat-failed", "portable", llm.Message{Role: llm.RoleUser, Content: "failed input"}); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendOnce(ctx, conv.ID, "chat-current", "portable", llm.Message{Role: llm.RoleUser, Content: "current input"}); err != nil {
		t.Fatal(err)
	}
	history, err := store.SuccessfulHistory(ctx, conv.ID, "chat-current", 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || history[0].Content != "first" || history[1].Content != "answer" {
		t.Fatalf("unexpected history: %+v", history)
	}
	messages, err := store.MessagesDetailed(ctx, conv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 4 {
		t.Fatalf("AppendOnce duplicated rows: %+v", messages)
	}
}
