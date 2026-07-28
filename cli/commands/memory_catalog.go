package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/smallnest/goclaw/config"
	"github.com/smallnest/goclaw/governance"
	"github.com/smallnest/goclaw/memory/catalog"
	"github.com/spf13/cobra"
)

func init() {
	MemoryCmd.AddCommand(newMemoryCatalogCommand())
}

func newMemoryCatalogCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "catalog",
		Short: "Govern durable memory as a provenance-aware catalog",
		Long:  "Manage pending, approved, superseded, expired, and withdrawn project memories without requiring an embedding API.",
	}
	command.AddCommand(
		newCatalogStatusCommand(),
		newCatalogListCommand(),
		newCatalogSearchCommand(),
		newCatalogIngestCommand(),
		newCatalogDecisionCommand("approve"),
		newCatalogDecisionCommand("reject"),
		newCatalogDecisionCommand("withdraw"),
		newCatalogRenewCommand(),
		newCatalogAuthorityCommand(),
	)
	return command
}

func newCatalogStatusCommand() *cobra.Command {
	var projectID string
	command := &cobra.Command{
		Use:   "status",
		Short: "Show catalog lifecycle and authority statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			service, _, err := openCatalogService()
			if err != nil {
				return err
			}
			defer service.Close()
			stats, err := service.Stats(projectID)
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, stats)
		},
	}
	command.Flags().StringVar(&projectID, "project", "", "Project id")
	return command
}

func newCatalogListCommand() *cobra.Command {
	var projectID, status string
	var limit int
	command := &cobra.Command{
		Use:   "list",
		Short: "List catalog records",
		RunE: func(cmd *cobra.Command, args []string) error {
			service, _, err := openCatalogService()
			if err != nil {
				return err
			}
			defer service.Close()
			records, err := service.List(projectID, catalog.RecordStatus(status), limit)
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, records)
		},
	}
	command.Flags().StringVar(&projectID, "project", "", "Project id")
	command.Flags().StringVar(&status, "status", "", "Lifecycle status filter")
	command.Flags().IntVar(&limit, "limit", 200, "Maximum records")
	return command
}

func newCatalogSearchCommand() *cobra.Command {
	var projectID string
	var kinds []string
	var limit int
	var includeExpired bool
	command := &cobra.Command{
		Use:   "search <query>",
		Short: "Search approved catalog memory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, _, err := openCatalogService()
			if err != nil {
				return err
			}
			defer service.Close()
			typedKinds := make([]catalog.RecordKind, 0, len(kinds))
			for _, kind := range kinds {
				typedKinds = append(typedKinds, catalog.RecordKind(kind))
			}
			results, err := service.Search(catalog.SearchQuery{
				Query:          args[0],
				ProjectID:      projectID,
				Kinds:          typedKinds,
				IncludeShared:  true,
				IncludeExpired: includeExpired,
				Limit:          limit,
			})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, results)
		},
	}
	command.Flags().StringVar(&projectID, "project", "", "Project id")
	command.Flags().StringSliceVar(&kinds, "kind", nil, "Memory kind; repeat or comma-separate")
	command.Flags().IntVar(&limit, "limit", 20, "Maximum results")
	command.Flags().BoolVar(&includeExpired, "include-expired", false, "Include expired active records")
	return command
}

