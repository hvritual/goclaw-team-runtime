package teamcontrol

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func executeDelivery(
	t *testing.T,
	service *Service,
	actorID string,
	revision uint64,
	commandID string,
	commandType DeliveryCommandType,
	payload any,
) DeliveryCommandResult {
	t.Helper()
	data, err := json.Marshal(payload)
	require.NoError(t, err)
	result, err := service.ExecuteDeliveryCommand(actorID, DeliveryCommand{
		ID: commandID, ProjectID: "project-a", Type: commandType,
		ExpectedRevision: revision, Payload: data,
	})
	require.NoError(t, err)
	require.Equal(t, revision+1, result.Receipt.ProjectRevision)
	require.Len(t, result.Events, 1)
	return result
}

func TestDeliveryRequirementDefectAndRiskClosedLoopsReplay(t *testing.T) {
	fixture := newTestFixture(t)
	revision := uint64(0)

	requestResult := executeDelivery(t, fixture.service, fixture.bob.ID, revision,
		"cmd-request", CommandCreateRequest, CreateRequestPayload{
			ID: "req-online", Title: "Raise device online rate",
			Description:        "Increase trustworthy device online rate without weakening the offline rule.",
			AcceptanceCriteria: []string{"daily online rate is measured", "rollback is proven"},
			NonGoals:           []string{"no device control"}, Constraints: []string{"single writer"},
		})
	revision = requestResult.Receipt.ProjectRevision

	intentResult := executeDelivery(t, fixture.service, fixture.alice.ID, revision,
		"cmd-intent", CommandApproveIntent, ApproveIntentPayload{
			ID: "intent-online", RequestID: "req-online",
			Goal:  "Raise measured online rate while preserving data integrity.",
			Users: []string{"IoT operations"}, Scenarios: []string{"weak network recovery"},
			Scope:              []string{"gateway", "device telemetry"},
			NonGoals:           []string{"remote device control"},
			Constraints:        []string{"no mock provider", "single writer"},
			AcceptanceCriteria: []string{"daily online rate is measured", "rollback is proven"},
			RiskBoundary:       "No silent state repair.", CostBoundary: "Stay within project budget.",
		})
	revision = intentResult.Receipt.ProjectRevision

	solutionResult := executeDelivery(t, fixture.service, fixture.bob.ID, revision,
		"cmd-solution", CommandCreateSolution, CreateSolutionPayload{
			ID: "solution-online", RequestID: "req-online", IntentID: "intent-online",
			Title:   "Event-backed online-rate delivery",
			Summary: "Use deterministic projections and evidence-bound work items.",
			ADRRefs: []string{"adr-001"}, AllowedPaths: []string{"teamcontrol/**"},
			ForbiddenPaths: []string{"workstation/**"},
			TestStrategy:   []string{"unit replay", "tamper rejection"},
			RollbackPlan:   "Revert the Stage One branch without changing accepted TC-W01 state.",
		})
	revision = solutionResult.Receipt.ProjectRevision

	for index, kind := range []DeliveryReviewKind{
		DeliveryReviewScenario,
		DeliveryReviewCapacity,
		DeliveryReviewRisk,
		DeliveryReviewCost,
	} {
		result := executeDelivery(t, fixture.service, fixture.alice.ID, revision,
			"cmd-review-"+string(kind), CommandDecideSolutionReview,
			DecideSolutionReviewPayload{
				SolutionID: "solution-online", Kind: kind,
				Decision: DeliveryReviewApproved, Comment: "approved",
			})
		revision = result.Receipt.ProjectRevision
		require.Equal(t, index+4, int(revision))
	}

	freezeResult := executeDelivery(t, fixture.service, fixture.alice.ID, revision,
		"cmd-freeze", CommandFreezePlan, FreezePlanPayload{
			ID: "plan-online", SolutionID: "solution-online",
			WorkItems: []FrozenWorkItem{{
				ID: "work-online", Title: "Implement deterministic delivery journal",
				Instructions:   "Add append-only events, replay, integrity checks, and evidence.",
				BusinessDomain: "platform", Priority: PriorityP1, EstimatePoints: 8,
				VerificationCommands: [][]string{{"go", "test", "./teamcontrol"}},
				EvidenceRequirements: []string{"unit test report", "integrity report"},
				RiskLevel:            SeverityHigh,
			}},
		})
	revision = freezeResult.Receipt.ProjectRevision

	projection, err := fixture.service.GetDeliveryProjection(fixture.mallory.ID, fixture.projectA.ID)
	require.NoError(t, err)
	require.Equal(t, RequestFrozen, projection.Requests["req-online"].Status)
	require.Equal(t, SolutionFrozen, projection.Solutions["solution-online"].Status)
	require.Equal(t, []string{"work-online"}, projection.FrozenPlans["plan-online"].WorkItemIDs)
	work, err := fixture.service.GetWorkItem(fixture.mallory.ID, fixture.projectA.ID, "work-online")
	require.NoError(t, err)
	require.Equal(t, ResourceRequest, work.SourceType)
	require.Equal(t, "intent-online", work.ContractID)
	require.NotEmpty(t, work.EvidenceRequirements)

	changeResult := executeDelivery(t, fixture.service, fixture.bob.ID, revision,
		"cmd-change", CommandCreateChangeIntent, CreateChangeIntentPayload{
			ID: "change-online", RequestID: "req-online", FrozenPlanID: "plan-online",
			Reason:          "Add a newly discovered recovery constraint.",
			Impact:          "Requires a new contract revision; frozen acceptance stays immutable.",
			AcceptanceDelta: []string{"recovery is idempotent"},
		})
	revision = changeResult.Receipt.ProjectRevision

	defectResult := executeDelivery(t, fixture.service, fixture.bob.ID, revision,
		"cmd-defect", CommandCreateDefect, CreateDefectPayload{
			ID: "defect-reconnect", Title: "Reconnect drops telemetry",
			Description: "A reconnect can lose the first telemetry frame.",
			Severity:    SeverityHigh, Priority: PriorityP1,
			Environment: "field", Module: "connectivity", OwnerID: fixture.bob.ID,
		})
	revision = defectResult.Receipt.ProjectRevision
	defectTransitions := []TransitionDefectPayload{
		{DefectID: "defect-reconnect", Status: DefectConfirmed},
		{DefectID: "defect-reconnect", Status: DefectReproduced,
			Reproduction: "disconnect and reconnect", Expected: "first frame retained", Actual: "first frame lost"},
		{DefectID: "defect-reconnect", Status: DefectClassified},
		{DefectID: "defect-reconnect", Status: DefectFixing, WorkItemIDs: []string{"work-online"}},
		{DefectID: "defect-reconnect", Status: DefectVerifying,
			RootCause:  "subscription installed after socket readiness",
			Resolution: "install subscription before readiness"},
	}
	for index, transition := range defectTransitions {
		result := executeDelivery(t, fixture.service, fixture.bob.ID, revision,
			"cmd-defect-transition-"+string(rune('a'+index)),
			CommandTransitionDefect, transition)
		revision = result.Receipt.ProjectRevision
	}
	evidenceResult := executeDelivery(t, fixture.service, fixture.bob.ID, revision,
		"cmd-defect-evidence", CommandRecordEvidence, RecordDeliveryEvidencePayload{
			ID: "evidence-defect", ResourceType: ResourceDefect,
			ResourceID: "defect-reconnect", Kind: "regression",
			URI:    "https://example.invalid/evidence/defect.json",
			SHA256: strings.Repeat("a", 64), Summary: "Regression suite passed.",
		})
	revision = evidenceResult.Receipt.ProjectRevision
	for index, transition := range []TransitionDefectPayload{
		{DefectID: "defect-reconnect", Status: DefectVerified, EvidenceIDs: []string{"evidence-defect"}},
		{DefectID: "defect-reconnect", Status: DefectReleased},
		{DefectID: "defect-reconnect", Status: DefectClosed},
	} {
		result := executeDelivery(t, fixture.service, fixture.bob.ID, revision,
			"cmd-defect-close-"+string(rune('a'+index)),
			CommandTransitionDefect, transition)
		revision = result.Receipt.ProjectRevision
	}

	riskResult := executeDelivery(t, fixture.service, fixture.bob.ID, revision,
		"cmd-risk", CommandCreateRisk, CreateRiskPayload{
			ID: "risk-backlog", Title: "Recovery backlog overloads Runner",
			Description: "A reconnect storm can exceed the recovery capacity.",
			Probability: "medium", Impact: "high", Trigger: "queue depth exceeds 100",
			OwnerID: fixture.alice.ID,
		})
	revision = riskResult.Receipt.ProjectRevision
	assessedResult := executeDelivery(t, fixture.service, fixture.alice.ID, revision,
		"cmd-risk-assess", CommandTransitionRisk, TransitionRiskPayload{
			RiskID: "risk-backlog", Status: RiskAssessed,
		})
	revision = assessedResult.Receipt.ProjectRevision
	reviewAt := time.Now().UTC().Add(24 * time.Hour)
	responseResult := executeDelivery(t, fixture.service, fixture.alice.ID, revision,
		"cmd-risk-response", CommandDecideRiskResponse, DecideRiskResponsePayload{
			RiskID: "risk-backlog", Response: RiskAccept,
			AcceptanceReason: "Pilot traffic is bounded and observable.",
			ReviewAt:         &reviewAt,
		})
	revision = responseResult.Receipt.ProjectRevision
	riskEvidenceResult := executeDelivery(t, fixture.service, fixture.bob.ID, revision,
		"cmd-risk-evidence", CommandRecordEvidence, RecordDeliveryEvidencePayload{
			ID: "evidence-risk", ResourceType: ResourceRisk,
			ResourceID: "risk-backlog", Kind: "capacity",
			URI:    "https://example.invalid/evidence/risk.json",
			SHA256: strings.Repeat("b", 64), Summary: "Queue capacity observed.",
		})
	revision = riskEvidenceResult.Receipt.ProjectRevision
	reviewedResult := executeDelivery(t, fixture.service, fixture.alice.ID, revision,
		"cmd-risk-reviewed", CommandTransitionRisk, TransitionRiskPayload{
			RiskID: "risk-backlog", Status: RiskReviewed,
			EvidenceIDs: []string{"evidence-risk"},
		})
	revision = reviewedResult.Receipt.ProjectRevision
	closedResult := executeDelivery(t, fixture.service, fixture.alice.ID, revision,
		"cmd-risk-closed", CommandTransitionRisk, TransitionRiskPayload{
			RiskID: "risk-backlog", Status: RiskClosed,
		})
	revision = closedResult.Receipt.ProjectRevision

	report, err := fixture.service.VerifyDeliveryIntegrity(fixture.mallory.ID, fixture.projectA.ID)
	require.NoError(t, err)
	require.True(t, report.ProjectionStable)
	require.Equal(t, int(revision), report.EventCount)

	before, err := fixture.service.GetDeliveryProjection(fixture.alice.ID, fixture.projectA.ID)
	require.NoError(t, err)
	reopened, err := Open(filepath.Dir(fixture.service.store.path))
	require.NoError(t, err)
	after, err := reopened.GetDeliveryProjection(fixture.alice.ID, fixture.projectA.ID)
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.Equal(t, DefectClosed, after.Defects["defect-reconnect"].Status)
	require.Equal(t, RiskClosed, after.Risks["risk-backlog"].Status)
}

