package gateway

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/smallnest/goclaw/session"
)

func TestTeamChatHistoryIsProjectScopedAndRedacted(t *testing.T) {
	fixture := newGatewayTeamFixture(t)
	manager, err := session.NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	conversation, _, err := manager.GetOrCreateProjectConversation(
		fixture.project.ID,
		"inbox",
	)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	conversation.AddMessage(session.Message{
		Role:      "system",
		Content:   "private-system-prompt",
		Timestamp: now,
	})
	conversation.AddMessage(session.Message{
		Role:      "user",
		Content:   "shared question",
		Timestamp: now.Add(time.Second),
		Metadata:  map[string]interface{}{"secret": "do-not-return"},
	})
	conversation.AddMessage(session.Message{
		Role:       "tool",
		Content:    "private-tool-result",
		Timestamp:  now.Add(2 * time.Second),
		ToolCallID: "tool-secret",
	})
	conversation.AddMessage(session.Message{
		Role:      "assistant",
		Content:   "shared answer",
		Timestamp: now.Add(3 * time.Second),
	})
	if err := manager.Save(conversation); err != nil {
		t.Fatal(err)
	}

	handler := &Handler{
		registry:   NewMethodRegistry(),
		sessionMgr: manager,
	}
	handler.registerAgentMethods()
	handler.SetTeamControlService(&fixture.service)
	response := handler.HandleRequest(
		teamSessionID(fixture.bob.ID),
		&JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      "history",
			Method:  "chat.history",
			Params: map[string]interface{}{
				"project_id": fixture.project.ID,
				"topic_id":   "inbox",
				"key":        "attacker-controlled",
				"limit":      float64(1),
			},
		},
	)
	if response.Error != nil {
		t.Fatalf("chat.history failed: %+v", response.Error)
	}
	encoded, err := json.Marshal(response.Result)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	if !strings.Contains(text, "shared answer") ||
		strings.Contains(text, "shared question") ||
		strings.Contains(text, "private-system-prompt") ||
		strings.Contains(text, "private-tool-result") ||
		strings.Contains(text, "do-not-return") {
		t.Fatalf("unexpected filtered history: %s", text)
	}

	denied := handler.HandleRequest(
		teamSessionID(fixture.viewer.ID),
		&JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      "cross-project-history",
			Method:  "chat.history",
			Params: map[string]interface{}{
				"project_id": fixture.project.ID,
				"topic_id":   "inbox",
			},
		},
	)
	if denied.Error == nil {
		t.Fatal("cross-project chat history was allowed")
	}
}

func TestTeamChatHistoryAuthorizesBeforeLegacyMigration(t *testing.T) {
	fixture := newGatewayTeamFixture(t)
	root := t.TempDir()
	legacyManager, err := session.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	legacyKey := "project:" + fixture.project.ID + ":inbox"
	legacy, err := legacyManager.GetOrCreate(legacyKey)
	if err != nil {
		t.Fatal(err)
	}
	legacy.MergeMetadata(map[string]interface{}{
		"project_id": fixture.project.ID,
		"topic_id":   "inbox",
	})
	legacy.AddMessage(session.Message{
		Role:      "user",
		Content:   "authorized readers only",
		Timestamp: time.Now().UTC(),
	})
	if err := legacyManager.Save(legacy); err != nil {
		t.Fatal(err)
	}

	manager, err := session.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	handler := &Handler{
		registry:   NewMethodRegistry(),
		sessionMgr: manager,
	}
	handler.registerAgentMethods()
	handler.SetTeamControlService(&fixture.service)
	response := handler.HandleRequest(
		teamSessionID(fixture.viewer.ID),
		&JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      "unauthorized-migration",
			Method:  "chat.history",
			Params: map[string]interface{}{
				"project_id": fixture.project.ID,
				"topic_id":   "inbox",
			},
		},
	)
	if response.Error == nil {
		t.Fatal("unauthorized history request was allowed")
	}
	keys, err := manager.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0] != strings.NewReplacer(":", "_").Replace(legacyKey) {
		t.Fatalf("authorization failure changed conversation storage: %v", keys)
	}
}

func TestTeamChatHistoryRejectsAmbiguousLegacyConversation(t *testing.T) {
	fixture := newGatewayTeamFixture(t)
	root := t.TempDir()
	legacyManager, err := session.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	legacyKey := "project:" + fixture.project.ID + ":release_one"
	legacy, err := legacyManager.GetOrCreate(legacyKey)
	if err != nil {
		t.Fatal(err)
	}
	legacy.MergeMetadata(map[string]interface{}{
		"project_id": fixture.project.ID,
		"topic_id":   "release_one",
	})
	if err := legacyManager.Save(legacy); err != nil {
		t.Fatal(err)
	}

	manager, err := session.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	handler := &Handler{
		registry:   NewMethodRegistry(),
		sessionMgr: manager,
	}
	handler.registerAgentMethods()
	handler.SetTeamControlService(&fixture.service)
	response := handler.HandleRequest(
		teamSessionID(fixture.alice.ID),
		&JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      "ambiguous-legacy",
			Method:  "chat.history",
			Params: map[string]interface{}{
				"project_id": fixture.project.ID,
				"topic_id":   "release_one",
			},
		},
	)
	if response.Error == nil ||
		!strings.Contains(
			response.Error.Message,
			session.ErrAmbiguousLegacyProjectConversation.Error(),
		) {
		t.Fatalf("ambiguous legacy history was not rejected: %+v", response)
	}
	_, _, _, directErr := manager.GetExistingProjectConversation(
		fixture.project.ID,
		"release_one",
	)
	if !errors.Is(directErr, session.ErrAmbiguousLegacyProjectConversation) {
		t.Fatal("ambiguous legacy sentinel was not preserved")
	}
}

func TestTeamChatHistoryRejectsAmbiguousTopic(t *testing.T) {
	fixture := newGatewayTeamFixture(t)
	manager, err := session.NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	handler := &Handler{
		registry:   NewMethodRegistry(),
		sessionMgr: manager,
	}
	handler.registerAgentMethods()
	handler.SetTeamControlService(&fixture.service)

	response := handler.HandleRequest(
		teamSessionID(fixture.alice.ID),
		&JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      "ambiguous-topic",
			Method:  "chat.history",
			Params: map[string]interface{}{
				"project_id": fixture.project.ID,
				"topic_id":   "inbox:project-other",
			},
		},
	)
	if response.Error == nil {
		t.Fatal("ambiguous topic was allowed")
	}
}
