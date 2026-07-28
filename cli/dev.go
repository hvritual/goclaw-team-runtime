package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/smallnest/goclaw/config"
	"github.com/smallnest/goclaw/governance"
	dev "github.com/smallnest/goclaw/orchestratorlite"
	"github.com/spf13/cobra"
)

var (
	devRoot               string
	devRepo               string
	devProject            string
	devTeam               string
	devRepositoryID       string
	devModule             string
	devAssignee           string
	devParent             string
	devIssues             []string
	devSpecRefs           []string
	devDocRefs            []string
	devPolicyHash         string
	devPolicyInstructions []string
	devRequireTrace       bool
	devRequirePolicy      bool
	devRequireDocs        bool
	devPullRequestURL     string
	devCommitSHA          string
	devJSON               bool
	devSpecPath           string
	devTaskID             string
	devTitle              string
	devRequest            string
	devBaseRef            string
	devWaveStep           string
	devAllowed            []string
	devDenied             []string
	devMaxFiles           int
	devMaxLines           int
	devAllowDeps          bool
	devVerify             []string
	devKind               string
	devDecision           string
	devReviewer           string
	devComment            string
	devReason             string
	devRepairReason       string
	devForce              bool
	devQueuePriority      int
	devQueueCapabilities  []string
	devQueueMaxAttempts   int
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Run approval-gated development tasks with Orchestrator Lite",
}

var devInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the development event store and worktree root",
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := loadDevService()
		if err != nil {
			return err
		}
		return printDevValue(map[string]any{
			"status":        "initialized",
			"root":          service.Config().Root,
			"worktree_root": service.Config().WorktreeRoot,
			"repo_path":     service.Config().RepoPath,
		})
	},
}

var devCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Compile a request into a review-gated development task",
	RunE: func(cmd *cobra.Command, args []string) error {
		request, err := loadCreateRequest()
		if err != nil {
			return err
		}
		if teamGatewayMode() {
			if strings.TrimSpace(request.ID) == "" {
				return errors.New("--id is required in team mode for safe create retries")
			}
			if err := normalizeTeamCreateRequest(&request, devWaveStep); err != nil {
				return err
			}
			params, err := developmentRPCParams(request)
			if err != nil {
				return err
			}
			task, err := callTeamRPCResult("dev.task.create", params)
			if err != nil {
				return err
			}
			return printDevValue(task)
		}
		service, err := loadDevService()
		if err != nil {
			return err
		}
		task, err := service.CreateTask(request)
		if err != nil {
			return err
		}
		return printDevValue(task)
	},
}

var devListCmd = &cobra.Command{
	Use:   "list",
	Short: "List compiled development tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		if teamGatewayMode() {
			if strings.TrimSpace(devProject) == "" {
				return errors.New("--project is required in team mode")
			}
			tasks, err := callTeamRPCResult(
				"dev.tasks",
				map[string]any{"project_id": devProject},
			)
			if err != nil {
				return err
			}
			return printDevValue(tasks)
		}
		service, err := loadDevService()
		if err != nil {
			return err
		}
		tasks, err := service.ListTasks(devProject)
		if err != nil {
			return err
		}
		return printDevValue(tasks)
	},
}

var devShowCmd = &cobra.Command{
	Use:   "show TASK_ID",
	Short: "Show a task projection rebuilt from SessionEvents",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if teamGatewayMode() {
			task, err := callTeamRPCResult(
				"dev.task.get",
				map[string]any{"id": args[0]},
			)
			if err != nil {
				return err
			}
			return printDevValue(task)
		}
		service, err := loadDevService()
		if err != nil {
			return err
		}
		task, err := service.GetTask(args[0])
		if err != nil {
			return err
		}
		return printDevValue(task)
	},
}