func TestDeliveryCommandsAreIdempotentCASBoundAndFailClosed(t *testing.T) {
	fixture := newTestFixture(t)
	payload := CreateRequestPayload{
		ID: "req-idempotent", Title: "Idempotent request",
		Description:        "The same command must never append twice.",
		AcceptanceCriteria: []string{"only one event is appended"},
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)
	command := DeliveryCommand{
		ID: "cmd-idempotent", ProjectID: fixture.projectA.ID,
		Type: CommandCreateRequest, ExpectedRevision: 0, Payload: data,
	}
	first, err := fixture.service.ExecuteDeliveryCommand(fixture.bob.ID, command)
	require.NoError(t, err)
	second, err := fixture.service.ExecuteDeliveryCommand(fixture.bob.ID, command)
	require.NoError(t, err)
	require.Equal(t, first, second)

	changed, err := json.Marshal(CreateRequestPayload{
		ID: "req-other", Title: "Changed payload", Description: "collision",
	})
	require.NoError(t, err)
	command.Payload = changed
	_, err = fixture.service.ExecuteDeliveryCommand(fixture.bob.ID, command)
	require.ErrorIs(t, err, ErrConflict)

	_, err = fixture.service.ExecuteDeliveryCommand(fixture.bob.ID, DeliveryCommand{
		ID: "cmd-stale", ProjectID: fixture.projectA.ID,
		Type: CommandCreateRequest, ExpectedRevision: 0, Payload: changed,
	})
	require.ErrorIs(t, err, ErrConflict)

	_, err = fixture.service.ExecuteDeliveryCommand(fixture.mallory.ID, DeliveryCommand{
		ID: "cmd-viewer", ProjectID: fixture.projectA.ID,
		Type: CommandCreateRequest, ExpectedRevision: 1, Payload: changed,
	})
	require.ErrorIs(t, err, ErrForbidden)

	charlie, err := fixture.service.CreateUser(CreateUserInput{
		ID: "charlie", DisplayName: "Charlie", Email: "charlie@example.com",
	})
	require.NoError(t, err)
	_, err = fixture.service.AddTeamMember(
		fixture.alice.ID,
		fixture.team.ID,
		AddTeamMemberInput{UserID: charlie.ID, Role: TeamRegularMember},
	)
	require.NoError(t, err)
	for _, testCase := range []struct {
		name        string
		commandType DeliveryCommandType
		payload     any
	}{
		{
			name: "defect owner outside project", commandType: CommandCreateDefect,
			payload: CreateDefectPayload{
				ID: "defect-invalid-owner", Title: "Invalid owner",
				Description: "Owners must be active project members.",
				Severity:    SeverityMedium, Priority: PriorityP2, OwnerID: charlie.ID,
			},
		},
		{
			name: "risk owner outside project", commandType: CommandCreateRisk,
			payload: CreateRiskPayload{
				ID: "risk-invalid-owner", Title: "Invalid owner",
				Description: "Owners must be active project members.",
				Probability: "low", Impact: "high", Trigger: "owner assignment",
				OwnerID: charlie.ID,
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			payload, marshalErr := json.Marshal(testCase.payload)
			require.NoError(t, marshalErr)
			_, commandErr := fixture.service.ExecuteDeliveryCommand(
				fixture.bob.ID,
				DeliveryCommand{
					ID:        "cmd-" + strings.ReplaceAll(testCase.name, " ", "-"),
					ProjectID: fixture.projectA.ID, Type: testCase.commandType,
					ExpectedRevision: 1, Payload: payload,
				},
			)
			require.ErrorIs(t, commandErr, ErrForbidden)
		})
	}

	require.NoError(t, fixture.service.store.update(func(st *state) error {
		st.Delivery.Events[0].Hash = strings.Repeat("0", 64)
		return nil
	}))
	_, err = Open(filepath.Dir(fixture.service.store.path))
	require.ErrorContains(t, err, "hash mismatch")
}

