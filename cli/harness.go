package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/smallnest/goclaw/config"
	"github.com/smallnest/goclaw/governance"
	harnesspkg "github.com/smallnest/goclaw/harness"
	"github.com/spf13/cobra"
)

var (
	harnessRoot      string
	harnessProjectID string
	harnessJSON      bool

	experimentBase        string
	experimentCandidate   string
	experimentTargets     []string
	experimentEvidence    []string
	experimentRootCause   string
	experimentSummary     string
	experimentFixTags     []string
	experimentRegressions []string
	experimentExecute     bool
	experimentReviewer    string
	experimentComment     string
	knowledgeStatus       string
)

var harnessCmd = &cobra.Command{
	Use:   "harness",
	Short: "Manage the Better-Harness eval and promotion loop",
}

var harnessInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a seed harness, evals, traces, and registry",
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := loadHarnessService()
		if err != nil {
			return err
		}
		active, err := service.ActiveState()
		if err != nil {
			return err
		}
		return printHarnessValue(map[string]any{
			"status":  "initialized",
			"root":    service.Config().Root,
			"project": service.ProjectID(),
			"active":  active,
		})
	},
}

var harnessStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show active harness and acceptance thresholds",
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := loadHarnessService()
		if err != nil {
			return err
		}
		active, err := service.ActiveState()
		if err != nil {
			return err
		}
		manifest, err := service.ActiveManifest()
		if err != nil {
			return err
		}
		return printHarnessValue(map[string]any{"active": active, "manifest": manifest})
	},
}

var harnessVersionsCmd = &cobra.Command{
	Use:   "versions",
	Short: "List immutable harness versions",
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := loadHarnessService()
		if err != nil {
			return err
		}
		value, err := service.ListVersions()
		if err != nil {
			return err
		}
		return printHarnessValue(value)
	},
}

var harnessEvalsCmd = &cobra.Command{
	Use:   "evals",
	Short: "List optimization, holdout, and golden evals",
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := loadHarnessService()
		if err != nil {
			return err
		}
		value, err := service.ListEvalCases()
		if err != nil {
			return err
		}
		return printHarnessValue(value)
	},
}

var harnessTracesCmd = &cobra.Command{
	Use:   "traces",
	Short: "List recent harness traces",
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := loadHarnessService()
		if err != nil {
			return err
		}
		value, err := service.ListTraces(service.ProjectID(), 100)
		if err != nil {
			return err
		}
		return printHarnessValue(value)
	},
}

var harnessExperimentCmd = &cobra.Command{
	Use:   "experiment",
	Short: "Create, validate, review, and promote harness candidates",
}

var harnessExperimentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List harness experiments",
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := loadHarnessService()
		if err != nil {
			return err
		}
		value, err := service.ListExperiments()
		if err != nil {
			return err
		}
		return printHarnessValue(value)
	},
}

var harnessExperimentCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an isolated candidate from the active harness",
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := loadHarnessService()
		if err != nil {
			return err
		}
		if experimentRootCause == "" || experimentSummary == "" {
			return fmt.Errorf("--root-cause and --summary are required")
		}
		value, err := service.CreateExperiment(experimentBase, experimentCandidate, harnesspkg.ChangeManifest{
			TargetComponents:    experimentTargets,
			EvidenceTraceIDs:    experimentEvidence,
			RootCause:           experimentRootCause,
			ChangeSummary:       experimentSummary,
			ExpectedFixTags:     experimentFixTags,
			PossibleRegressions: experimentRegressions,
		})
		if err != nil {
			return err
		}
		return printHarnessValue(value)
	},
}

var harnessExperimentValidateCmd = &cobra.Command{
	Use:   "validate EXPERIMENT_ID",
	Short: "Run optimization, golden, and holdout evals",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := loadHarnessService()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		value, err := service.ValidateExperiment(ctx, args[0], experimentExecute)
		if err != nil {
			return err
		}
		return printHarnessValue(value)
	},
}

