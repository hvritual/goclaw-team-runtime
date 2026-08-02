// Package postgres contains Postgres adapters for the space module.
// Implement domain or application ports here and keep driver types inside this package.
package postgres

import (
	"github.com/multica-ai/multica/server/internal/modules/space/contract"
	"github.com/multica-ai/multica/server/internal/modules/space/internal/application"
)

// Config is the provider-owned composition input. Add the native connection
// and provider settings here; never pass them into domain or application APIs.
type Config struct {
	// Application is an optional assembled service used by composition and tests.
	// Replace or extend this seam with provider-native connections and adapters.
	Application contract.Service
}

func New(config Config) contract.Service {
	if config.Application != nil {
		return config.Application
	}
	return application.New()
}
