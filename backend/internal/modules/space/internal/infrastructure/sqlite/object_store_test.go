package sqlite

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestObjectStoreRejectsEscapedKeys(t *testing.T) {
	store, err := NewObjectStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"", ".", "..", "../outside", `..\outside`, filepath.Join(t.TempDir(), "absolute")} {
		if err := store.Put(t.Context(), key, []byte("secret")); err == nil {
			t.Fatalf("Put(%q) unexpectedly succeeded", key)
		}
	}
}

func TestObjectStoreReconcileRestoresReferencedTombstoneAndRemovesOrphans(t *testing.T) {
	root := t.TempDir()
	store, err := NewObjectStore(root)
	if err != nil {
		t.Fatal(err)
	}
	const key = "workspace/asset/version.blob"
	if err := store.Put(t.Context(), key, []byte("retained")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Quarantine(t.Context(), key); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(t.Context(), "workspace/orphan/version.blob", []byte("orphan")); err != nil {
		t.Fatal(err)
	}

	if err := store.Reconcile(t.Context(), []string{key}); err != nil {
		t.Fatalf("Reconcile referenced tombstone: %v", err)
	}
	reader, err := store.Open(t.Context(), key)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	content, err := io.ReadAll(reader)
	if err != nil || string(content) != "retained" {
		t.Fatalf("restored content = %q, %v", content, err)
	}
	orphan, err := store.pathFor("workspace/orphan/version.blob")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Fatalf("orphan still present: %v", err)
	}
}

func TestObjectStoreReconcileRemovesCommittedDeleteTombstone(t *testing.T) {
	root := t.TempDir()
	store, err := NewObjectStore(root)
	if err != nil {
		t.Fatal(err)
	}
	const key = "workspace/asset/version.blob"
	if err := store.Put(context.Background(), key, []byte("deleted")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Quarantine(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	if err := store.Reconcile(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	count := 0
	if err := filepath.WalkDir(root, func(_ string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			count++
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("committed-delete files = %d", count)
	}
}
