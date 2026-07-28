package memory

import (
	"context"
	"path/filepath"
	"testing"
)

func TestLocalHashBuiltinSearchRequiresNoAPIKey(t *testing.T) {
	provider := NewLocalHashProvider()
	storeConfig := DefaultStoreConfig(filepath.Join(t.TempDir(), "memory.db"), provider)
	storeConfig.EnableVectorSearch = false
	storeConfig.EnableFTS = false
	store, err := NewSQLiteStore(storeConfig)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewMemoryManager(DefaultManagerConfig(store, provider))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()
	if _, err := manager.AddMemory(
		ctx,
		"SQLite stores the local project catalog with WAL enabled.",
		MemorySourceLongTerm,
		MemoryTypeFact,
		MemoryMetadata{FilePath: "MEMORY.md"},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.AddMemory(
		ctx,
		"Coffee beans should be stored in an airtight container.",
		MemorySourceLongTerm,
		MemoryTypeFact,
		MemoryMetadata{FilePath: "OTHER.md"},
	); err != nil {
		t.Fatal(err)
	}
	options := DefaultSearchOptions()
	options.MinScore = 0
	options.Limit = 2
	results, err := manager.Search(ctx, "SQLite local catalog", options)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 || results[0].Metadata.FilePath != "MEMORY.md" {
		t.Fatalf("unexpected search results: %+v", results)
	}
}
