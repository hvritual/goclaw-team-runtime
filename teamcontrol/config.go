package teamcontrol

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Enabled bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Root    string `mapstructure:"root" json:"root" yaml:"root"`
}

func DefaultConfig() Config {
	return Config{Enabled: false}
}

// NewService constructs the control plane from application configuration.
// Empty Root resolves to ~/.goclaw/teamcontrol; callers that need an explicit
// test or embedded path may use Open directly.
func NewService(cfg Config) (*Service, error) {
	root := strings.TrimSpace(cfg.Root)
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		if home == "" {
			return nil, errors.New("cannot resolve teamcontrol root")
		}
		root = filepath.Join(home, ".goclaw", "teamcontrol")
	}
	return openService(root, cfg.Enabled)
}