var devEventsCmd = &cobra.Command{
	Use:   "events TASK_ID",
	Short: "Verify and show the append-only SessionEvent chain",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if teamGatewayMode() {
			events, err := callTeamRPCResult(
				"dev.task.events",
				map[string]any{"id": args[0]},
			)
			if err != nil {
				return err
			}
			return printDevValue(events)
		}
		service, err := loadDevService()
		if err != nil {
			return err
		}
		events, err := service.ListEvents(args[0])
		if err != nil {
			return err
		}
		return printDevValue(events)
	},
}

var devReviewCmd = &cobra.Command{
	Use:   "review TASK_ID",
	Short: "Record Scenario, Capacity, Risk, or Cost human review",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if teamGatewayMode() {
			params := developmentReviewParams(args[0])
			params["kind"] = devKind
			params["decision"] = devDecision
			task, err := callTeamRPCResult("dev.task.review", params)
			if err != nil {
				return err
			}
			return printDevValue(task)
		}
		service, err := loadDevService()
		if err != nil {
			return err
		}
		kind := dev.ReviewKind(devKind)
		role := governance.RoleScenarioReview
		switch kind {
		case dev.ReviewCapacity:
			role = governance.RoleCapacityReview
		case dev.ReviewRisk:
			role = governance.RoleRiskReview
		case dev.ReviewCost:
			role = governance.RoleCostReview
		}
		review, err := cliReview(
			service.GovernancePolicy(),
			devReviewerOrDefault(),
			role,
			devComment,
		)
		if err != nil {
			return err
		}
		task, err := service.ReviewTaskWithReview(
			args[0],
			kind,
			dev.ReviewDecision(devDecision),
			review,
		)
		if err != nil {
			return err
		}
		return printDevValue(task)
	},
}

var devFreezeCmd = &cobra.Command{
	Use:   "freeze TASK_ID",
	Short: "Freeze the execution bundle after all four reviews pass",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if teamGatewayMode() {
			task, err := callTeamRPCResult(
				"dev.task.freeze",
				map[string]any{"id": args[0]},
			)
			if err != nil {
				return err
			}
			return printDevValue(task)
		}
		service, err := loadDevService()
		if err != nil {
			return err
		}
		task, err := service.FreezeTask(cmd.Context(), args[0], devReviewerOrDefault())
		if err != nil {
			return err
		}
		return printDevValue(task)
	},
}

var devReviseCmd = &cobra.Command{
	Use:   "revise TASK_ID",
	Short: "Apply a ChangeIntent and return the task to all four reviews",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(devReason) == "" {
			return errors.New("--reason is required")
		}
		if devSpecPath == "" {
			return errors.New("--spec with a replacement Task JSON document is required")
		}
		data, err := os.ReadFile(devSpecPath)
		if err != nil {
			return err
		}
		var replacement dev.Task
		if err := json.Unmarshal(data, &replacement); err != nil {
			return err
		}
		if teamGatewayMode() {
			expectedRevision, err := developmentCurrentRevision(args[0])
			if err != nil {
				return err
			}
			task, err := callTeamRPCResult("dev.task.revise", map[string]any{
				"id":                args[0],
				"reason":            devReason,
				"expected_revision": expectedRevision,
				"replacement":       replacement,
			})
			if err != nil {
				return err
			}
			return printDevValue(task)
		}
		service, err := loadDevService()
		if err != nil {
			return err
		}
		task, err := service.ReviseTask(args[0], devReviewerOrDefault(), devReason, replacement)
		if err != nil {
			return err
		}
		return printDevValue(task)
	},
}

var devRunCmd = &cobra.Command{
	Use:   "run TASK_ID",
	Short: "Execute a frozen task in an isolated Git worktree",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDevOperation(cmd, args[0], "run")
	},
}

var devEnqueueCmd = &cobra.Command{
	Use:   "enqueue TASK_ID",
	Short: "Build a server-trusted ExecutionPack from a frozen task and queue it for a workstation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := callTeamRPCResult(
			"dev.task.enqueue",
			developmentEnqueueParams(args[0]),
		)
		if err != nil {
			return err
		}
		return printDevValue(result)
	},
}

