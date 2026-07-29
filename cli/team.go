package cli

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/smallnest/goclaw/config"
	"github.com/smallnest/goclaw/teamcontrol"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newTeamCommand())
}

func newTeamCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "team",
		Short: "Bootstrap and administer the authenticated team control plane",
	}
	command.AddCommand(
		newTeamBootstrapCommand(),
		newTeamCreateCommand(),
		newTeamUserCreateCommand(),
		newTeamTokenIssueCommand(),
		newTeamMemberAddCommand(),
		newProjectCreateCommand(),
		newProjectMemberAddCommand(),
		newRepositoryCreateCommand(),
		newTokenBudgetPutCommand(),
		newControlSummaryCommand(),
		newContextCompileCommand(),
		newTeamRPCCommand(),
	)
	return command
}

func newTeamRPCCommand() *cobra.Command {
	var paramsFile string
	command := &cobra.Command{
		Use:   "rpc METHOD",
		Short: "Call an authenticated team RPC using parameters from a JSON file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := make(map[string]any)
			if strings.TrimSpace(paramsFile) != "" {
				info, err := os.Stat(paramsFile)
				if err != nil {
					return err
				}
				if !info.Mode().IsRegular() {
					return errors.New("--params must name a regular JSON file")
				}
				if info.Size() > 1024*1024 {
					return errors.New("--params exceeds 1 MiB")
				}
				data, err := os.ReadFile(paramsFile)
				if err != nil {
					return err
				}
				if err := json.Unmarshal(data, &params); err != nil {
					return fmt.Errorf("decode --params: %w", err)
				}
			}
			result, err := callTeamRPCResult(args[0], params)
			if err != nil {
				return err
			}
			return printTeamValue(result)
		},
	}
	command.Flags().StringVar(&paramsFile, "params", "", "JSON object file; omitted sends an empty object")
	return command
}

func newTeamBootstrapCommand() *cobra.Command {
	var root, userID, displayName, email, label, tokenFile string
	command := &cobra.Command{
		Use:   "bootstrap",
		Short: "Create the first local administrator and personal access token",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load("")
			if err != nil {
				return err
			}
			if strings.TrimSpace(root) != "" {
				cfg.TeamControl.Root = root
			}
			service, err := teamcontrol.NewService(cfg.TeamControl)
			if err != nil {
				return err
			}
			initialized, err := service.HasUsers()
			if err != nil {
				return err
			}
			if initialized {
				return errors.New("team control is already initialized; use authenticated team commands")
			}
			token, err := generateAccessToken()
			if err != nil {
				return err
			}
			cleanup, err := stageSecretFile(tokenFile, token)
			if err != nil {
				return err
			}
			keep := false
			defer func() {
				if !keep {
					cleanup()
				}
			}()
			user, credential, err := service.BootstrapFirstUser(
				teamcontrol.CreateUserInput{
					ID:          userID,
					DisplayName: displayName,
					Email:       email,
				},
				label,
				token,
			)
			if err != nil {
				return err
			}
			keep = true
			return printTeamValue(map[string]any{
				"user":            user,
				"credential":      credential,
				"token_file":      tokenFile,
				"team_root":       service.Config().Root,
				"next":            "export GOCLAW_USER_TOKEN=\"$(cat " + tokenFile + ")\"",
				"plaintext_token": "written once to token_file; never persisted by team control",
			})
		},
	}
	command.Flags().StringVar(&root, "root", "", "Team control root override")
	command.Flags().StringVar(&userID, "user", "", "First administrator user id")
	command.Flags().StringVar(&displayName, "name", "", "Administrator display name")
	command.Flags().StringVar(&email, "email", "", "Administrator email")
	command.Flags().StringVar(&label, "label", "bootstrap", "Credential label")
	command.Flags().StringVar(&tokenFile, "token-file", "", "New 0600 file for the generated personal token")
	_ = command.MarkFlagRequired("user")
	_ = command.MarkFlagRequired("name")
	_ = command.MarkFlagRequired("token-file")
	return command
}

func newTeamCreateCommand() *cobra.Command {
	var id, name, description string
	command := &cobra.Command{
		Use:   "create",
		Short: "Create a team and make the caller its owner",
		RunE: func(cmd *cobra.Command, args []string) error {
			return callTeamRPC("team.create", map[string]any{
				"id":          id,
				"name":        name,
				"description": description,
			})
		},
	}
	command.Flags().StringVar(&id, "id", "", "Team id")
	command.Flags().StringVar(&name, "name", "", "Team name")
	command.Flags().StringVar(&description, "description", "", "Team description")
	_ = command.MarkFlagRequired("id")
	_ = command.MarkFlagRequired("name")
	return command
}