var harnessExperimentApproveCmd = &cobra.Command{
	Use:   "approve EXPERIMENT_ID",
	Short: "Record explicit human approval",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := loadHarnessService()
		if err != nil {
			return err
		}
		review, err := cliReview(
			service.GovernancePolicy(),
			reviewerOrDefault(),
			governance.RoleHarnessApprove,
			experimentComment,
		)
		if err != nil {
			return err
		}
		value, err := service.ApproveExperimentWithReview(args[0], review)
		if err != nil {
			return err
		}
		return printHarnessValue(value)
	},
}

var harnessExperimentRejectCmd = &cobra.Command{
	Use:   "reject EXPERIMENT_ID",
	Short: "Reject a harness candidate",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := loadHarnessService()
		if err != nil {
			return err
		}
		review, err := cliReview(
			service.GovernancePolicy(),
			reviewerOrDefault(),
			governance.RoleHarnessApprove,
			experimentComment,
		)
		if err != nil {
			return err
		}
		value, err := service.RejectExperimentWithReview(args[0], review)
		if err != nil {
			return err
		}
		return printHarnessValue(value)
	},
}

var harnessExperimentPromoteCmd = &cobra.Command{
	Use:   "promote EXPERIMENT_ID",
	Short: "Promote a human-approved candidate to production",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := loadHarnessService()
		if err != nil {
			return err
		}
		review, err := cliReview(
			service.GovernancePolicy(),
			reviewerOrDefault(),
			governance.RoleHarnessPromote,
			experimentComment,
		)
		if err != nil {
			return err
		}
		value, err := service.PromoteExperimentWithReview(args[0], review)
		if err != nil {
			return err
		}
		return printHarnessValue(value)
	},
}

var harnessRollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Atomically return to the previous Harness version with an auditable decision",
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := loadHarnessService()
		if err != nil {
			return err
		}
		review, err := cliReview(
			service.GovernancePolicy(),
			reviewerOrDefault(),
			governance.RoleHarnessRollback,
			experimentComment,
		)
		if err != nil {
			return err
		}
		value, err := service.RollbackWithReview(review)
		if err != nil {
			return err
		}
		return printHarnessValue(value)
	},
}

var harnessKnowledgeCmd = &cobra.Command{
	Use:   "knowledge",
	Short: "Review approval-gated Obsidian knowledge proposals",
}

var harnessKnowledgeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List knowledge proposals",
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := loadHarnessService()
		if err != nil {
			return err
		}
		value, err := service.ListKnowledgeProposals(harnesspkg.KnowledgeProposalStatus(knowledgeStatus))
		if err != nil {
			return err
		}
		return printHarnessValue(value)
	},
}

var harnessKnowledgeApproveCmd = &cobra.Command{
	Use:   "approve PROPOSAL_ID",
	Short: "Apply a conflict-free knowledge proposal to the Vault",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := loadHarnessService()
		if err != nil {
			return err
		}
		review, err := cliReview(
			service.GovernancePolicy(),
			reviewerOrDefault(),
			governance.RoleKnowledgeApprove,
			experimentComment,
		)
		if err != nil {
			return err
		}
		value, err := service.ApproveKnowledgeProposalWithReview(args[0], review)
		if err != nil {
			return err
		}
		return printHarnessValue(value)
	},
}

var harnessKnowledgeRejectCmd = &cobra.Command{
	Use:   "reject PROPOSAL_ID",
	Short: "Reject a pending knowledge proposal",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := loadHarnessService()
		if err != nil {
			return err
		}
		review, err := cliReview(
			service.GovernancePolicy(),
			reviewerOrDefault(),
			governance.RoleKnowledgeApprove,
			experimentComment,
		)
		if err != nil {
			return err
		}
		value, err := service.RejectKnowledgeProposalWithReview(args[0], review)
		if err != nil {
			return err
		}
		return printHarnessValue(value)
	},
}