func developmentEnqueueParams(taskID string) map[string]any {
	return map[string]any{
		"task_id":      taskID,
		"priority":     devQueuePriority,
		"capabilities": append([]string(nil), devQueueCapabilities...),
		"max_attempts": devQueueMaxAttempts,
	}
}

var devRepairCmd = &cobra.Command{
	Use:   "repair TASK_ID",
	Short: "Repair a DoneGate failure; team mode creates a newly reviewed revision",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if teamGatewayMode() {
			if strings.TrimSpace(devRepairReason) == "" {
				return errors.New("--reason is required for a team repair revision")
			}
			expectedRevision, err := developmentCurrentRevision(args[0])
			if err != nil {
				return err
			}
			task, err := callTeamRPCResult("dev.task.revise", map[string]any{
				"id":                args[0],
				"reason":            devRepairReason,
				"expected_revision": expectedRevision,
			})
			if err != nil {
				return err
			}
			return printDevValue(task)
		}
		return runDevOperation(cmd, args[0], "repair")
	},
}

var devResumeCmd = &cobra.Command{
	Use:   "resume TASK_ID",
	Short: "Resume an interrupted task using its worktree and Codex thread",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDevOperation(cmd, args[0], "resume")
	},
}

var devAcceptCmd = &cobra.Command{
	Use:   "accept TASK_ID",
	Short: "Human-accept a passing, unchanged EvidencePackage",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if teamGatewayMode() {
			task, err := callTeamRPCResult(
				"dev.task.accept",
				developmentReviewParams(args[0]),
			)
			if err != nil {
				return err
			}
			return printDevValue(task)
		}
		service, err := loadDevService()
		if err != nil {
			return err
		}
		review, err := cliReview(
			service.GovernancePolicy(),
			devReviewerOrDefault(),
			governance.RoleTaskAccept,
			devComment,
		)
		if err != nil {
			return err
		}
		task, err := service.AcceptTaskWithReview(cmd.Context(), args[0], review)
		if err != nil {
			return err
		}
		return printDevValue(task)
	},
}

var devCommitCmd = &cobra.Command{
	Use:   "commit TASK_ID",
	Short: "Commit a human-accepted, unchanged task in its isolated branch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if teamGatewayMode() {
			return errors.New(
				"team mode never commits workstation evidence from the control plane; " +
					"apply the accepted patch in your normal Git workflow, then use " +
					"`goclaw dev link-pr TASK_ID --commit <SHA> --url <PR_URL>`",
			)
		}
		service, err := loadDevService()
		if err != nil {
			return err
		}
		task, err := service.CommitTask(cmd.Context(), args[0], devReviewerOrDefault(), devComment)
		if err != nil {
			return err
		}
		return printDevValue(task)
	},
}

var devLinkPRCmd = &cobra.Command{
	Use:   "link-pr TASK_ID",
	Short: "Link a committed task to its pull request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if teamGatewayMode() {
			if strings.TrimSpace(devCommitSHA) == "" {
				return errors.New("--commit is required in team mode")
			}
			task, err := callTeamRPCResult("dev.task.link-pr", map[string]any{
				"id":         args[0],
				"url":        devPullRequestURL,
				"commit_sha": devCommitSHA,
			})
			if err != nil {
				return err
			}
			return printDevValue(task)
		}
		service, err := loadDevService()
		if err != nil {
			return err
		}
		task, err := service.RecordPullRequest(args[0], devReviewerOrDefault(), devPullRequestURL)
		if err != nil {
			return err
		}
		return printDevValue(task)
	},
}

var devCancelCmd = &cobra.Command{
	Use:   "cancel TASK_ID",
	Short: "Cancel a task without deleting its audit trail",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if teamGatewayMode() {
			params := developmentReviewParams(args[0])
			params["reason"] = devReason
			task, err := callTeamRPCResult("dev.task.cancel", params)
			if err != nil {
				return err
			}
			return printDevValue(task)
		}
		service, err := loadDevService()
		if err != nil {
			return err
		}
		review, err := cliReview(
			service.GovernancePolicy(),
			devReviewerOrDefault(),
			governance.RoleTaskCancel,
			devComment,
		)
		if err != nil {
			return err
		}
		task, err := service.CancelTaskWithReview(args[0], devReason, review)
		if err != nil {
			return err
		}
		return printDevValue(task)
	},
}