func newTeamUserCreateCommand() *cobra.Command {
	var teamID, id, name, email string
	command := &cobra.Command{
		Use:   "user-create",
		Short: "Create a user as an authenticated team administrator",
		RunE: func(cmd *cobra.Command, args []string) error {
			return callTeamRPC("team.user.create", map[string]any{
				"team_id":      teamID,
				"id":           id,
				"display_name": name,
				"email":        email,
			})
		},
	}
	command.Flags().StringVar(&teamID, "team", "", "Administered team id")
	command.Flags().StringVar(&id, "id", "", "User id")
	command.Flags().StringVar(&name, "name", "", "Display name")
	command.Flags().StringVar(&email, "email", "", "Email")
	_ = command.MarkFlagRequired("team")
	_ = command.MarkFlagRequired("id")
	_ = command.MarkFlagRequired("name")
	return command
}

func newTeamTokenIssueCommand() *cobra.Command {
	var userID, label, tokenFile, expiry string
	command := &cobra.Command{
		Use:   "token-issue",
		Short: "Issue a personal token and save its plaintext exactly once",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := generateAccessToken()
			if err != nil {
				return err
			}
			cleanup, err := stageSecretFile(tokenFile, token)
			if err != nil {
				return err
			}
			keep := false
			defer func() {
				if !keep {
					cleanup()
				}
			}()
			params := map[string]any{
				"user_id": userID,
				"label":   label,
				"token":   token,
			}
			if strings.TrimSpace(expiry) != "" {
				parsed, err := time.Parse(time.RFC3339, expiry)
				if err != nil {
					return fmt.Errorf("parse --expires: %w", err)
				}
				params["expires_at"] = parsed.UTC().Format(time.RFC3339)
			}
			result, err := callTeamRPCResult("team.token.issue", params)
			if err != nil {
				return err
			}
			keep = true
			return printTeamValue(map[string]any{
				"credential": result,
				"token_file": tokenFile,
			})
		},
	}
	command.Flags().StringVar(&userID, "user", "", "Target user id")
	command.Flags().StringVar(&label, "label", "workstation", "Credential label")
	command.Flags().StringVar(&tokenFile, "token-file", "", "New 0600 file for the generated token")
	command.Flags().StringVar(&expiry, "expires", "", "Optional RFC3339 expiry")
	_ = command.MarkFlagRequired("user")
	_ = command.MarkFlagRequired("token-file")
	return command
}

func newTeamMemberAddCommand() *cobra.Command {
	var teamID, userID, role string
	command := &cobra.Command{
		Use:   "member-add",
		Short: "Add a user to a team",
		RunE: func(cmd *cobra.Command, args []string) error {
			return callTeamRPC("team.member.add", map[string]any{
				"team_id": teamID,
				"user_id": userID,
				"role":    role,
			})
		},
	}
	command.Flags().StringVar(&teamID, "team", "", "Team id")
	command.Flags().StringVar(&userID, "user", "", "User id")
	command.Flags().StringVar(&role, "role", "member", "owner, admin, or member")
	_ = command.MarkFlagRequired("team")
	_ = command.MarkFlagRequired("user")
	return command
}

func newProjectCreateCommand() *cobra.Command {
	var teamID, id, key, name, description string
	command := &cobra.Command{
		Use:   "project-create",
		Short: "Create a project in a team",
		RunE: func(cmd *cobra.Command, args []string) error {
			return callTeamRPC("project.create", map[string]any{
				"team_id":     teamID,
				"id":          id,
				"key":         key,
				"name":        name,
				"description": description,
			})
		},
	}
	command.Flags().StringVar(&teamID, "team", "", "Team id")
	command.Flags().StringVar(&id, "id", "", "Project id")
	command.Flags().StringVar(&key, "key", "", "Short unique project key")
	command.Flags().StringVar(&name, "name", "", "Project name")
	command.Flags().StringVar(&description, "description", "", "Project description")
	_ = command.MarkFlagRequired("team")
	_ = command.MarkFlagRequired("id")
	_ = command.MarkFlagRequired("key")
	_ = command.MarkFlagRequired("name")
	return command
}

func newProjectMemberAddCommand() *cobra.Command {
	var projectID, userID, role string
	var domains []string
	var capacity int
	command := &cobra.Command{
		Use:   "project-member-add",
		Short: "Assign a user, business domains and capacity to a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			return callTeamRPC("project.member.add", map[string]any{
				"project_id":       projectID,
				"user_id":          userID,
				"role":             role,
				"business_domains": domains,
				"capacity_points":  capacity,
			})
		},
	}
	command.Flags().StringVar(&projectID, "project", "", "Project id")
	command.Flags().StringVar(&userID, "user", "", "User id")
	command.Flags().StringVar(&role, "role", "developer", "Project role")
	command.Flags().StringSliceVar(&domains, "domain", nil, "Owned business domain; repeat as needed")
	command.Flags().IntVar(&capacity, "capacity", 0, "Planned capacity points")
	_ = command.MarkFlagRequired("project")
	_ = command.MarkFlagRequired("user")
	return command
}