func newCatalogIngestCommand() *cobra.Command {
	var projectID, collection, sourceRoot, sourceScheme, sourceKind, sourceRevision, actor, kind string
	command := &cobra.Command{
		Use:   "ingest <file-or-directory>",
		Short: "Create pending candidates from a Markdown file or directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, _, err := openCatalogService()
			if err != nil {
				return err
			}
			defer service.Close()
			report, err := service.IngestPath(args[0], catalog.IngestOptions{
				ProjectID:      projectID,
				Collection:     collection,
				DefaultKind:    catalog.RecordKind(kind),
				SourceRoot:     sourceRoot,
				SourceScheme:   sourceScheme,
				SourceKind:     sourceKind,
				SourceRevision: sourceRevision,
				Actor:          actor,
			})
			if err != nil {
				return err
			}
			if err := writeCommandJSON(cmd, report); err != nil {
				return err
			}
			if report.Failed > 0 {
				return fmt.Errorf("%d catalog input(s) failed; inspect the report above", report.Failed)
			}
			return nil
		},
	}
	command.Flags().StringVar(&projectID, "project", "", "Project id")
	command.Flags().StringVar(&collection, "collection", "knowledge-markdown", "Catalog collection")
	command.Flags().StringVar(&sourceRoot, "source-root", "", "Stable root used to form cross-computer source URIs")
	command.Flags().StringVar(&sourceScheme, "source-scheme", "markdown", "Stable URI scheme, for example markdown or git+markdown")
	command.Flags().StringVar(&sourceKind, "source-kind", "", "Provenance kind; inferred from source scheme when empty")
	command.Flags().StringVar(&sourceRevision, "source-revision", "", "Immutable source revision; git+markdown resolves HEAD when empty")
	command.Flags().StringVar(&actor, "actor", "catalog-importer", "Candidate creator identity")
	command.Flags().StringVar(&kind, "kind", "", "Default kind when path and frontmatter do not specify one")
	return command
}

func newCatalogDecisionCommand(action string) *cobra.Command {
	flags := &catalogReviewFlags{}
	command := &cobra.Command{
		Use:   action + " <record-id>",
		Short: strings.ToUpper(action[:1]) + action[1:] + " a catalog record",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, cfg, err := openCatalogService()
			if err != nil {
				return err
			}
			defer service.Close()
			review, err := flags.review(cfg.Governance, governance.RoleMemoryApprove)
			if err != nil {
				return err
			}
			var record catalog.Record
			switch action {
			case "approve":
				record, err = service.ApproveCandidate(args[0], review)
			case "reject":
				record, err = service.RejectCandidate(args[0], review)
			case "withdraw":
				record, err = service.Withdraw(args[0], review)
			default:
				return fmt.Errorf("unsupported decision %q", action)
			}
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, record)
		},
	}
	flags.bind(command)
	return command
}

func newCatalogRenewCommand() *cobra.Command {
	flags := &catalogReviewFlags{}
	var days int
	command := &cobra.Command{
		Use:   "renew <record-id>",
		Short: "Confirm an active record and schedule its next review",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, cfg, err := openCatalogService()
			if err != nil {
				return err
			}
			defer service.Close()
			review, err := flags.review(cfg.Governance, governance.RoleMemoryApprove)
			if err != nil {
				return err
			}
			record, err := service.RenewReview(args[0], review, days)
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, record)
		},
	}
	command.Flags().IntVar(&days, "days", 0, "Days until next review; 0 uses configured default")
	flags.bind(command)
	return command
}

func newCatalogAuthorityCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "authority",
		Short: "Manage preferred names, aliases, and redirects",
	}
	command.AddCommand(
		newAuthorityListCommand(),
		newAuthorityResolveCommand(),
		newAuthorityUpsertCommand(),
		newAuthorityMergeCommand(),
	)
	return command
}

func newAuthorityListCommand() *cobra.Command {
	var projectID string
	var includeRedirected bool
	command := &cobra.Command{
		Use:   "list",
		Short: "List authority records",
		RunE: func(cmd *cobra.Command, args []string) error {
			service, _, err := openCatalogService()
			if err != nil {
				return err
			}
			defer service.Close()
			authorities, err := service.ListAuthorities(projectID, includeRedirected)
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, authorities)
		},
	}
	command.Flags().StringVar(&projectID, "project", "", "Project id")
	command.Flags().BoolVar(&includeRedirected, "include-redirected", false, "Include deprecated and redirected names")
	return command
}

func newAuthorityResolveCommand() *cobra.Command {
	var projectID string
	command := &cobra.Command{
		Use:   "resolve <label-or-alias>",
		Short: "Resolve a preferred label or alias",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, _, err := openCatalogService()
			if err != nil {
				return err
			}
			defer service.Close()
			authority, err := service.ResolveAuthority(projectID, args[0])
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, authority)
		},
	}
	command.Flags().StringVar(&projectID, "project", "", "Project id")
	return command
}