func init() {
	harnessCmd.PersistentFlags().StringVar(&harnessRoot, "root", "", "Harness registry root (defaults to config or ~/.goclaw/harness)")
	harnessCmd.PersistentFlags().StringVar(&harnessProjectID, "project", "", "Project id override")
	harnessCmd.PersistentFlags().BoolVar(&harnessJSON, "json", false, "Print machine-readable JSON")

	harnessExperimentCreateCmd.Flags().StringVar(&experimentBase, "base", "", "Base harness version (defaults to active)")
	harnessExperimentCreateCmd.Flags().StringVar(&experimentCandidate, "candidate", "", "Candidate version")
	harnessExperimentCreateCmd.Flags().StringSliceVar(&experimentTargets, "target", nil, "Target component paths")
	harnessExperimentCreateCmd.Flags().StringSliceVar(&experimentEvidence, "evidence", nil, "Evidence trace ids")
	harnessExperimentCreateCmd.Flags().StringVar(&experimentRootCause, "root-cause", "", "Observed root cause")
	harnessExperimentCreateCmd.Flags().StringVar(&experimentSummary, "summary", "", "Targeted harness change")
	harnessExperimentCreateCmd.Flags().StringSliceVar(&experimentFixTags, "fix-tags", nil, "Expected eval tags to improve")
	harnessExperimentCreateCmd.Flags().StringSliceVar(&experimentRegressions, "regression-tags", nil, "Eval tags at risk")

	harnessExperimentValidateCmd.Flags().BoolVar(&experimentExecute, "execute", false, "Allow command-backed eval runners")

	for _, command := range []*cobra.Command{harnessExperimentApproveCmd, harnessExperimentRejectCmd} {
		command.Flags().StringVar(&experimentReviewer, "reviewer", "", "Human reviewer identity")
		command.Flags().StringVar(&experimentComment, "comment", "", "Review comment")
		addDecisionFlags(command)
	}
	harnessExperimentPromoteCmd.Flags().StringVar(&experimentReviewer, "reviewer", "", "Human reviewer identity")
	harnessExperimentPromoteCmd.Flags().StringVar(&experimentComment, "comment", "", "Promotion rationale")
	addDecisionFlags(harnessExperimentPromoteCmd)
	harnessRollbackCmd.Flags().StringVar(&experimentReviewer, "reviewer", "", "Human reviewer identity")
	harnessRollbackCmd.Flags().StringVar(&experimentComment, "comment", "", "Rollback rationale")
	addDecisionFlags(harnessRollbackCmd)
	harnessKnowledgeListCmd.Flags().StringVar(&knowledgeStatus, "status", "", "Filter: pending, approved, or rejected")
	for _, command := range []*cobra.Command{harnessKnowledgeApproveCmd, harnessKnowledgeRejectCmd} {
		command.Flags().StringVar(&experimentReviewer, "reviewer", "", "Human reviewer identity")
		command.Flags().StringVar(&experimentComment, "comment", "", "Review comment")
		addDecisionFlags(command)
	}

	harnessExperimentCmd.AddCommand(
		harnessExperimentListCmd,
		harnessExperimentCreateCmd,
		harnessExperimentValidateCmd,
		harnessExperimentApproveCmd,
		harnessExperimentRejectCmd,
		harnessExperimentPromoteCmd,
	)
	harnessKnowledgeCmd.AddCommand(
		harnessKnowledgeListCmd,
		harnessKnowledgeApproveCmd,
		harnessKnowledgeRejectCmd,
	)
	harnessCmd.AddCommand(
		harnessInitCmd,
		harnessStatusCmd,
		harnessVersionsCmd,
		harnessEvalsCmd,
		harnessTracesCmd,
		harnessExperimentCmd,
		harnessKnowledgeCmd,
		harnessRollbackCmd,
	)
	rootCmd.AddCommand(harnessCmd)
}

func loadHarnessService() (*harnesspkg.Service, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, err
	}
	harnessConfig := cfg.Harness
	harnessConfig.Enabled = true
	if harnessRoot != "" {
		harnessConfig.Root = harnessRoot
	}
	if harnessProjectID != "" {
		harnessConfig.ProjectID = harnessProjectID
	}
	service, err := harnesspkg.NewService(harnessConfig)
	if err != nil {
		return nil, err
	}
	service.SetGovernancePolicy(cfg.Governance)
	return service, nil
}

func reviewerOrDefault() string {
	if experimentReviewer != "" {
		return experimentReviewer
	}
	if value := os.Getenv("USER"); value != "" {
		return value
	}
	return "human"
}

func printHarnessValue(value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if harnessJSON {
		fmt.Println(string(data))
		return nil
	}
	fmt.Println(string(data))
	return nil
}
