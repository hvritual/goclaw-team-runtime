package outbox_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/knowledge"
	"github.com/multica-ai/multica/server/internal/knowledge/adapter/memory"
	"github.com/multica-ai/multica/server/internal/knowledge/outbox"
)

type queue struct {
	messages  []outbox.Message
	delivered []string
	failed    []string
}

type missingProjectValidator struct{}

func (missingProjectValidator) ValidateProject(context.Context, string, string) error {
	return errors.New("project was deleted")
}

func (q *queue) NextBatch(context.Context, int) ([]outbox.Message, error) {
	return append([]outbox.Message(nil), q.messages...), nil
}

func (q *queue) MarkDelivered(_ context.Context, id string, _ time.Time) error {
	q.delivered = append(q.delivered, id)
	q.messages = nil
	return nil
}

func (q *queue) MarkFailed(_ context.Context, id string, _ time.Time, _ error) error {
	q.failed = append(q.failed, id)
	return nil
}

func TestDispatcherMarksIdempotentEvidenceDelivered(t *testing.T) {
	service := knowledge.NewService(memory.New(), nil)
	evidence := knowledge.Evidence{
		ID:             "evidence-1",
		WorkspaceID:    "workspace-1",
		SourceType:     "task",
		SourceID:       "task-1",
		SourceRevision: "2",
		EventType:      "task.completed",
		Kind:           knowledge.KindReference,
		Title:          "Task completed",
		Content:        "The restore task completed.",
		ActorID:        "user-1",
		IdempotencyKey: "task-1:2:task.completed",
		OccurredAt:     time.Now().UTC(),
		Terminal:       true,
		Validated:      true,
		Confidence:     1,
		SourceRefs: []knowledge.SourceRef{{
			Type: "task",
			ID:   "task-1",
			URI:  "multica://tasks/task-1",
		}},
	}
	message, err := outbox.NewMessage("message-1", evidence)
	if err != nil {
		t.Fatal(err)
	}
	source := &queue{messages: []outbox.Message{message}}
	dispatcher := outbox.NewDispatcher(source, service)

	report, err := dispatcher.Drain(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if report.Delivered != 1 || len(source.delivered) != 1 {
		t.Fatalf("report = %#v, delivered = %#v", report, source.delivered)
	}
}

func TestDispatcherRetainsFailedEvidenceForRetry(t *testing.T) {
	message := outbox.Message{ID: "message-1", Payload: []byte(`{"invalid":true}`)}
	source := &queue{messages: []outbox.Message{message}}
	dispatcher := outbox.NewDispatcher(source, knowledge.NewService(memory.New(), nil))

	report, err := dispatcher.Drain(context.Background(), 10)
	if !errors.Is(err, outbox.ErrDeliveryFailed) {
		t.Fatalf("error = %v, want %v", err, outbox.ErrDeliveryFailed)
	}
	if report.Failed != 1 || len(source.failed) != 1 || len(source.messages) != 1 {
		t.Fatalf("report = %#v, failed = %#v, remaining = %#v", report, source.failed, source.messages)
	}
}

func TestDispatcherReplaysEvidenceAfterSourceProjectDeletion(t *testing.T) {
	service := knowledge.NewService(memory.New(), nil, missingProjectValidator{})
	evidence := knowledge.Evidence{
		ID:             "evidence-1",
		WorkspaceID:    "workspace-1",
		ProjectID:      "deleted-project",
		SourceType:     "task",
		SourceID:       "task-1",
		SourceRevision: "2",
		EventType:      "task.completed",
		Kind:           knowledge.KindReference,
		Title:          "Task completed",
		Content:        "The project-scoped task completed before its project was deleted.",
		ActorID:        "user-1",
		IdempotencyKey: "task-1:2:task.completed",
		OccurredAt:     time.Now().UTC(),
		Terminal:       true,
		Validated:      true,
		Confidence:     1,
		SourceRefs: []knowledge.SourceRef{{
			Type: "task",
			ID:   "task-1",
			URI:  "multica://tasks/task-1",
		}},
	}
	message, err := outbox.NewMessage("message-1", evidence)
	if err != nil {
		t.Fatal(err)
	}
	source := &queue{messages: []outbox.Message{message}}
	dispatcher := outbox.NewDispatcher(source, service)

	report, err := dispatcher.Drain(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if report.Delivered != 1 || len(source.delivered) != 1 {
		t.Fatalf("report = %#v, delivered = %#v", report, source.delivered)
	}
}