func init() {
	devCmd.PersistentFlags().StringVar(&devRoot, "root", "", "Development runtime root override")
	devCmd.PersistentFlags().StringVar(&devRepo, "repo", "", "Git repository path override")
	devCmd.PersistentFlags().StringVar(&devProject, "project", "", "Project id")
	devCmd.PersistentFlags().BoolVar(&devJSON, "json", false, "Print machine-readable JSON")

	devCreateCmd.Flags().StringVar(&devSpecPath, "spec", "", "CreateRequest JSON file")
	devCreateCmd.Flags().StringVar(&devTaskID, "id", "", "Stable development task id")
	devCreateCmd.Flags().StringVar(&devTitle, "title", "", "Task title")
	devCreateCmd.Flags().StringVar(&devRequest, "request", "", "Raw development request")
	devCreateCmd.Flags().StringVar(&devBaseRef, "base", "HEAD", "Frozen Git base ref")
	devCreateCmd.Flags().StringVar(
		&devWaveStep,
		"wave-step",
		"",
		"Declared active Wave step; the team Gateway resolves and freezes all other Wave fields",
	)
	devCreateCmd.Flags().StringVar(&devTeam, "team", "", "Owning team id")
	devCreateCmd.Flags().StringVar(&devRepositoryID, "repository-id", "", "Registered repository id")
	devCreateCmd.Flags().StringVar(&devModule, "module", "", "Business or code module")
	devCreateCmd.Flags().StringVar(&devAssignee, "assignee", "", "Assigned team member id")
	devCreateCmd.Flags().StringVar(&devParent, "parent-task", "", "Parent task id")
	devCreateCmd.Flags().StringSliceVar(&devIssues, "issue", nil, "Linked issue id; repeat as needed")
	devCreateCmd.Flags().StringSliceVar(&devSpecRefs, "spec-ref", nil, "Frozen specification reference")
	devCreateCmd.Flags().StringSliceVar(&devDocRefs, "document-ref", nil, "Governed document reference")
	devCreateCmd.Flags().StringVar(&devPolicyHash, "policy-hash", "", "Resolved policy bundle SHA-256")
	devCreateCmd.Flags().StringSliceVar(&devPolicyInstructions, "policy-instruction", nil, "Frozen policy instruction")
	devCreateCmd.Flags().BoolVar(&devRequireTrace, "require-workitem-trace", false, "Reject unattributed changed files")
	devCreateCmd.Flags().BoolVar(&devRequirePolicy, "require-policy", false, "Require a frozen policy bundle")
	devCreateCmd.Flags().BoolVar(&devRequireDocs, "require-docs", false, "Require governed document evidence")
	devCreateCmd.Flags().StringSliceVar(&devAllowed, "allow-path", nil, "Approved path/glob; repeat as needed")
	devCreateCmd.Flags().StringSliceVar(&devDenied, "deny-path", nil, "Denied path/glob; repeat as needed")
	devCreateCmd.Flags().IntVar(&devMaxFiles, "max-files", 0, "Maximum changed files")
	devCreateCmd.Flags().IntVar(&devMaxLines, "max-lines", 0, "Maximum added plus deleted lines")
	devCreateCmd.Flags().BoolVar(&devAllowDeps, "allow-dependencies", false, "Allow dependency manifest changes")
	devCreateCmd.Flags().StringArrayVar(&devVerify, "verify", nil, `Verification argv as JSON, e.g. '["go","test","./..."]'`)

	devReviewCmd.Flags().StringVar(&devKind, "kind", "", "scenario, capacity, risk, or cost")
	devReviewCmd.Flags().StringVar(&devDecision, "decision", "", "approved or rejected")
	devReviewCmd.Flags().StringVar(&devReviewer, "reviewer", "", "Human reviewer identity")
	devReviewCmd.Flags().StringVar(&devComment, "comment", "", "Review comment")
	addDecisionFlags(devReviewCmd)

	devFreezeCmd.Flags().StringVar(&devReviewer, "reviewer", "", "Human freezer identity")
	devReviseCmd.Flags().StringVar(&devSpecPath, "spec", "", "Replacement Task JSON file")
	devReviseCmd.Flags().StringVar(&devReason, "reason", "", "ChangeIntent reason")
	devReviseCmd.Flags().StringVar(&devReviewer, "reviewer", "", "Human change author")
	devAcceptCmd.Flags().StringVar(&devReviewer, "reviewer", "", "Human reviewer identity")
	devAcceptCmd.Flags().StringVar(&devComment, "comment", "", "Acceptance comment")
	addDecisionFlags(devAcceptCmd)
	devCommitCmd.Flags().StringVar(&devReviewer, "reviewer", "", "Human committer identity")
	devCommitCmd.Flags().StringVar(&devComment, "message", "", "Git commit message")
	devLinkPRCmd.Flags().StringVar(&devReviewer, "reviewer", "", "Human actor identity")
	devLinkPRCmd.Flags().StringVar(&devPullRequestURL, "url", "", "Absolute pull request URL")
	devLinkPRCmd.Flags().StringVar(
		&devCommitSHA,
		"commit",
		"",
		"External Git commit containing the exact accepted workstation patch",
	)
	for _, command := range []*cobra.Command{devRunCmd, devRepairCmd, devResumeCmd} {
		command.Flags().StringVar(&devReviewer, "reviewer", "", "Local execution actor identity")
	}
	devRepairCmd.Flags().StringVar(
		&devRepairReason,
		"reason",
		"",
		"ChangeIntent reason for the new team repair revision",
	)
	devCancelCmd.Flags().StringVar(&devReviewer, "reviewer", "", "Human actor identity")
	devCancelCmd.Flags().StringVar(&devReason, "reason", "", "Cancellation reason")
	devCancelCmd.Flags().StringVar(&devComment, "comment", "", "Cancellation decision rationale")
	addDecisionFlags(devCancelCmd)
	devResumeCmd.Flags().BoolVar(&devForce, "force", false, "Remove a stale task run lock after confirming the previous process stopped")
	devEnqueueCmd.Flags().IntVar(&devQueuePriority, "priority", 0, "Queue priority; higher values run first")
	devEnqueueCmd.Flags().StringSliceVar(&devQueueCapabilities, "capability", []string{"codex"}, "Required workstation capability")
	devEnqueueCmd.Flags().IntVar(&devQueueMaxAttempts, "max-attempts", 0, "Maximum attempts; zero uses the scheduler default")

	devCmd.AddCommand(
		devInitCmd,
		devCreateCmd,
		devListCmd,
		devShowCmd,
		devEventsCmd,
		devReviewCmd,
		devReviseCmd,
		devFreezeCmd,
		devEnqueueCmd,
		devRunCmd,
		devRepairCmd,
		devResumeCmd,
		devAcceptCmd,
		devCommitCmd,
		devLinkPRCmd,
		devCancelCmd,
	)
	rootCmd.AddCommand(devCmd)
}

