package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/smallnest/goclaw/config"
	"github.com/smallnest/goclaw/governance"
	"github.com/smallnest/goclaw/integration/ouroborosprovider"
	dev "github.com/smallnest/goclaw/orchestratorlite"
	"github.com/smallnest/goclaw/ouroboros"
	"github.com/smallnest/goclaw/providers"
	"github.com/spf13/cobra"
)

var (
	ouroRoot         string
	ouroProject      string
	ouroTopic        string
	ouroJSON         bool
	ouroTitle        string
	ouroRequest      string
	ouroRepo         string
	ouroBase         string
	ouroContext      string
	ouroContextFile  string
	ouroBrownfield   bool
	ouroQuestionID   string
	ouroAnswer       string
	ouroAnswersFile  string
	ouroReviewer     string
	ouroComment      string
	ouroTaskID       string
	ouroReason       string
	ouroStakeholders []string
	ouroReady        bool
	ouroConflictID   string
	ouroResolution   string
	ouroOutcomeKind  string
	ouroConditionID  string
	ouroEvaluationID string
	ouroAccepted     bool
)

var ouroCmd = &cobra.Command{
	Use:   "ouroboros",
	Short: "Run the Go-native specification-first Ouroboros loop",
}

var ouroInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the Ouroboros event and immutable Seed stores",
	RunE: func(cmd *cobra.Command, args []string) error {
		service, _, closeProvider, err := loadOuroborosRuntime(false)
		if err != nil {
			return err
		}
		defer closeProvider()
		return printOuroValue(map[string]any{
			"status":                "initialized",
			"root":                  service.Config().Root,
			"ambiguity_threshold":   service.Config().AmbiguityThreshold,
			"convergence_threshold": service.Config().ConvergenceThreshold,
		})
	},
}

var ouroStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a Socratic interview from a vague development request",
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(ouroRequest) == "" {
			return errors.New("--request is required")
		}
		service, cfg, closeProvider, err := loadOuroborosRuntime(true)
		if err != nil {
			return err
		}
		defer closeProvider()
		contextSummary := ouroContext
		if ouroContextFile != "" {
			data, readErr := os.ReadFile(ouroContextFile)
			if readErr != nil {
				return readErr
			}
			contextSummary = string(data)
		}
		repo := ouroRepo
		if repo == "" {
			repo = cfg.Development.RepoPath
		}
		session, err := service.Start(cmd.Context(), ouroboros.StartRequest{
			ProjectID:      valueOrCLI(ouroProject, "default"),
			TopicID:        valueOrCLI(ouroTopic, "inbox"),
			Title:          ouroTitle,
			RepoPath:       repo,
			BaseRef:        valueOrCLI(ouroBase, "HEAD"),
			RawRequest:     ouroRequest,
			ContextSummary: contextSummary,
			Brownfield:     ouroBrownfield,
			CreatedBy:      ouroActor(),
			Stakeholders:   append([]string(nil), ouroStakeholders...),
		})
		if err != nil {
			return err
		}
		return printOuroValue(session)
	},
}

var ouroListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Ouroboros sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		service, _, closeProvider, err := loadOuroborosRuntime(false)
		if err != nil {
			return err
		}
		defer closeProvider()
		sessions, err := service.ListSessions(ouroProject)
		if err != nil {
			return err
		}
		return printOuroValue(sessions)
	},
}

var ouroShowCmd = &cobra.Command{
	Use:   "show SESSION_ID",
	Short: "Show an Ouroboros session rebuilt from its event chain",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, _, closeProvider, err := loadOuroborosRuntime(false)
		if err != nil {
			return err
		}
		defer closeProvider()
		session, err := service.GetSession(args[0])
		if err != nil {
			return err
		}
		return printOuroValue(session)
	},
}

