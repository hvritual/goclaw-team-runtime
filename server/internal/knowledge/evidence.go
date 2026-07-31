package knowledge

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/google/uuid"
)

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
		}},
	}
}