func loadDevService() (*dev.Service, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, err
	}
	development := cfg.Development
	development.Enabled = true
	if devRoot != "" {
		development.Root = devRoot
	}
	if devRepo != "" {
		development.RepoPath = devRepo
	}
	service, err := dev.NewService(development)
	if err != nil {
		return nil, err
	}
	service.SetGovernancePolicy(cfg.Governance)
	return service, nil
}

func loadCreateRequest() (dev.CreateRequest, error) {
	if devSpecPath != "" {
		data, err := os.ReadFile(devSpecPath)
		if err != nil {
			return dev.CreateRequest{}, err
		}
		var request dev.CreateRequest
		if err := json.Unmarshal(data, &request); err != nil {
			return dev.CreateRequest{}, err
		}
		if request.RepoPath == "" {
			request.RepoPath = devRepo
		}
		if request.ProjectID == "" {
			request.ProjectID = devProject
		}
		if request.ID == "" {
			request.ID = devTaskID
		}
		applyDevTeamFlags(&request)
		return request, nil
	}
	if strings.TrimSpace(devTitle) == "" || strings.TrimSpace(devRequest) == "" {
		return dev.CreateRequest{}, errors.New("--title and --request are required without --spec")
	}
	commands, err := parseVerificationCommands(devVerify)
	if err != nil {
		return dev.CreateRequest{}, err
	}
	if len(commands) == 0 {
		return dev.CreateRequest{}, errors.New("at least one --verify JSON argv command is required")
	}
	return dev.CreateRequest{
		ID:                 devTaskID,
		TeamID:             devTeam,
		ProjectID:          valueOrCLI(devProject, "default"),
		RepositoryID:       devRepositoryID,
		Module:             devModule,
		AssigneeID:         devAssignee,
		ParentTaskID:       devParent,
		IssueIDs:           append([]string(nil), devIssues...),
		SpecRefs:           append([]string(nil), devSpecRefs...),
		DocumentRefs:       append([]string(nil), devDocRefs...),
		PolicyBundleHash:   devPolicyHash,
		PolicyInstructions: append([]string(nil), devPolicyInstructions...),
		Title:              devTitle,
		RepoPath:           devRepo,
		BaseRef:            devBaseRef,
		Request: dev.RequestFrame{
			RawRequest: devRequest,
			Source:     "cli",
		},
		Goal: dev.GoalSpec{
			Objective:    devRequest,
			SuccessTests: commandLabels(commands),
		},
		Plan: dev.PlanSpec{
			Summary: "Implement the approved request in one isolated worktree.",
			Milestones: []dev.Milestone{{
				ID:    "implementation",
				Title: "Implementation and verification",
				WorkItems: []dev.WorkItem{{
					ID:                   "implementation",
					Title:                devTitle,
					Instructions:         devRequest,
					VerificationCommands: commands,
				}},
			}},
		},
		EvidencePlan: dev.EvidencePlan{Commands: commands},
		Scope: dev.ScopePolicy{
			AllowedPaths:       devAllowed,
			DeniedPaths:        devDenied,
			MaxChangedFiles:    devMaxFiles,
			MaxChangedLines:    devMaxLines,
			AllowNewDependency: devAllowDeps,
		},
		DoneGate: dev.DoneGateSpec{
			RequireWorkItemTrace:    devRequireTrace,
			RequirePolicyBundle:     devRequirePolicy,
			RequireDocumentEvidence: devRequireDocs,
		},
		CreatedBy: devReviewerOrDefault(),
	}, nil
}

