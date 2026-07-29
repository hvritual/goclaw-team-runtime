package cli

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestApplicationCommandSurfaces(t *testing.T) {
	t.Parallel()

	allCommands := []string{
		"config",
		"dev",
		"gateway",
		"harness",
		"health",
		"ouroboros",
		"runner",
		"status",
		"team",
		"version",
	}
	tests := []struct {
		name      string
		mode      ApplicationMode
		wantUse   string
		want      []string
		forbidden []string
	}{
		{
			name:    "unified compatibility",
			mode:    ApplicationUnified,
			wantUse: "goclaw",
			want:    allCommands,
		},
		{
			name:    "team control",
			mode:    ApplicationTeamControl,
			wantUse: "goclaw-team-control",
			want: []string{
				"config",
				"dev",
				"gateway",
				"harness",
				"health",
				"ouroboros",
				"status",
				"team",
				"version",
			},
			forbidden: []string{"runner"},
		},
		{
			name:    "runner",
			mode:    ApplicationRunner,
			wantUse: "goclaw-runner",
			want: []string{
				"config",
				"health",
				"runner",
				"status",
				"version",
			},
			forbidden: []string{
				"dev",
				"gateway",
				"harness",
				"ouroboros",
				"team",
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := &cobra.Command{Use: "test"}
			for _, name := range allCommands {
				root.AddCommand(&cobra.Command{Use: name})
			}
			spec, err := applicationSpecification(test.mode)
			require.NoError(t, err)
			require.Equal(t, test.wantUse, spec.use)
			applyApplicationSpecification(root, spec)
			require.Equal(t, test.want, applicationCommandNames(root))
			for _, forbidden := range test.forbidden {
				require.NotContains(t, applicationCommandNames(root), forbidden)
			}
		})
	}
}

func TestApplicationSpecificationRejectsUnknownMode(t *testing.T) {
	t.Parallel()
	_, err := applicationSpecification(ApplicationMode("unknown"))
	require.ErrorContains(t, err, "unsupported application mode")
}

func TestDedicatedEntrypointsExposeActualCommandSurfaces(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and probes the actual dedicated entrypoints")
	}
	repoDir := filepath.Clean("..")
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	tests := []struct {
		name      string
		pkg       string
		wantUse   string
		required  []string
		forbidden []string
	}{
		{
			name:      "team-control",
			pkg:       "./cmd/team-control",
			wantUse:   "goclaw-team-control",
			required:  []string{"dev", "gateway", "harness", "team"},
			forbidden: []string{"runner"},
		},
		{
			name:      "runner",
			pkg:       "./cmd/runner",
			wantUse:   "goclaw-runner",
			required:  []string{"config", "health", "runner", "status", "version"},
			forbidden: []string{"dev", "gateway", "harness", "ouroboros", "team"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			binary := filepath.Join(t.TempDir(), test.wantUse+suffix)
			build := exec.Command(
				"go", "build", "-buildvcs=false", "-trimpath",
				"-o", binary, test.pkg,
			)
			build.Dir = repoDir
			output, err := build.CombinedOutput()
			require.NoError(t, err, string(output))

			help, err := exec.Command(binary, "--help").CombinedOutput()
			require.NoError(t, err, string(help))
			helpText := string(help)
			require.Contains(t, helpText, "Usage:\n  "+test.wantUse+" [command]")
			for _, command := range test.required {
				require.Contains(t, helpText, "\n  "+command+" ")
			}
			for _, command := range test.forbidden {
				require.NotContains(t, helpText, "\n  "+command+" ")
			}

			version, err := exec.Command(binary, "version").CombinedOutput()
			require.NoError(t, err, string(version))
			require.True(t, strings.HasPrefix(string(version), test.wantUse+" "))
		})
	}
}
