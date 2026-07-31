package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	_, err := h.appendKnowledgeEvidence(ctx, executor, evidence)
	return err
}

func (h *Handler) appendKnowledgeEvidence(
	ctx context.Context,
	executor knowledgeEvidenceExecutor,
	evidence knowledge.Evidence,
) (bool, error) {
	if !h.knowledgeEvidenceEnabled {
		return false, nil
	}
	payload, err := json.Marshal(evidence)
	if err != nil {
		return false, fmt.Errorf("encode knowledge evidence: %w", err)
	}
	tag, err := executor.Exec(ctx, `
		INSERT INTO knowledge_evidence_outbox(
			workspace_id, evidence_id, idempotency_key, payload_json
		) VALUES ($1, $2, $3, $4)
		ON CONFLICT (idempotency_key) DO NOTHING`,
		parseUUID(evidence.WorkspaceID),
		evidence.ID,
		evidence.IdempotencyKey,
		payload,
	)
	if err != nil {
		return false, fmt.Errorf("append knowledge evidence outbox: %w", err)
	}
	return tag.RowsAffected() == 1, nil
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
	return knowledge.NewEvidence(knowledge.EvidenceDraft{
		WorkspaceID: workspaceID, ProjectID: projectID,
		SourceType: sourceType, SourceID: sourceID, SourceRevision: revision,
		EventType: eventType, Kind: kind, Title: title, Content: content,
		ActorID: actorID, OccurredAt: updatedAt, Terminal: terminal,
	})
}

func optionalKnowledgeUUID(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return util.UUIDToString(value)
}