func applyDevTeamFlags(request *dev.CreateRequest) {
	if request.TeamID == "" {
		request.TeamID = devTeam
	}
	if request.RepositoryID == "" {
		request.RepositoryID = devRepositoryID
	}
	if request.Module == "" {
		request.Module = devModule
	}
	if request.AssigneeID == "" {
		request.AssigneeID = devAssignee
	}
	if request.ParentTaskID == "" {
		request.ParentTaskID = devParent
	}
	if len(request.IssueIDs) == 0 {
		request.IssueIDs = append([]string(nil), devIssues...)
	}
	if len(request.SpecRefs) == 0 {
		request.SpecRefs = append([]string(nil), devSpecRefs...)
	}
	if len(request.DocumentRefs) == 0 {
		request.DocumentRefs = append([]string(nil), devDocRefs...)
	}
	if request.PolicyBundleHash == "" {
		request.PolicyBundleHash = devPolicyHash
	}
	if len(request.PolicyInstructions) == 0 {
		request.PolicyInstructions = append([]string(nil), devPolicyInstructions...)
	}
	if devRequireTrace {
		request.DoneGate.RequireWorkItemTrace = true
	}
	if devRequirePolicy {
		request.DoneGate.RequirePolicyBundle = true
	}
	if devRequireDocs {
		request.DoneGate.RequireDocumentEvidence = true
	}
}

