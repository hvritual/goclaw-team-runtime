package cli

import (
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
