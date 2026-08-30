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
		orm.RegisterModel(new(models.AgentConversation), new(models.AgentConversationMessage))
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