func newRepositoryCreateCommand() *cobra.Command {
	var projectID, id, name, remoteURL, localPath, branch string
	command := &cobra.Command{
		Use:   "repository-create",
		Short: "Register the server-resolved repository boundary for a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			return callTeamRPC("repository.create", map[string]any{
				"project_id":     projectID,
				"id":             id,
				"name":           name,
				"remote_url":     remoteURL,
				"local_path":     localPath,
				"default_branch": branch,
			})
		},
	}
	command.Flags().StringVar(&projectID, "project", "", "Project id")
	command.Flags().StringVar(&id, "id", "", "Repository id")
	command.Flags().StringVar(&name, "name", "", "Repository name")
	command.Flags().StringVar(&remoteURL, "remote", "", "Canonical Git remote URL")
	command.Flags().StringVar(&localPath, "local-path", "", "Absolute control-plane checkout path")
	command.Flags().StringVar(&branch, "branch", "main", "Default branch")
	_ = command.MarkFlagRequired("project")
	_ = command.MarkFlagRequired("id")
	_ = command.MarkFlagRequired("name")
	return command
}

func newTokenBudgetPutCommand() *cobra.Command {
	var projectID, id, userID string
	var limit int64
	command := &cobra.Command{
		Use:   "budget-put",
		Short: "Create or update a project/member token budget",
		RunE: func(cmd *cobra.Command, args []string) error {
			return callTeamRPC("budget.put", map[string]any{
				"id": id, "project_id": projectID,
				"user_id": userID, "limit_tokens": limit,
			})
		},
	}
	command.Flags().StringVar(&projectID, "project", "", "Project id")
	command.Flags().StringVar(&id, "id", "", "Stable budget id")
	command.Flags().StringVar(&userID, "user", "", "Optional project member id")
	command.Flags().Int64Var(&limit, "limit", 0, "Hard token limit")
	_ = command.MarkFlagRequired("project")
	_ = command.MarkFlagRequired("id")
	_ = command.MarkFlagRequired("limit")
	return command
}

func newControlSummaryCommand() *cobra.Command {
	var projectID string
	command := &cobra.Command{
		Use:   "control-summary",
		Short: "Show budget, knowledge, skill, Runner release and context totals",
		RunE: func(cmd *cobra.Command, args []string) error {
			return callTeamRPC(
				"control.summary",
				map[string]any{"project_id": projectID},
			)
		},
	}
	command.Flags().StringVar(&projectID, "project", "", "Project id")
	_ = command.MarkFlagRequired("project")
	return command
}

func newContextCompileCommand() *cobra.Command {
	var projectID, repositoryID, userID, budgetID string
	var knowledgeIDs, skillIDs []string
	command := &cobra.Command{
		Use:   "context-compile",
		Short: "Compile an immutable project context bundle from approved inputs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return callTeamRPC("context.compile", map[string]any{
				"project_id": projectID, "repository_id": repositoryID,
				"user_id": userID, "budget_id": budgetID,
				"knowledge_ids": knowledgeIDs, "skill_ids": skillIDs,
			})
		},
	}
	command.Flags().StringVar(&projectID, "project", "", "Project id")
	command.Flags().StringVar(&repositoryID, "repository", "", "Optional repository id")
	command.Flags().StringVar(&userID, "user", "", "Optional target project member")
	command.Flags().StringVar(&budgetID, "budget", "", "Optional budget id")
	command.Flags().StringSliceVar(&knowledgeIDs, "knowledge", nil, "Approved knowledge source id; repeat as needed")
	command.Flags().StringSliceVar(&skillIDs, "skill", nil, "Approved skill release id; repeat as needed")
	_ = command.MarkFlagRequired("project")
	return command
}

func callTeamRPC(method string, params map[string]any) error {
	result, err := callTeamRPCResult(method, params)
	if err != nil {
		return err
	}
	return printTeamValue(result)
}

func callTeamRPCResult(method string, params map[string]any) (any, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(os.Getenv("GOCLAW_USER_TOKEN")) == "" {
		return nil, errors.New("GOCLAW_USER_TOKEN is required for authenticated team operations")
	}
	return callGatewayRPC(cfg, method, params)
}

func generateAccessToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate access token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func stageSecretFile(path, value string) (func(), error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("token file is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(absolute), 0o700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(absolute, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create token file: %w", err)
	}
	remove := func() { _ = os.Remove(absolute) }
	if _, err := file.WriteString(value + "\n"); err != nil {
		_ = file.Close()
		remove()
		return nil, err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		remove()
		return nil, err
	}
	if err := file.Close(); err != nil {
		remove()
		return nil, err
	}
	return remove, nil
}

func printTeamValue(value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