var ouroEventsCmd = &cobra.Command{
	Use:   "events SESSION_ID",
	Short: "Verify and print the append-only Ouroboros event chain",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, _, closeProvider, err := loadOuroborosRuntime(false)
		if err != nil {
			return err
		}
		defer closeProvider()
		events, err := service.ListEvents(args[0])
		if err != nil {
			return err
		}
		return printOuroValue(events)
	},
}

var ouroSeedCmd = &cobra.Command{
	Use:   "seed HASH",
	Short: "Verify and print an immutable Seed by SHA-256",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, _, closeProvider, err := loadOuroborosRuntime(false)
		if err != nil {
			return err
		}
		defer closeProvider()
		seed, err := service.GetSeed(args[0])
		if err != nil {
			return err
		}
		return printOuroValue(seed)
	},
}

var ouroAnswerCmd = &cobra.Command{
	Use:   "answer SESSION_ID",
	Short: "Record human interview answers and reassess ambiguity",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		answers, err := loadOuroAnswers()
		if err != nil {
			return err
		}
		service, _, closeProvider, err := loadOuroborosRuntime(true)
		if err != nil {
			return err
		}
		defer closeProvider()
		session, err := service.Answer(cmd.Context(), args[0], ouroboros.AnswerRequest{
			Answers:  answers,
			Actor:    ouroActor(),
			Reassess: true,
		})
		if err != nil {
			return err
		}
		return printOuroValue(session)
	},
}

var ouroReassessCmd = modelSessionCommand(
	"reassess SESSION_ID",
	"Run another ambiguity assessment",
	func(cmd *cobra.Command, service *ouroboros.Service, id string) (ouroboros.Session, error) {
		return service.Reassess(cmd.Context(), id, ouroActor())
	},
)

var ouroCrystallizeCmd = modelSessionCommand(
	"crystallize SESSION_ID",
	"Crystallize a seed-ready interview into an immutable proposal",
	func(cmd *cobra.Command, service *ouroboros.Service, id string) (ouroboros.Session, error) {
		return service.Crystallize(cmd.Context(), id, ouroActor())
	},
)

var ouroApproveSeedCmd = &cobra.Command{
	Use:   "approve-seed SESSION_ID",
	Short: "Human-approve the pending immutable Seed",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, _, closeProvider, err := loadOuroborosRuntime(false)
		if err != nil {
			return err
		}
		defer closeProvider()
		review, err := cliReview(
			service.GovernancePolicy(),
			ouroReviewerOrDefault(),
			governance.RoleSeedApprove,
			ouroComment,
		)
		if err != nil {
			return err
		}
		session, err := service.ApproveSeedWithReview(args[0], review)
		if err != nil {
			return err
		}
		return printOuroValue(session)
	},
}

var ouroRejectSeedCmd = &cobra.Command{
	Use:   "reject-seed SESSION_ID",
	Short: "Reject the pending Seed and return to clarification",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, _, closeProvider, err := loadOuroborosRuntime(false)
		if err != nil {
			return err
		}
		defer closeProvider()
		review, err := cliReview(
			service.GovernancePolicy(),
			ouroReviewerOrDefault(),
			governance.RoleSeedApprove,
			ouroComment,
		)
		if err != nil {
			return err
		}
		session, err := service.RejectSeedWithReview(args[0], review)
		if err != nil {
			return err
		}
		return printOuroValue(session)
	},
}

var ouroCompileCmd = &cobra.Command{
	Use:   "compile SESSION_ID",
	Short: "Compile an approved Seed into a four-review Orchestrator Lite task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, cfg, closeProvider, err := loadOuroborosRuntime(false)
		if err != nil {
			return err
		}
		defer closeProvider()
		development, err := developmentFromConfig(cfg)
		if err != nil {
			return err
		}
		task, err := service.CompileTask(args[0], ouroReviewerOrDefault(), development)
		if err != nil {
			return err
		}
		return printOuroValue(task)
	},
}

