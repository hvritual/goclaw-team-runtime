package knowledge

import "time"

type SourceRef struct {
	Type, ID, Revision, Citation string
	AssetID, AssetVersionID      *string
}

type Revision struct {
	Number, SupersedesRevision int
	Title, Content, CreatedBy  string
	CreatedAt                  time.Time
	SourceRefs                 []SourceRef
}

type GovernedEntry struct {
	ID, WorkspaceID, Kind, Status string
	ProjectID, CandidateID        *string
	CurrentRevision               int
	Revisions                     []Revision
	CreatedAt, UpdatedAt          time.Time
}
