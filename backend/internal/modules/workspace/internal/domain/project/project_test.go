package project

import (
	"errors"
	"testing"
	"time"
)

func TestProjectStatusCompatibility(t *testing.T) {
	now := time.Date(2026, 8, 3, 1, 2, 3, 0, time.FixedZone("test", 8*60*60))
	for _, status := range []string{"", StatusPlanned, StatusInProgress, StatusPaused, StatusCompleted, StatusCancelled} {
		t.Run(status, func(t *testing.T) {
			value, err := New("project-1", "workspace-1", "  Delivery  ", "desc", status, []string{" asset-1 "}, now)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			wantStatus := status
			if wantStatus == "" {
				wantStatus = StatusPlanned
			}
			if value.Status() != wantStatus || value.Name() != "Delivery" {
				t.Fatalf("Project status/name = %q/%q", value.Status(), value.Name())
			}
			if got := value.AssetIDs(); len(got) != 1 || got[0] != "asset-1" {
				t.Fatalf("AssetIDs() = %v", got)
			}
			if value.CreatedAt().Location() != time.UTC || value.UpdatedAt().Location() != time.UTC {
				t.Fatal("timestamps must be normalized to UTC")
			}
		})
	}
}

func TestProjectRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		workspace string
		project   string
		status    string
		assets    []string
		wantErr   error
	}{
		{name: "missing id", workspace: "workspace-1", project: "Project", wantErr: ErrIDRequired},
		{name: "missing workspace", id: "project-1", project: "Project", wantErr: ErrWorkspaceRequired},
		{name: "missing name", id: "project-1", workspace: "workspace-1", project: " ", wantErr: ErrNameRequired},
		{name: "legacy invalid active status", id: "project-1", workspace: "workspace-1", project: "Project", status: "active", wantErr: ErrInvalidStatus},
		{name: "empty asset reference", id: "project-1", workspace: "workspace-1", project: "Project", assets: []string{" "}, wantErr: ErrAssetIDRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.id, tt.workspace, tt.project, "", tt.status, tt.assets, time.Now())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("New() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestProjectAssetIDsReturnsCopy(t *testing.T) {
	value, err := New("project-1", "workspace-1", "Project", "", "", []string{"asset-1"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	assets := value.AssetIDs()
	assets[0] = "changed"
	if value.AssetIDs()[0] != "asset-1" {
		t.Fatal("AssetIDs exposed mutable aggregate state")
	}
}

func TestProjectUpdatePreservesOmittedFieldsAndAssetReferences(t *testing.T) {
	createdAt := time.Date(2026, 8, 3, 1, 2, 3, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	value, err := New("project-1", "workspace-1", "Original", "description", StatusPlanned, []string{"asset-1"}, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	name := "  Delivery  "
	description := ""
	status := StatusInProgress
	updated, err := value.Update(&name, &description, &status, updatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name() != "Delivery" || updated.Description() != "" || updated.Status() != StatusInProgress {
		t.Fatalf("updated Project = %q/%q/%q", updated.Name(), updated.Description(), updated.Status())
	}
	if updated.AssetIDs()[0] != "asset-1" || !updated.CreatedAt().Equal(createdAt) || !updated.UpdatedAt().Equal(updatedAt) {
		t.Fatalf("immutable fields changed: assets=%v created=%v updated=%v", updated.AssetIDs(), updated.CreatedAt(), updated.UpdatedAt())
	}
	status = "active"
	if _, err := updated.Update(nil, nil, &status, updatedAt); !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("Update() invalid status error = %v", err)
	}
}