var ouroEvaluateCmd = &cobra.Command{
	Use:   "evaluate SESSION_ID",
	Short: "Run mechanical, semantic, and independent consensus evaluation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if ouroTaskID == "" {
			return errors.New("--task is required")
		}
		service, cfg, closeProvider, err := loadOuroborosRuntime(true)
		if err != nil {
			return err
		}
		defer closeProvider()
		development, err := developmentFromConfig(cfg)
		if err != nil {
			return err
		}
		task, evidence, diff, err := readOuroEvidence(development, ouroTaskID)
		if err != nil {
			return err
		}
		session, err := service.EvaluateTask(
			cmd.Context(),
			args[0],
			ouroReviewerOrDefault(),
			task,
			evidence,
			diff,
		)
		if err != nil {
			return err
		}
		return printOuroValue(session)
	},
}

var ouroEvolveCmd = modelSessionCommand(
	"evolve SESSION_ID",
	"Generate a candidate-only successor Seed from the latest evaluation",
	func(cmd *cobra.Command, service *ouroboros.Service, id string) (ouroboros.Session, error) {
		return service.ProposeEvolution(cmd.Context(), id, ouroActor())
	},
)

var ouroApproveEvolutionCmd = &cobra.Command{
	Use:   "approve-evolution SESSION_ID",
	Short: "Human-approve a successor Seed candidate",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, _, closeProvider, err := loadOuroborosRuntime(false)
		if err != nil {
			return err
		}
		defer closeProvider()
		review, err := cliReview(
			service.GovernancePolicy(),
			ouroReviewerOrDefault(),
			governance.RoleEvolutionApprove,
			ouroComment,
		)
		if err != nil {
			return err
		}
		session, err := service.ApproveEvolutionWithReview(args[0], review)
		if err != nil {
			return err
		}
		return printOuroValue(session)
	},
}

var ouroRejectEvolutionCmd = &cobra.Command{
	Use:   "reject-evolution SESSION_ID",
	Short: "Reject a successor Seed candidate without mutating the active Seed",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, _, closeProvider, err := loadOuroborosRuntime(false)
		if err != nil {
			return err
		}
		defer closeProvider()
		review, err := cliReview(
			service.GovernancePolicy(),
			ouroReviewerOrDefault(),
			governance.RoleEvolutionApprove,
			ouroComment,
		)
		if err != nil {
			return err
		}
		session, err := service.RejectEvolutionWithReview(args[0], review)
		if err != nil {
			return err
		}
		return printOuroValue(session)
	},
}

var ouroResolveReadinessCmd = &cobra.Command{
	Use:   "resolve-readiness SESSION_ID",
	Short: "Resolve an assessor gray-zone escalation with an authenticated human decision",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, _, closeProvider, err := loadOuroborosRuntime(false)
		if err != nil {
			return err
		}
		defer closeProvider()
		review, err := cliReview(
			service.GovernancePolicy(),
			ouroReviewerOrDefault(),
			governance.RoleReadinessOverride,
			ouroComment,
		)
		if err != nil {
			return err
		}
		session, err := service.ResolveReadiness(args[0], review, ouroReady)
		if err != nil {
			return err
		}
		return printOuroValue(session)
	},
}

var ouroResolveConflictCmd = &cobra.Command{
	Use:   "resolve-conflict SESSION_ID",
	Short: "Resolve a preserved stakeholder conflict without silently merging claims",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if ouroConflictID == "" || ouroResolution == "" {
			return errors.New("--conflict and --resolution are required")
		}
		service, _, closeProvider, err := loadOuroborosRuntime(false)
		if err != nil {
			return err
		}
		defer closeProvider()
		review, err := cliReview(
			service.GovernancePolicy(),
			ouroReviewerOrDefault(),
			governance.RoleConflictResolve,
			ouroComment,
		)
		if err != nil {
			return err
		}
		session, err := service.ResolveConflict(args[0], ouroConflictID, ouroResolution, review)
		if err != nil {
			return err
		}
		return printOuroValue(session)
	},
}