func normalizeTeamCreateRequest(request *dev.CreateRequest, flagStepID string) error {
	if request == nil {
		return errors.New("development create request is required")
	}
	stepID := strings.TrimSpace(flagStepID)
	if stepID == "" && request.Wave != nil {
		stepID = strings.TrimSpace(request.Wave.StepID)
	}
	if stepID == "" {
		return errors.New(
			"--wave-step is required in team mode; the Gateway resolves the active Wave plan and frozen Git base",
		)
	}
	// Never send client-supplied plan paths, revisions, or hashes. The Gateway
	// derives those values from the registered repository at the exact base
	// commit and treats only the step intent as client input.
	request.Wave = &dev.WaveBinding{StepID: stepID}
	return nil
}

func parseVerificationCommands(values []string) ([]dev.CommandSpec, error) {
	commands := make([]dev.CommandSpec, 0, len(values))
	for index, value := range values {
		var argv []string
		if err := json.Unmarshal([]byte(value), &argv); err != nil {
			return nil, fmt.Errorf("--verify %d must be a JSON string array: %w", index+1, err)
		}
		if len(argv) == 0 || strings.TrimSpace(argv[0]) == "" {
			return nil, fmt.Errorf("--verify %d has empty argv", index+1)
		}
		commands = append(commands, dev.CommandSpec{
			Name: strings.Join(argv, " "),
			Argv: argv,
		})
	}
	return commands, nil
}

func commandLabels(commands []dev.CommandSpec) []string {
	result := make([]string, 0, len(commands))
	for _, command := range commands {
		result = append(result, command.Name)
	}
	return result
}

func runDevOperation(cmd *cobra.Command, id, operation string) error {
	if teamGatewayMode() {
		return errors.New(
			"team mode executes frozen work with `goclaw dev enqueue` and `goclaw runner work`; local run/resume is disabled",
		)
	}
	service, err := loadDevService()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), 24*time.Hour)
	defer cancel()
	var task dev.Task
	switch operation {
	case "run":
		task, err = service.RunTask(ctx, id, devReviewerOrDefault())
	case "repair":
		task, err = service.RepairTask(ctx, id, devReviewerOrDefault())
	case "resume":
		task, err = service.ResumeTask(ctx, id, devReviewerOrDefault(), devForce)
	default:
		err = fmt.Errorf("unsupported operation %s", operation)
	}
	if err != nil {
		return err
	}
	return printDevValue(task)
}

func teamGatewayMode() bool {
	return strings.TrimSpace(os.Getenv("GOCLAW_USER_TOKEN")) != ""
}

func developmentRPCParams(value any) (map[string]any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var params map[string]any
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, err
	}
	return params, nil
}

func developmentReviewParams(taskID string) map[string]any {
	return map[string]any{
		"id":              taskID,
		"reviewer_id":     devReviewer,
		"reviewer_token":  os.Getenv("GOCLAW_REVIEWER_TOKEN"),
		"rationale":       devComment,
		"counterargument": decisionCounterargument,
		"evidence_refs":   append([]string(nil), decisionEvidence...),
	}
}

func developmentCurrentRevision(taskID string) (int, error) {
	current, err := callTeamRPCResult(
		"dev.task.get",
		map[string]any{"id": taskID},
	)
	if err != nil {
		return 0, err
	}
	var task dev.Task
	if err := remarshal(current, &task); err != nil {
		return 0, err
	}
	if task.Compile.Revision <= 0 {
		return 0, errors.New("development task has no current revision")
	}
	return task.Compile.Revision, nil
}

func devReviewerOrDefault() string {
	if devReviewer != "" {
		return devReviewer
	}
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	return "human"
}

func printDevValue(value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if devJSON {
		fmt.Println(string(data))
		return nil
	}
	fmt.Println(string(data))
	return nil
}

func valueOrCLI(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
