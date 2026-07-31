package knowledge

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CommentDecisionEvidenceDraft struct {
	WorkspaceID string
	ProjectID   string
	CommentID   string
	Content     string
	UpdatedAt   time.Time
	ActorID     string
}

func NewCommentDecisionEvidence(draft CommentDecisionEvidenceDraft) Evidence {
	contentChecksum := sha256.Sum256([]byte(draft.Content))
	return NewEvidence(EvidenceDraft{
		WorkspaceID:    draft.WorkspaceID,
		ProjectID:      draft.ProjectID,
		SourceType:     "comment",
		SourceID:       draft.CommentID,
		SourceRevision: fmt.Sprintf("%s@sha256:%x", draft.UpdatedAt.UTC().Format(time.RFC3339Nano), contentChecksum),
		EventType:      "comment.decision_proposed",
		Kind:           KindDecision,
		Title:          commentDecisionTitle(draft.Content),
		Content:        draft.Content,
		ActorID:        draft.ActorID,
		OccurredAt:     draft.UpdatedAt,
	})
}

func commentDecisionTitle(content string) string {
	firstLine := strings.TrimSpace(strings.SplitN(content, "\n", 2)[0])
	if firstLine == "" {
		return "Decision from comment"
	}
	const maxRunes = 100
	runes := []rune(firstLine)
	if len(runes) > maxRunes {
		firstLine = string(runes[:maxRunes-1]) + "…"
	}
	return "Decision: " + firstLine
}

func NewEvidence(draft EvidenceDraft) Evidence {
	occurredAt := draft.OccurredAt.UTC()
	revision := draft.SourceRevision
	if revision == "" {
		revision = occurredAt.Format(time.RFC3339Nano)
	}
	uri := "multica://" + draft.SourceType + "s/" + draft.SourceID
	checksum := sha256.Sum256([]byte(draft.Content))
	return Evidence{
		ID: uuid.NewString(), WorkspaceID: draft.WorkspaceID, ProjectID: draft.ProjectID,
		SourceType: draft.SourceType, SourceID: draft.SourceID, SourceRevision: revision,
		EventType: draft.EventType, Kind: draft.Kind, Title: draft.Title, Content: draft.Content,
		ActorID:        draft.ActorID,
		IdempotencyKey: fmt.Sprintf("%s:%s:%s", draft.SourceID, revision, draft.EventType),
		ProvenanceURI:  uri,
		Checksum:       fmt.Sprintf("sha256:%x", checksum),
		OccurredAt:     occurredAt,
		Terminal:       draft.Terminal,
		Validated:      true,
		Confidence:     1,
		SourceRefs: []SourceRef{{
			Type: draft.SourceType, ID: draft.SourceID, Revision: revision, URI: uri,
			Checksum: fmt.Sprintf("sha256:%x", checksum),
		}},
	}
}