var ouroResolveEvaluationCmd = &cobra.Command{
	Use:   "resolve-evaluation SESSION_ID",
	Short: "Adjudicate disputed evaluation evidence without accepting the task or deployment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(ouroEvaluationID) == "" {
			return errors.New("--evaluation is required")
		}
		service, _, closeProvider, err := loadOuroborosRuntime(false)
		if err != nil {
			return err
		}
		defer closeProvider()
		review, err := cliReview(
			service.GovernancePolicy(),
			ouroReviewerOrDefault(),
			governance.RoleEvaluationResolve,
			ouroComment,
		)
		if err != nil {
			return err
		}
		session, err := service.ResolveEvaluation(
			args[0],
			ouroEvaluationID,
			ouroAccepted,
			review,
		)
		if err != nil {
			return err
		}
		return printOuroValue(session)
	},
}

var ouroRecordOutcomeCmd = &cobra.Command{
	Use:   "record-outcome SESSION_ID",
	Short: "Record pass, failure, rollback, cancellation, or missing feedback in the denominator",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, _, closeProvider, err := loadOuroborosRuntime(false)
		if err != nil {
			return err
		}
		defer closeProvider()
		review, err := cliReview(
			service.GovernancePolicy(),
			ouroReviewerOrDefault(),
			governance.RoleOutcomeRecord,
			ouroComment,
		)
		if err != nil {
			return err
		}
		session, err := service.RecordOutcome(args[0], ouroboros.OutcomeRequest{
			Kind:         ouroOutcomeKind,
			TaskID:       ouroTaskID,
			Reason:       ouroReason,
			EvidenceRefs: append([]string(nil), decisionEvidence...),
			Review:       review,
		})
		if err != nil {
			return err
		}
		return printOuroValue(session)
	},
}

var ouroTriggerKillCmd = &cobra.Command{
	Use:   "trigger-kill SESSION_ID",
	Short: "Fail closed when a preregistered kill condition is observed",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, _, closeProvider, err := loadOuroborosRuntime(false)
		if err != nil {
			return err
		}
		defer closeProvider()
		review, err := cliReview(
			service.GovernancePolicy(),
			ouroReviewerOrDefault(),
			governance.RoleKillSwitch,
			ouroComment,
		)
		if err != nil {
			return err
		}
		session, err := service.TriggerKillCondition(
			args[0],
			ouroConditionID,
			ouroReason,
			decisionEvidence,
			review,
		)
		if err != nil {
			return err
		}
		return printOuroValue(session)
	},
}

var ouroReferenceClassCmd = &cobra.Command{
	Use:   "reference-class",
	Short: "Show project outcomes including failures, rollbacks, cancellations, and no-feedback cases",
	RunE: func(cmd *cobra.Command, args []string) error {
		service, _, closeProvider, err := loadOuroborosRuntime(false)
		if err != nil {
			return err
		}
		defer closeProvider()
		stats, err := service.ReferenceClass(ouroProject)
		if err != nil {
			return err
		}
		return printOuroValue(stats)
	},
}

var ouroCancelCmd = &cobra.Command{
	Use:   "cancel SESSION_ID",
	Short: "Cancel a session while preserving its complete audit trail",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, _, closeProvider, err := loadOuroborosRuntime(false)
		if err != nil {
			return err
		}
		defer closeProvider()
		review, err := cliReview(
			service.GovernancePolicy(),
			ouroReviewerOrDefault(),
			governance.RoleSessionCancel,
			ouroComment,
		)
		if err != nil {
			return err
		}
		session, err := service.CancelWithReview(args[0], ouroReason, review)
		if err != nil {
			return err
		}
		return printOuroValue(session)
	},
}

