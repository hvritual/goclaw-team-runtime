package workspace

// Workspace is the aggregate root. Keep invariants and state transitions here.
type Workspace struct {
	id   string
	name string
}

func New(id, name string) *Workspace { return &Workspace{id: id, name: name} }
func (a Workspace) ID() string       { return a.id }
func (a Workspace) Name() string     { return a.name }
