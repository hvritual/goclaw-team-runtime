package space

import (
	"context"
	"testing"
)

type postgresSelectionService struct{}

func (postgresSelectionService) Ping(context.Context) (string, error) { return "postgres", nil }

func TestPostgresPersistenceSelection(t *testing.T) {
	module := NewWithPostgresPersistence(PostgresPersistenceConfig{Application: postgresSelectionService{}})
	message, err := module.Local().Ping(context.Background())
	if err != nil || message != "postgres" {
		t.Fatalf("selected persistence result = %q, %v", message, err)
	}
}