func init() {
	ouroCmd.PersistentFlags().StringVar(&ouroRoot, "root", "", "Ouroboros runtime root override")
	ouroCmd.PersistentFlags().StringVar(&ouroProject, "project", "", "Project id")
	ouroCmd.PersistentFlags().BoolVar(&ouroJSON, "json", false, "Print machine-readable JSON")

	ouroStartCmd.Flags().StringVar(&ouroTopic, "topic", "inbox", "Project topic id")
	ouroStartCmd.Flags().StringVar(&ouroTitle, "title", "", "Interview title")
	ouroStartCmd.Flags().StringVar(&ouroRequest, "request", "", "Raw development request")
	ouroStartCmd.Flags().StringVar(&ouroRepo, "repo", "", "Target Git repository")
	ouroStartCmd.Flags().StringVar(&ouroBase, "base", "HEAD", "Target Git base ref")
	ouroStartCmd.Flags().StringVar(&ouroContext, "context", "", "Existing project context summary")
	ouroStartCmd.Flags().StringVar(&ouroContextFile, "context-file", "", "Read project context summary from a file")
	ouroStartCmd.Flags().BoolVar(&ouroBrownfield, "brownfield", true, "Score existing-codebase context")
	ouroStartCmd.Flags().StringSliceVar(&ouroStakeholders, "stakeholder", nil, "Named stakeholder; repeat as needed")

	ouroAnswerCmd.Flags().StringVar(&ouroQuestionID, "question", "", "Question id")
	ouroAnswerCmd.Flags().StringVar(&ouroAnswer, "answer", "", "Explicit human answer")
	ouroAnswerCmd.Flags().StringVar(&ouroAnswersFile, "answers", "", "JSON array or {answers: []} file")

	for _, command := range []*cobra.Command{
		ouroApproveSeedCmd,
		ouroRejectSeedCmd,
		ouroApproveEvolutionCmd,
		ouroRejectEvolutionCmd,
	} {
		command.Flags().StringVar(&ouroReviewer, "reviewer", "", "Human reviewer identity")
		command.Flags().StringVar(&ouroComment, "comment", "", "Review rationale")
		addDecisionFlags(command)
	}
	ouroResolveReadinessCmd.Flags().StringVar(&ouroReviewer, "reviewer", "", "Human reviewer identity")
	ouroResolveReadinessCmd.Flags().StringVar(&ouroComment, "comment", "", "Decision rationale")
	ouroResolveReadinessCmd.Flags().BoolVar(&ouroReady, "ready", false, "Declare the gray-zone assessment ready")
	addDecisionFlags(ouroResolveReadinessCmd)
	ouroResolveConflictCmd.Flags().StringVar(&ouroReviewer, "reviewer", "", "Human resolver identity")
	ouroResolveConflictCmd.Flags().StringVar(&ouroComment, "comment", "", "Decision rationale")
	ouroResolveConflictCmd.Flags().StringVar(&ouroConflictID, "conflict", "", "Conflict id")
	ouroResolveConflictCmd.Flags().StringVar(&ouroResolution, "resolution", "", "Explicit conflict resolution")
	addDecisionFlags(ouroResolveConflictCmd)
	ouroResolveEvaluationCmd.Flags().StringVar(&ouroReviewer, "reviewer", "", "Human evidence adjudicator identity")
	ouroResolveEvaluationCmd.Flags().StringVar(&ouroComment, "comment", "", "Evidence disposition rationale")
	ouroResolveEvaluationCmd.Flags().StringVar(&ouroEvaluationID, "evaluation", "", "Disputed evaluation id")
	ouroResolveEvaluationCmd.Flags().BoolVar(&ouroAccepted, "accept", false, "Accept the disputed evaluation evidence")
	addDecisionFlags(ouroResolveEvaluationCmd)
	ouroRecordOutcomeCmd.Flags().StringVar(&ouroReviewer, "reviewer", "", "Human recorder identity")
	ouroRecordOutcomeCmd.Flags().StringVar(&ouroComment, "comment", "", "Why this outcome record is trustworthy")
	ouroRecordOutcomeCmd.Flags().StringVar(&ouroOutcomeKind, "kind", "", "passed, failed, cancelled, rolled_back, or no_feedback")
	ouroRecordOutcomeCmd.Flags().StringVar(&ouroTaskID, "task", "", "Related task id")
	ouroRecordOutcomeCmd.Flags().StringVar(&ouroReason, "reason", "", "Observed outcome")
	addDecisionFlags(ouroRecordOutcomeCmd)
	ouroTriggerKillCmd.Flags().StringVar(&ouroReviewer, "reviewer", "", "Human reviewer identity")
	ouroTriggerKillCmd.Flags().StringVar(&ouroComment, "comment", "", "Decision rationale")
	ouroTriggerKillCmd.Flags().StringVar(&ouroConditionID, "condition", "", "Kill condition id")
	ouroTriggerKillCmd.Flags().StringVar(&ouroReason, "reason", "", "Observed trigger")
	addDecisionFlags(ouroTriggerKillCmd)
	ouroCompileCmd.Flags().StringVar(&ouroReviewer, "reviewer", "", "Human compiler identity")
	ouroEvaluateCmd.Flags().StringVar(&ouroReviewer, "reviewer", "", "Human evaluator identity")
	ouroEvaluateCmd.Flags().StringVar(&ouroTaskID, "task", "", "Compiled Orchestrator Lite task id")
	ouroCancelCmd.Flags().StringVar(&ouroReviewer, "reviewer", "", "Human actor identity")
	ouroCancelCmd.Flags().StringVar(&ouroReason, "reason", "", "Cancellation reason")
	ouroCancelCmd.Flags().StringVar(&ouroComment, "comment", "", "Cancellation decision rationale")
	addDecisionFlags(ouroCancelCmd)

	ouroCmd.AddCommand(
		ouroInitCmd,
		ouroStartCmd,
		ouroListCmd,
		ouroShowCmd,
		ouroEventsCmd,
		ouroSeedCmd,
		ouroAnswerCmd,
		ouroReassessCmd,
		ouroCrystallizeCmd,
		ouroApproveSeedCmd,
		ouroRejectSeedCmd,
		ouroCompileCmd,
		ouroEvaluateCmd,
		ouroEvolveCmd,
		ouroApproveEvolutionCmd,
		ouroRejectEvolutionCmd,
		ouroResolveReadinessCmd,
		ouroResolveConflictCmd,
		ouroResolveEvaluationCmd,
		ouroRecordOutcomeCmd,
		ouroTriggerKillCmd,
		ouroReferenceClassCmd,
		ouroCancelCmd,
	)
	rootCmd.AddCommand(ouroCmd)
}