func TestDeliveryIntegrityIsProjectScopedAndRiskReviewIsIndependent(t *testing.T) {
	fixture := newTestFixture(t)
	executeDelivery(t, fixture.service, fixture.bob.ID, 0,
		"cmd-project-a", CommandCreateRequest, CreateRequestPayload{
			ID: "req-project-a", Title: "Project A request", Description: "Scoped to project A.",
		})

	projectBPayload, err := json.Marshal(CreateRequestPayload{
		ID: "req-project-b", Title: "Project B request", Description: "Scoped to project B.",
	})
	require.NoError(t, err)
	_, err = fixture.service.ExecuteDeliveryCommand(fixture.alice.ID, DeliveryCommand{
		ID: "cmd-project-b", ProjectID: fixture.projectB.ID,
		Type: CommandCreateRequest, ExpectedRevision: 0, Payload: projectBPayload,
	})
	require.NoError(t, err)

	report, err := fixture.service.VerifyDeliveryIntegrity(fixture.mallory.ID, fixture.projectA.ID)
	require.NoError(t, err)
	require.Equal(t, 1, report.EventCount)
	require.Equal(t, 1, report.ProjectCount)
	require.Equal(t, uint64(1), report.LastSequence)

	riskResult := executeDelivery(t, fixture.service, fixture.alice.ID, 1,
		"cmd-self-risk", CommandCreateRisk, CreateRiskPayload{
			ID: "risk-self-review", Title: "Independent risk response",
			Description: "The creator must not approve their own response.",
			Probability: "low", Impact: "high", Trigger: "creator attempts self-review",
			OwnerID: fixture.alice.ID,
		})
	assessed := executeDelivery(t, fixture.service, fixture.alice.ID,
		riskResult.Receipt.ProjectRevision, "cmd-self-risk-assess",
		CommandTransitionRisk, TransitionRiskPayload{
			RiskID: "risk-self-review", Status: RiskAssessed,
		})
	reviewAt := time.Now().UTC().Add(time.Hour)
	responsePayload, err := json.Marshal(DecideRiskResponsePayload{
		RiskID: "risk-self-review", Response: RiskAccept,
		AcceptanceReason: "Bounded pilot.", ReviewAt: &reviewAt,
	})
	require.NoError(t, err)
	_, err = fixture.service.ExecuteDeliveryCommand(fixture.alice.ID, DeliveryCommand{
		ID: "cmd-self-risk-response", ProjectID: fixture.projectA.ID,
		Type:             CommandDecideRiskResponse,
		ExpectedRevision: assessed.Receipt.ProjectRevision, Payload: responsePayload,
	})
	require.ErrorIs(t, err, ErrForbidden)
}
