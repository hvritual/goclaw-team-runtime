package contract

import (
	"context"
	"errors"
	"io"
)

var (
	ErrSkillObjectNotFound  = errors.New("Skill object not found")
	ErrSkillObjectForbidden = errors.New("Skill object forbidden")
)

type SkillObject struct {
	ID          string
	WorkspaceID string
	ObjectKey   string
	MediaType   string
	SizeBytes   int64
	Checksum    string
	State       string
}

type StageSkillObjectRequest struct {
	WorkspaceID string
	MediaType   string
	Content     []byte
}

type SkillObjectExecutor interface {
	Execute(context.Context, string, ...any) error
	ExecuteResult(context.Context, string, ...any) (int64, error)
}

type SkillObjectService interface {
	Stage(context.Context, StageSkillObjectRequest) (SkillObject, error)
	Promote(context.Context, SkillObjectExecutor, string, string) error
	Discard(context.Context, string, string) error
	Open(context.Context, string, string) (SkillObject, io.ReadCloser, error)
	Reconcile(context.Context, []string) error
}