func modelSessionCommand(
	use, short string,
	action func(*cobra.Command, *ouroboros.Service, string) (ouroboros.Session, error),
) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, _, closeProvider, err := loadOuroborosRuntime(true)
			if err != nil {
				return err
			}
			defer closeProvider()
			session, err := action(cmd, service, args[0])
			if err != nil {
				return err
			}
			return printOuroValue(session)
		},
	}
}

func loadOuroborosRuntime(needsModel bool) (*ouroboros.Service, *config.Config, func(), error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, nil, func() {}, err
	}
	runtimeConfig := cfg.Ouroboros
	runtimeConfig.Enabled = true
	if ouroRoot != "" {
		runtimeConfig.Root = ouroRoot
	}
	service, err := ouroboros.NewService(runtimeConfig)
	if err != nil {
		return nil, nil, func() {}, err
	}
	service.SetGovernancePolicy(cfg.Governance)
	if !needsModel {
		return service, cfg, func() {}, nil
	}
	provider, err := providers.NewProvider(cfg)
	if err != nil {
		return nil, nil, func() {}, err
	}
	adapter, err := ouroborosprovider.New(provider)
	if err != nil {
		_ = provider.Close()
		return nil, nil, func() {}, err
	}
	service.SetModel(adapter)
	return service, cfg, func() { _ = provider.Close() }, nil
}

