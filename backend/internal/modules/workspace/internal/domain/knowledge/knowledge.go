package knowledge

import (
	"errors"
	"strings"
	"time"
)

const StatusCandidate = "candidate"

var ErrInvalid = errors.New("invalid knowledge")

type Knowledge struct {
	ID, WorkspaceID, Title, Summary, Status string
	AssetIDs                                []string
	CreatedAt, UpdatedAt                    time.Time
}

func New(id, workspaceID, title, summary string, assetIDs []string, now time.Time) (Knowledge, error) {
	id, workspaceID, title = strings.TrimSpace(id), strings.TrimSpace(workspaceID), strings.TrimSpace(title)
	if id == "" || workspaceID == "" || title == "" {
		return Knowledge{}, ErrInvalid
	}
	assets := make([]string, len(assetIDs))
	for index, asset := range assetIDs {
		assets[index] = strings.TrimSpace(asset)
		if assets[index] == "" {
			return Knowledge{}, ErrInvalid
		}
	}
	now = now.UTC()
	return Knowledge{ID: id, WorkspaceID: workspaceID, Title: title, Summary: summary, Status: StatusCandidate, AssetIDs: assets, CreatedAt: now, UpdatedAt: now}, nil
}

func Rehydrate(value Knowledge) (Knowledge, error) {
	created, err := New(value.ID, value.WorkspaceID, value.Title, value.Summary, value.AssetIDs, value.CreatedAt)
	if err != nil {
		return Knowledge{}, err
	}
	if value.Status != StatusCandidate && value.Status != "in_review" && value.Status != "published" && value.Status != "quarantined" {
		return Knowledge{}, ErrInvalid
	}
	created.Status, created.UpdatedAt = value.Status, value.UpdatedAt.UTC()
	return created, nil
}
