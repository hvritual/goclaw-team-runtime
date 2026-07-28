package session

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSaveSerializesConcurrentProjectConversationWrites(t *testing.T) {
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

	const writers = 32
	var wait sync.WaitGroup
	errs := make(chan error, writers)
	wait.Add(writers)
	for index := range writers {
		go func() {
			defer wait.Done()
			conversation.AddMessage(Message{
				Role:      "user",
				Content:   fmt.Sprintf("message-%02d", index),
				Timestamp: time.Now().UTC(),
			})
			if err := manager.Save(conversation); err != nil {
				errs <- err
			}
		}()
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	reloaded, err := NewManager(manager.baseDir)
	if err != nil {
		t.Fatal(err)
	}
	stored, _, found, err := reloaded.GetExistingProjectConversation(
		"alpha",
		"inbox",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !found || len(stored.GetHistory(0)) != writers {
		t.Fatalf("stored history count = %d, want %d", len(stored.GetHistory(0)), writers)
	}
	entries, err := os.ReadDir(manager.baseDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp-") {
			t.Fatalf("temporary session file leaked: %s", entry.Name())
		}
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(manager.sessionPath(conversation.Key))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("session mode = %o, want 600", info.Mode().Perm())
		}
	}
}
