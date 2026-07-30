package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/multica-ai/multica/server/internal/cli"
)

const (
	defaultCloudServerURL = "https://api.multica.ai"
	defaultCloudAppURL    = "https://multica.ai"
)

var uuidRegexp = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func resolveProfile(cmd *cobra.Command) string {
	value, _ := cmd.Flags().GetString("profile")
	return value
}

func tryResolveServerURL(cmd *cobra.Command) string {
	if value := cli.FlagOrEnv(cmd, "server-url", "MULTICA_SERVER_URL", ""); value != "" {
		return normalizeAPIBaseURL(value)
	}
	config, err := cli.LoadCLIConfigForProfile(resolveProfile(cmd))
	if err == nil && strings.TrimSpace(config.ServerURL) != "" {
		return normalizeAPIBaseURL(config.ServerURL)
	}
	return ""
}

func resolveServerURL(cmd *cobra.Command) string {
	if value := tryResolveServerURL(cmd); value != "" {
		return value
	}
	fmt.Fprintln(os.Stderr, "No server configured. Run 'multica setup' first.")
	return ""
}

func resolveLoginTokenServerURL(cmd *cobra.Command) string {
	if value := tryResolveServerURL(cmd); value != "" {
		return value
	}
	return defaultCloudServerURL
}

func normalizeAPIBaseURL(value string) string {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	value = strings.Replace(value, "ws://", "http://", 1)
	value = strings.Replace(value, "wss://", "https://", 1)
	if !strings.Contains(value, "://") {
		return "http://" + value
	}
	if strings.HasSuffix(value, "/ws") {
		value = strings.TrimSuffix(value, "/ws")
	}
	return value
}

func resolveWorkspaceID(cmd *cobra.Command) string {
	if value := cli.FlagOrEnv(cmd, "workspace-id", "MULTICA_WORKSPACE_ID", ""); value != "" {
		return value
	}
	config, _ := cli.LoadCLIConfigForProfile(resolveProfile(cmd))
	return config.WorkspaceID
}

func requireWorkspaceID(cmd *cobra.Command) (string, error) {
	if value := resolveWorkspaceID(cmd); value != "" {
		return value, nil
	}
	return "", fmt.Errorf("workspace_id is required: use --workspace-id, MULTICA_WORKSPACE_ID, or configure a default workspace")
}

func newAPIClient(cmd *cobra.Command) (*cli.APIClient, error) {
	serverURL := resolveServerURL(cmd)
	if serverURL == "" {
		return nil, fmt.Errorf("server URL is required")
	}
	return cli.NewAPIClient(serverURL, resolveWorkspaceID(cmd), resolveToken(cmd)), nil
}

func strVal(values map[string]any, key string) string {
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}
