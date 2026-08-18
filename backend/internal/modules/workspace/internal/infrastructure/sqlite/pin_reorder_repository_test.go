package sqlite_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	persistence "github.com/hvritual/workspace/internal/modules/workspace/internal/infrastructure/sqlite"
)

func TestPinReorderRepositoryIsAtomicRevisionedAndWorkspaceScoped(t *testing.T) {
	db := openProjectSearchDB(t)
	for _, statement := range []string{
		`INSERT INTO workspace_pins(id,workspace_id,user_id,item_type,item_id,position,created_at) VALUES('pin-1','workspace-1','user-1','issue','issue-1',1,'2026-08-18T00:00:00Z')`,
		`INSERT INTO workspace_pins(id,workspace_id,user_id,item_type,item_id,position,created_at) VALUES('pin-2','workspace-1','user-1','project','project-1',2,'2026-08-18T00:00:01Z')`,
		`INSERT INTO workspace_pins(id,workspace_id,user_id,item_type,item_id,position,created_at) VALUES('foreign','workspace-2','user-1','project','project-2',1,'2026-08-18T00:00:02Z')`,
		`INSERT INTO workspace_pins(id,workspace_id,user_id,item_type,item_id,position,created_at) VALUES('other-user','workspace-1','user-2','issue','issue-2',1,'2026-08-18T00:00:03Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	repository, err := persistence.NewProjectSurfaceRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	pins, err := repository.ListPins(context.Background(), "workspace-1", "user-1")
	if err != nil || len(pins) != 2 || pins[0].OrderRevision != 2 || pins[1].OrderRevision != 2 {
		t.Fatalf("initial pins = %+v error=%v", pins, err)
	}
	if err := repository.ReorderPins(context.Background(), "workspace-1", "user-1", []string{"pin-2", "pin-1"}, 2); err != nil {
		t.Fatal(err)
	}
	assertPinOrderAndRevision(t, repository, []string{"pin-2", "pin-1"}, 3)

	var conflict contract.RevisionConflictError
	if err := repository.ReorderPins(context.Background(), "workspace-1", "user-1", []string{"pin-1", "pin-2"}, 2); !errors.As(err, &conflict) || conflict.CurrentRevision != 3 {
		t.Fatalf("stale reorder error = %v", err)
	}
	assertPinOrderAndRevision(t, repository, []string{"pin-2", "pin-1"}, 3)

	for _, ids := range [][]string{{"pin-2"}, {"pin-2", "foreign"}, {"pin-2", "other-user"}, {"pin-2", "pin-2"}, {"pin-2", "pin-1", "foreign"}} {
		if err := repository.ReorderPins(context.Background(), "workspace-1", "user-1", ids, 3); !errors.Is(err, application.ErrInvalidProjectSurfaceRequest) {
			t.Fatalf("invalid set %v error = %v", ids, err)
		}
		assertPinOrderAndRevision(t, repository, []string{"pin-2", "pin-1"}, 3)
	}
}

func TestPinReorderRepositoryRollsBackEarlierPositionWritesOnFailure(t *testing.T) {
	db := openProjectSearchDB(t)
	if _, err := db.Exec(`INSERT INTO workspace_pins(id,workspace_id,user_id,item_type,item_id,position,created_at) VALUES('pin-1','workspace-1','user-1','issue','issue-1',1,'2026-08-18T00:00:00Z'),('pin-2','workspace-1','user-1','project','project-1',2,'2026-08-18T00:00:01Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TRIGGER reject_second_pin_position BEFORE UPDATE OF position ON workspace_pins WHEN OLD.id='pin-1' BEGIN SELECT RAISE(ABORT,'reject pin position'); END`); err != nil {
		t.Fatal(err)
	}
	repository, err := persistence.NewProjectSurfaceRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.ReorderPins(context.Background(), "workspace-1", "user-1", []string{"pin-2", "pin-1"}, 2); err == nil {
		t.Fatal("ReorderPins() error = nil")
	}
	assertPinOrderAndRevision(t, repository, []string{"pin-1", "pin-2"}, 2)
}

func TestPinOrderRevisionTracksDeleteAndCancellationDoesNotMutate(t *testing.T) {
	db := openProjectSearchDB(t)
	if _, err := db.Exec(`INSERT INTO workspace_pins(id,workspace_id,user_id,item_type,item_id,position,created_at) VALUES('pin-1','workspace-1','user-1','issue','issue-1',1,'2026-08-18T00:00:00Z'),('pin-2','workspace-1','user-1','project','project-1',2,'2026-08-18T00:00:01Z')`); err != nil {
		t.Fatal(err)
	}
	repository, err := persistence.NewProjectSurfaceRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.DeletePin(context.Background(), "workspace-1", "user-1", "project", "project-1"); err != nil {
		t.Fatal(err)
	}
	assertPinOrderAndRevision(t, repository, []string{"pin-1"}, 3)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := repository.ReorderPins(cancelled, "workspace-1", "user-1", []string{"pin-1"}, 3); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled reorder error = %v", err)
	}
	assertPinOrderAndRevision(t, repository, []string{"pin-1"}, 3)
}

func assertPinOrderAndRevision(t *testing.T, repository *persistence.ProjectSurfaceRepository, ids []string, revision int64) {
	t.Helper()
	pins, err := repository.ListPins(context.Background(), "workspace-1", "user-1")
	if err != nil || len(pins) != len(ids) {
		t.Fatalf("pins = %+v error=%v", pins, err)
	}
	for index, id := range ids {
		if pins[index].ID != id || pins[index].Position != float64(index+1) || pins[index].OrderRevision != revision {
			t.Fatalf("pin %d = %+v, want id=%s position=%d revision=%d", index, pins[index], id, index+1, revision)
		}
	}
}
