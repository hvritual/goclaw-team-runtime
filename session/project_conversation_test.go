package session

import (
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestProjectConversationKey(t *testing.T) {
	tests := []struct {
		name      string
		projectID string
		topicID   string
		wantKey   string
		wantTopic string
		wantError bool
	}{
		{
			name:      "default topic",
			projectID: "project-alpha",
			wantKey:   "project-v2.cHJvamVjdC1hbHBoYQ.aW5ib3g",
			wantTopic: "inbox",
		},
		{
			name:      "explicit topic",
			projectID: "project-alpha",
			topicID:   "release_1.2",
			wantKey:   "project-v2.cHJvamVjdC1hbHBoYQ.cmVsZWFzZV8xLjI",
			wantTopic: "release_1.2",
		},
		{
			name:      "rejects delimiter collision",
			projectID: "project-alpha",
			topicID:   "topic:other",
			wantError: true,
		},
		{
			name:      "rejects traversal",
			projectID: "../alpha",
			topicID:   "inbox",
			wantError: true,
		},
		{
			name:      "rejects filename overflow",
			projectID: strings.Repeat("a", 65),
			topicID:   "inbox",
			wantError: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			key, topic, err := ProjectConversationKey(test.projectID, test.topicID)
			if test.wantError {
				if err == nil {
					t.Fatalf("ProjectConversationKey() error = nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if key != test.wantKey || topic != test.wantTopic {
				t.Fatalf(
					"ProjectConversationKey() = (%q, %q), want (%q, %q)",
					key,
					topic,
					test.wantKey,
					test.wantTopic,
				)
			}
		})
	}
}

func TestProjectConversationCollisionPairHasDifferentKeysAndPaths(t *testing.T) {
	manager, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	first, _, err := ProjectConversationKey("alpha", "beta_gamma")
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := ProjectConversationKey("alpha_beta", "gamma")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("collision pair returned the same key %q", first)
	}
	if manager.sessionPath(first) == manager.sessionPath(second) {
		t.Fatalf(
			"collision pair returned the same path %q",
			manager.sessionPath(first),
		)
	}
}

func TestGetExistingProjectConversationDoesNotCreateMissing(t *testing.T) {
	manager, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if value, topic, found, err := manager.GetExistingProjectConversation(
		"alpha",
		"missing",
	); err != nil {
		t.Fatal(err)
	} else if found || value != nil || topic != "missing" {
		t.Fatalf(
			"missing conversation returned value=%v topic=%q found=%v",
			value,
			topic,
			found,
		)
	}
	if keys, err := manager.List(); err != nil {
		t.Fatal(err)
	} else if len(keys) != 0 {
		t.Fatalf("history read created sessions: %v", keys)
	}

	conversation, _, err := manager.GetOrCreateProjectConversation(
		"alpha",
		"inbox",
	)
	if err != nil {
		t.Fatal(err)
	}
	conversation.AddMessage(Message{
		Role:      "user",
		Content:   "hello",
		Timestamp: time.Now().UTC(),
	})
	if err := manager.Save(conversation); err != nil {
		t.Fatal(err)
	}
	if value, _, found, err := manager.GetExistingProjectConversation(
		"alpha",
		"inbox",
	); err != nil {
		t.Fatal(err)
	} else if !found || len(value.GetHistory(0)) != 1 {
		t.Fatalf("saved conversation returned value=%v found=%v", value, found)
	}
}

func TestProjectConversationMigratesUnambiguousLegacyFile(t *testing.T) {
	root := t.TempDir()
	legacyManager, err := NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	legacyKey := legacyProjectConversationKey("alpha", "inbox")
	legacy, err := legacyManager.GetOrCreate(legacyKey)
	if err != nil {
		t.Fatal(err)
	}
	legacy.MergeMetadata(map[string]interface{}{
		"project_id": "alpha",
		"topic_id":   "inbox",
	})
	legacy.AddMessage(Message{
		Role:      "user",
		Content:   "legacy message",
		Timestamp: time.Now().UTC(),
	})
	if err := legacyManager.Save(legacy); err != nil {
		t.Fatal(err)
	}
	oldPath := legacyManager.sessionPath(legacyKey)

	manager, err := NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	conversation, topic, found, err := manager.GetExistingProjectConversation(
		"alpha",
		"inbox",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !found || topic != "inbox" || conversation == nil {
		t.Fatalf(
			"migration returned conversation=%v topic=%q found=%v",
			conversation,
			topic,
			found,
		)
	}
	newKey, _, err := ProjectConversationKey("alpha", "inbox")
	if err != nil {
		t.Fatal(err)
	}
	if conversation.Key != newKey ||
		len(conversation.GetHistory(0)) != 1 {
		t.Fatalf("unexpected migrated conversation: %#v", conversation)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("legacy path still exists after migration: %v", err)
	}
	if _, err := os.Stat(manager.sessionPath(newKey)); err != nil {
		t.Fatalf("new path does not exist after migration: %v", err)
	}
}

func TestProjectConversationRejectsAmbiguousLegacyFile(t *testing.T) {
	root := t.TempDir()
	legacyManager, err := NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	legacyKey := legacyProjectConversationKey("alpha_beta", "gamma")
	legacy, err := legacyManager.GetOrCreate(legacyKey)
	if err != nil {
		t.Fatal(err)
	}
	legacy.MergeMetadata(map[string]interface{}{
		"project_id": "alpha_beta",
		"topic_id":   "gamma",
	})
	legacy.AddMessage(Message{
		Role:      "user",
		Content:   "must not guess",
		Timestamp: time.Now().UTC(),
	})
	if err := legacyManager.Save(legacy); err != nil {
		t.Fatal(err)
	}

	manager, err := NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := manager.GetExistingProjectConversation(
		"alpha_beta",
		"gamma",
	); !errors.Is(err, ErrAmbiguousLegacyProjectConversation) {
		t.Fatalf("ambiguous migration error = %v", err)
	}
	if _, err := os.Stat(manager.sessionPath(legacyKey)); err != nil {
		t.Fatalf("ambiguous legacy path was modified: %v", err)
	}
}

func TestProjectConversationRejectsLegacyScopeMismatch(t *testing.T) {
	root := t.TempDir()
	legacyManager, err := NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	legacyKey := legacyProjectConversationKey("alpha", "inbox")
	legacy, err := legacyManager.GetOrCreate(legacyKey)
	if err != nil {
		t.Fatal(err)
	}
	legacy.MergeMetadata(map[string]interface{}{
		"project_id": "other",
		"topic_id":   "inbox",
	})
	if err := legacyManager.Save(legacy); err != nil {
		t.Fatal(err)
	}

	manager, err := NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := manager.GetExistingProjectConversation(
		"alpha",
		"inbox",
	); !errors.Is(err, ErrAmbiguousLegacyProjectConversation) {
		t.Fatalf("scope mismatch error = %v", err)
	}
}

func TestProjectConversationConcurrentReadsUseOneScopedSession(t *testing.T) {
	manager, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	conversation, _, err := manager.GetOrCreateProjectConversation(
		"alpha",
		"inbox",
	)
	if err != nil {
		t.Fatal(err)
	}
	conversation.AddMessage(Message{
		Role:      "assistant",
		Content:   "ready",
		Timestamp: time.Now().UTC(),
	})
	if err := manager.Save(conversation); err != nil {
		t.Fatal(err)
	}

	const readers = 32
	var wait sync.WaitGroup
	errs := make(chan error, readers)
	wait.Add(readers)
	for range readers {
		go func() {
			defer wait.Done()
			value, topic, found, err := manager.GetExistingProjectConversation(
				"alpha",
				"inbox",
			)
			if err != nil {
				errs <- err
				return
			}
			if !found || topic != "inbox" || value != conversation {
				errs <- errors.New("reader observed a different conversation")
			}
		}()
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
