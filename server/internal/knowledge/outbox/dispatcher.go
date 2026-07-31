package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/multica-ai/multica/server/internal/knowledge"
)

var ErrDeliveryFailed = errors.New("knowledge evidence delivery failed")

type Message struct {
	ID          string
	WorkspaceID string
	Payload     []byte
	Attempts    int
	CreatedAt   time.Time
}

type Store interface {
	NextBatch(context.Context, int) ([]Message, error)
	MarkDelivered(context.Context, string, time.Time) error
	MarkFailed(context.Context, string, time.Time, error) error
}

type Report struct {
	Scanned   int
	Delivered int
	Failed    int
}

type Dispatcher struct {
	store   Store
	service *knowledge.Service
}

func NewDispatcher(store Store, service *knowledge.Service) *Dispatcher {
	return &Dispatcher{store: store, service: service}
}

func NewMessage(id string, evidence knowledge.Evidence) (Message, error) {
	payload, err := json.Marshal(evidence)
	if err != nil {
		return Message{}, fmt.Errorf("encode knowledge evidence: %w", err)
	}
	return Message{
		ID:          id,
		WorkspaceID: evidence.WorkspaceID,
		Payload:     payload,
		CreatedAt:   time.Now().UTC(),
	}, nil
}

func (d *Dispatcher) Drain(ctx context.Context, limit int) (Report, error) {
	if limit <= 0 {
		limit = 50
	}
	messages, err := d.store.NextBatch(ctx, limit)
	if err != nil {
		return Report{}, fmt.Errorf("read knowledge evidence outbox: %w", err)
	}
	report := Report{Scanned: len(messages)}
	var deliveryErrors []error
	for _, message := range messages {
		var evidence knowledge.Evidence
		deliveryErr := json.Unmarshal(message.Payload, &evidence)
		if deliveryErr == nil {
			_, deliveryErr = d.service.IngestOutboxEvidence(ctx, evidence)
		}
		if deliveryErr != nil {
			report.Failed++
			deliveryErrors = append(deliveryErrors, deliveryErr)
			if markErr := d.store.MarkFailed(ctx, message.ID, time.Now().UTC(), deliveryErr); markErr != nil {
				deliveryErrors = append(deliveryErrors, markErr)
			}
			continue
		}
		if err := d.store.MarkDelivered(ctx, message.ID, time.Now().UTC()); err != nil {
			report.Failed++
			deliveryErrors = append(deliveryErrors, err)
			continue
		}
		report.Delivered++
	}
	if len(deliveryErrors) > 0 {
		return report, errors.Join(append([]error{ErrDeliveryFailed}, deliveryErrors...)...)
	}
	return report, nil
}