func newAuthorityUpsertCommand() *cobra.Command {
	flags := &catalogReviewFlags{}
	var id, projectID, authorityType, description, creator string
	var aliases []string
	command := &cobra.Command{
		Use:   "upsert <preferred-label>",
		Short: "Create or update an authority record",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, cfg, err := openCatalogService()
			if err != nil {
				return err
			}
			defer service.Close()
			review, err := flags.review(cfg.Governance, governance.RoleAuthorityManage)
			if err != nil {
				return err
			}
			authority, err := service.UpsertAuthority(catalog.AuthorityInput{
				ID:             id,
				ProjectID:      projectID,
				Type:           catalog.AuthorityType(authorityType),
				PreferredLabel: args[0],
				Aliases:        aliases,
				Description:    description,
				CreatedBy:      creator,
			}, review)
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, authority)
		},
	}
	command.Flags().StringVar(&id, "id", "", "Existing authority id to update")
	command.Flags().StringVar(&projectID, "project", "", "Project id")
	command.Flags().StringVar(&authorityType, "type", "topic", "person, organization, project, system, topic, place, or device")
	command.Flags().StringSliceVar(&aliases, "alias", nil, "Variant name; repeat or comma-separate")
	command.Flags().StringVar(&description, "description", "", "Disambiguating description")
	command.Flags().StringVar(&creator, "creator", "catalog-importer", "Authority proposal creator")
	flags.bind(command)
	return command
}

func newAuthorityMergeCommand() *cobra.Command {
	flags := &catalogReviewFlags{}
	command := &cobra.Command{
		Use:   "merge <source-id> <target-id>",
		Short: "Redirect a duplicate authority to the canonical authority",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, cfg, err := openCatalogService()
			if err != nil {
				return err
			}
			defer service.Close()
			review, err := flags.review(cfg.Governance, governance.RoleAuthorityManage)
			if err != nil {
				return err
			}
			authority, err := service.MergeAuthority(args[0], args[1], review)
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, authority)
		},
	}
	flags.bind(command)
	return command
}

type catalogReviewFlags struct {
	reviewer        string
	rationale       string
	counterargument string
	evidence        []string
}

func (f *catalogReviewFlags) bind(command *cobra.Command) {
	command.Flags().StringVar(&f.reviewer, "reviewer", "", "Registered human reviewer id")
	command.Flags().StringVar(&f.rationale, "rationale", "", "Evidence-based decision rationale")
	command.Flags().StringVar(&f.counterargument, "counterargument", "", "Strongest reason this decision could be wrong")
	command.Flags().StringSliceVar(&f.evidence, "evidence-ref", nil, "Evidence reference; repeat or comma-separate")
}

func (f *catalogReviewFlags) review(policy governance.Config, role string) (governance.Review, error) {
	if strings.TrimSpace(f.reviewer) == "" {
		return governance.Review{}, fmt.Errorf("--reviewer is required")
	}
	review, err := governance.ResolveReviewer(policy, governance.Credential{
		ReviewerID: f.reviewer,
		Token:      os.Getenv("GOCLAW_REVIEWER_TOKEN"),
		Source:     "local-cli",
	}, role)
	if err != nil {
		return governance.Review{}, err
	}
	review.Rationale = strings.TrimSpace(f.rationale)
	review.Counterargument = strings.TrimSpace(f.counterargument)
	review.EvidenceRefs = append([]string(nil), f.evidence...)
	return review, nil
}

func openCatalogService() (*catalog.Service, *config.Config, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, nil, err
	}
	service, err := catalog.NewService(cfg.Memory.Catalog)
	if err != nil {
		return nil, nil, err
	}
	service.SetGovernancePolicy(cfg.Governance)
	return service, cfg, nil
}

func writeCommandJSON(command *cobra.Command, value interface{}) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(command.OutOrStdout(), string(data))
	return err
}
