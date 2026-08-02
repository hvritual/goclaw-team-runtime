package asset

// Asset is the aggregate root. Keep invariants and state transitions here.
type Asset struct {
	id string
}

func New(id string) *Asset  { return &Asset{id: id} }
func (a *Asset) ID() string { return a.id }