func developmentFromConfig(cfg *config.Config) (*dev.Service, error) {
	development := cfg.Development
	development.Enabled = true
	service, err := dev.NewService(development)
	if err != nil {
		return nil, err
	}
	service.SetGovernancePolicy(cfg.Governance)
	return service, nil
}

func loadOuroAnswers() ([]ouroboros.Answer, error) {
	if ouroAnswersFile == "" {
		if ouroQuestionID == "" || strings.TrimSpace(ouroAnswer) == "" {
			return nil, errors.New("--question and --answer, or --answers, are required")
		}
		return []ouroboros.Answer{{QuestionID: ouroQuestionID, Text: ouroAnswer}}, nil
	}
	data, err := os.ReadFile(ouroAnswersFile)
	if err != nil {
		return nil, err
	}
	var answers []ouroboros.Answer
	if json.Unmarshal(data, &answers) == nil {
		return answers, nil
	}
	var envelope struct {
		Answers []ouroboros.Answer `json:"answers"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	return envelope.Answers, nil
}

func readOuroEvidence(
	development *dev.Service,
	taskID string,
) (dev.Task, dev.EvidencePackage, string, error) {
	task, err := development.GetTask(taskID)
	if err != nil {
		return dev.Task{}, dev.EvidencePackage{}, "", err
	}
	if task.LastEvidence == "" {
		return dev.Task{}, dev.EvidencePackage{}, "", errors.New("task has no EvidencePackage")
	}
	evidencePath, err := resolveOuroEvidenceFile(
		development.Config().Root,
		task.LastEvidence,
		32*1024*1024,
	)
	if err != nil {
		return dev.Task{}, dev.EvidencePackage{}, "", err
	}
	data, err := os.ReadFile(evidencePath)
	if err != nil {
		return dev.Task{}, dev.EvidencePackage{}, "", err
	}
	var evidence dev.EvidencePackage
	if err := json.Unmarshal(data, &evidence); err != nil {
		return dev.Task{}, dev.EvidencePackage{}, "", err
	}
	diff := ""
	if evidence.DiffPath != "" {
		diffPath, resolveErr := resolveOuroEvidenceFile(
			development.Config().Root,
			evidence.DiffPath,
			8*1024*1024,
		)
		if resolveErr != nil {
			return dev.Task{}, dev.EvidencePackage{}, "", resolveErr
		}
		diffData, readErr := os.ReadFile(diffPath)
		if readErr != nil {
			return dev.Task{}, dev.EvidencePackage{}, "", readErr
		}
		diff = string(diffData)
	}
	return task, evidence, diff, nil
}

func resolveOuroEvidenceFile(root, candidate string, maximum int64) (string, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	resolvedRoot, err := filepath.EvalSymlinks(absoluteRoot)
	if err != nil {
		return "", fmt.Errorf("resolve development runtime root: %w", err)
	}
	absoluteCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	resolvedCandidate, err := filepath.EvalSymlinks(absoluteCandidate)
	if err != nil {
		return "", err
	}
	if resolvedCandidate != resolvedRoot &&
		!strings.HasPrefix(resolvedCandidate, resolvedRoot+string(filepath.Separator)) {
		return "", errors.New("evidence path escapes development runtime root")
	}
	info, err := os.Stat(resolvedCandidate)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("evidence path is not a regular file")
	}
	if maximum > 0 && info.Size() > maximum {
		return "", fmt.Errorf("evidence file exceeds %d bytes", maximum)
	}
	return resolvedCandidate, nil
}

func printOuroValue(value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func ouroActor() string {
	return ouroReviewerOrDefault()
}

func ouroReviewerOrDefault() string {
	if strings.TrimSpace(ouroReviewer) != "" {
		return strings.TrimSpace(ouroReviewer)
	}
	return "human"
}
