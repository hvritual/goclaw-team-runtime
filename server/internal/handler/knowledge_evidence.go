package handler

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/knowledge"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type knowledgeEvidenceExecutor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func (h *Handler) enqueueKnowledgeEvidence(
	ctx context.Context,
	executor knowledgeEvidenceExecutor,
	evidence knowledge.Evidence,
) error {
	if !h.knowledgeEvidenceEnabled {
		return nil
	}
	payload, err := json.Marshal(evidence)
	if err != nil {
		return fmt.Errorf("encode knowledge evidence: %w", err)
	}
	if _, err := executor.Exec(ctx, `
		INSERT INTO knowledge_evidence_outbox(
			workspace_id, evidence_id, idempotency_key, payload_json
		) VALUES ($1, $2, $3, $4)
		ON CONFLICT (idempotency_key) DO NOTHING`,
		parseUUID(evidence.WorkspaceID),
		evidence.ID,
		evidence.IdempotencyKey,
		payload,
	); err != nil {
		return fmt.Errorf("append knowledge evidence outbox: %w", err)
	}
	return nil
}

func projectKnowledgeEvidence(project db.Project, actorID, eventType string) knowledge.Evidence {
	content := project.Title
	if project.Description.Valid && strings.TrimSpace(project.Description.String) != "" {
		content += "\n\n" + project.Description.String
	}
	return buildKnowledgeEvidence(
		util.UUIDToString(project.WorkspaceID),
		util.UUIDToString(project.ID),
		"project",
		util.UUIDToString(project.ID),
		project.UpdatedAt.Time,
		eventType,
		knowledge.KindGoal,
		project.Title,
		content,
		actorID,
		project.Status == "completed",
	)
}

func issueKnowledgeEvidence(issue db.Issue, actorID, eventType string) knowledge.Evidence {
	content := issue.Title
	if issue.Description.Valid && strings.TrimSpace(issue.Description.String) != "" {
		content += "\n\n" + issue.Description.String
	}
	return buildKnowledgeEvidence(
		util.UUIDToString(issue.WorkspaceID),
		optionalKnowledgeUUID(issue.ProjectID),
		"issue",
		util.UUIDToString(issue.ID),
		issue.UpdatedAt.Time,
		eventType,
		knowledge.KindRequirement,
		issue.Title,
		content,
		actorID,
		issue.Status == "done",
	)
}

func taskKnowledgeEvidence(task db.Task, actorID, eventType string) knowledge.Evidence {
	content := task.Title
	if strings.TrimSpace(task.Description) != "" {
		content += "\n\n" + task.Description
	}
	return buildKnowledgeEvidence(
		util.UUIDToString(task.WorkspaceID),
		optionalKnowledgeUUID(task.ProjectID),
		"task",
		util.UUIDToString(task.ID),
		task.UpdatedAt.Time,
		eventType,
		knowledge.KindReference,
		task.Title,
		content,
		actorID,
		task.Status == "done" || task.Status == "cancelled",
	)
}

func buildKnowledgeEvidence(
	workspaceID string,
	projectID string,
	sourceType string,
	sourceID string,
	updatedAt time.Time,
	eventType string,
	kind knowledge.Kind,
	title string,
	content string,
	actorID string,
	terminal bool,
) knowledge.Evidence {
	revision := updatedAt.UTC().Format(time.RFC3339Nano)
	checksum := sha256.Sum256([]byte(content))
	return knowledge.Evidence{
		ID: uuid.NewString(), WorkspaceID: workspaceID, ProjectID: projectID,
		SourceType: sourceType, SourceID: sourceID, SourceRevision: revision,
		EventType: eventType, Kind: kind, Title: title, Content: content,
		ActorID:        actorID,
		IdempotencyKey: fmt.Sprintf("%s:%s:%s", sourceID, revision, eventType),
		ProvenanceURI:  "multica://" + sourceType + "s/" + sourceID,
		Checksum:       fmt.Sprintf("sha256:%x", checksum),
		OccurredAt:     updatedAt.UTC(), Terminal: terminal, Validated: true, Confidence: 1,
		SourceRefs: []knowledge.SourceRef{{
			Type: sourceType, ID: sourceID, Revision: revision,
			URI: "multica://" + sourceType + "s/" + sourceID,
		}},
	}
}

func optionalKnowledgeUUID(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return util.UUIDToString(value)
}
