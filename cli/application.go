package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

// ApplicationMode selects the command surface exposed by a compiled entrypoint.
// Authorization continues to be enforced by the server; this is a deployment
// and least-confusion boundary, not a replacement for RBAC.
type ApplicationMode string

const (
	ApplicationUnified     ApplicationMode = "unified"
	ApplicationTeamControl ApplicationMode = "team-control"
	ApplicationRunner      ApplicationMode = "runner"
)

type applicationSpec struct {
	use     string
	short   string
	allowed map[string]struct{}
}

var activeApplication = ApplicationUnified

// ConfigureApplication narrows the global command tree for a dedicated
// executable. It must be called once by main before Execute.
func ConfigureApplication(mode ApplicationMode) error {
	spec, err := applicationSpecification(mode)
	if err != nil {
		return err
	}
	activeApplication = mode
	rootCmd.Use = spec.use
	rootCmd.Short = spec.short
	rootCmd.Long = spec.short + "."
	applyApplicationSpecification(rootCmd, spec)
	return nil
}

func applicationSpecification(mode ApplicationMode) (applicationSpec, error) {
	switch mode {
	case ApplicationUnified:
		return applicationSpec{
			use:   "goclaw",
			short: "Go-based AI Agent framework",
		}, nil
	case ApplicationTeamControl:
		return applicationSpec{
			use:   "goclaw-team-control",
			short: "Central GoClaw team governance and runtime control plane",
			allowed: commandSet(
				"agent",
				"agents",
				"approvals",
				"browser",
				"channels",
				"config",
				"cron",
				"dev",
				"gateway",
				"harness",
				"health",
				"install",
				"logs",
				"memory",
				"onboard",
				"ouroboros",
				"pairing",
				"pilot",
				"sessions",
				"skills",
				"start",
				"status",
				"system",
				"team",
				"tui",
				"version",
			),
		}, nil
	case ApplicationRunner:
		return applicationSpec{
			use:   "goclaw-runner",
			short: "Project-scoped GoClaw Codex workstation runner",
			allowed: commandSet(
				"config",
				"health",
				"runner",
				"status",
				"version",
			),
		}, nil
	default:
		return applicationSpec{}, fmt.Errorf("unsupported application mode %q", mode)
	}
}

func commandSet(names ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(names))
	for _, name := range names {
		result[name] = struct{}{}
	}
	return result
}

func applyApplicationSpecification(root *cobra.Command, spec applicationSpec) {
	if spec.allowed == nil {
		return
	}
	for _, command := range root.Commands() {
		if _, ok := spec.allowed[command.Name()]; !ok {
			root.RemoveCommand(command)
		}
	}
}

func applicationCommandNames(root *cobra.Command) []string {
	names := make([]string, 0, len(root.Commands()))
	for _, command := range root.Commands() {
		names = append(names, command.Name())
	}
	sort.Strings(names)
	return names
}

func applicationBinaryName() string {
	spec, err := applicationSpecification(activeApplication)
	if err != nil {
		return "goclaw"
	}
	return spec.use
}
