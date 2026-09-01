package domain

import "context"

type SourceBindingLookupRepository interface {
	FindSourceBindingBySource(context.Context, string, string, string) (SourceBinding, error)
}
