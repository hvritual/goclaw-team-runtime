package teamcontrol

import (
	"encoding/json"
	"time"
)

const DeliveryEventSchemaVersion = 1

type DeliveryCommandType string

const (
	CommandCreateRequest        DeliveryCommandType = "request.create"
	CommandApproveIntent        DeliveryCommandType = "intent.approve"
	CommandCreateSolution       DeliveryCommandType = "solution.create"
	CommandDecideSolutionReview DeliveryCommandType = "solution.review.decide"
	CommandFreezePlan           DeliveryCommandType = "plan.freeze"
	CommandCreateChangeIntent   DeliveryCommandType = "change_intent.create"
	CommandCreateDefect         DeliveryCommandType = "defect.create"
	CommandTransitionDefect     DeliveryCommandType = "defect.transition"
	CommandCreateRisk           DeliveryCommandType = "risk.create"
	CommandDecideRiskResponse   DeliveryCommandType = "risk.response.decide"
	CommandTransitionRisk       DeliveryCommandType = "risk.transition"
	CommandRecordEvidence       DeliveryCommandType = "delivery.evidence.record"
)

type DeliveryEventType string

const (
	EventRequestCreated           DeliveryEventType = "request.created"
	EventIntentApproved           DeliveryEventType = "intent.approved"
	EventSolutionCreated          DeliveryEventType = "solution.created"
	EventSolutionReviewDecided    DeliveryEventType = "solution.review.decided"
	EventPlanFrozen               DeliveryEventType = "plan.frozen"
	EventChangeIntentCreated      DeliveryEventType = "change_intent.created"
	EventDefectCreated            DeliveryEventType = "defect.created"
	EventDefectTransitioned       DeliveryEventType = "defect.transitioned"
	EventRiskCreated              DeliveryEventType = "risk.created"
	EventRiskResponseDecided      DeliveryEventType = "risk.response.decided"
	EventRiskTransitioned         DeliveryEventType = "risk.transitioned"
	EventDeliveryEvidenceRecorded DeliveryEventType = "delivery.evidence.recorded"
)

type DeliveryCommand struct {
	ID               string              `json:"id"`
	ProjectID        string              `json:"project_id"`
	Type             DeliveryCommandType `json:"type"`
	ActorID          string              `json:"actor_id"`
	ExpectedRevision uint64              `json:"expected_revision"`
	Payload          json.RawMessage     `json:"payload"`
}

type DeliveryCommandReceipt struct {
	CommandID       string              `json:"command_id"`
	ProjectID       string              `json:"project_id"`
	CommandType     DeliveryCommandType `json:"command_type"`
	CommandHash     string              `json:"command_hash"`
	EventIDs        []string            `json:"event_ids"`
	ProjectRevision uint64              `json:"project_revision"`
	RecordedAt      time.Time           `json:"recorded_at"`
}

type DeliveryCommandResult struct {
	Receipt DeliveryCommandReceipt `json:"receipt"`
	Events  []DeliveryEvent        `json:"events"`
}

type DeliveryState struct {
	Events   []DeliveryEvent                   `json:"events"`
	Commands map[string]DeliveryCommandReceipt `json:"commands"`
	Projects map[string]DeliveryProjection     `json:"projects"`
	LastHash string                            `json:"last_hash,omitempty"`
}

func newDeliveryState() DeliveryState {
	return DeliveryState{
		Events:   make([]DeliveryEvent, 0),
		Commands: make(map[string]DeliveryCommandReceipt),
		Projects: make(map[string]DeliveryProjection),
	}
}

func normalizeDeliveryState(value *DeliveryState) {
	if value.Events == nil {
		value.Events = make([]DeliveryEvent, 0)
	}
	if value.Commands == nil {
		value.Commands = make(map[string]DeliveryCommandReceipt)
	}
	if value.Projects == nil {
		value.Projects = make(map[string]DeliveryProjection)
	}
	for projectID, projection := range value.Projects {
		normalizeDeliveryProjection(&projection, projectID)
		value.Projects[projectID] = projection
	}
}

func normalizeDeliveryProjection(value *DeliveryProjection, projectID string) {
	if value.ProjectID == "" {
		value.ProjectID = projectID
	}
	if value.Requests == nil {
		value.Requests = make(map[string]DeliveryRequest)
	}
	if value.Intents == nil {
		value.Intents = make(map[string]IntentContract)
	}
	if value.Solutions == nil {
		value.Solutions = make(map[string]SolutionSpec)
	}
	if value.FrozenPlans == nil {
		value.FrozenPlans = make(map[string]FrozenPlan)
	}
	if value.ChangeIntents == nil {
		value.ChangeIntents = make(map[string]ChangeIntent)
	}
	if value.Defects == nil {
		value.Defects = make(map[string]Defect)
	}
	if value.Risks == nil {
		value.Risks = make(map[string]Risk)
	}
	if value.Evidence == nil {
		value.Evidence = make(map[string]DeliveryEvidence)
	}
}

type DeliveryEvent struct {
	ID            string            `json:"id"`
	ProjectID     string            `json:"project_id"`
	StreamID      string            `json:"stream_id"`
	StreamVersion uint64            `json:"stream_version"`
	Sequence      uint64            `json:"sequence"`
	SchemaVersion int               `json:"schema_version"`
	Type          DeliveryEventType `json:"event_type"`
	ActorID       string            `json:"actor_id"`
	CommandID     string            `json:"command_id"`
	CommandHash   string            `json:"command_hash"`
	Payload       json.RawMessage   `json:"payload"`
	OccurredAt    time.Time         `json:"occurred_at"`
	PreviousHash  string            `json:"previous_hash,omitempty"`
	Hash          string            `json:"hash"`
}

type RequestStatus string

const (
	RequestDraft          RequestStatus = "draft"
	RequestIntentApproved RequestStatus = "intent_approved"
	RequestReviewPending  RequestStatus = "review_pending"
	RequestFrozen         RequestStatus = "frozen"
	RequestChangePending  RequestStatus = "change_pending"
)

type DeliveryRequest struct {
	ID                 string        `json:"id"`
	ProjectID          string        `json:"project_id"`
	Title              string        `json:"title"`
	Description        string        `json:"description"`
	Source             string        `json:"source,omitempty"`
	Status             RequestStatus `json:"status"`
	Revision           uint64        `json:"revision"`
	AcceptanceCriteria []string      `json:"acceptance_criteria,omitempty"`
	NonGoals           []string      `json:"non_goals,omitempty"`
	Constraints        []string      `json:"constraints,omitempty"`
	CreatedBy          string        `json:"created_by"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}

type IntentContract struct {
	ID                 string    `json:"id"`
	ProjectID          string    `json:"project_id"`
	RequestID          string    `json:"request_id"`
	Goal               string    `json:"goal"`
	Users              []string  `json:"users,omitempty"`
	Scenarios          []string  `json:"scenarios,omitempty"`
	Scope              []string  `json:"scope,omitempty"`
	NonGoals           []string  `json:"non_goals,omitempty"`
	Constraints        []string  `json:"constraints,omitempty"`
	AcceptanceCriteria []string  `json:"acceptance_criteria"`
	RiskBoundary       string    `json:"risk_boundary,omitempty"`
	CostBoundary       string    `json:"cost_boundary,omitempty"`
	Revision           uint64    `json:"revision"`
	ApprovedBy         string    `json:"approved_by"`
	ApprovedAt         time.Time `json:"approved_at"`
}

type SolutionStatus string

const (
	SolutionProposed SolutionStatus = "proposed"
	SolutionApproved SolutionStatus = "approved"
	SolutionFrozen   SolutionStatus = "frozen"
)

type DeliveryReviewKind string

const (
	DeliveryReviewScenario DeliveryReviewKind = "scenario"
	DeliveryReviewCapacity DeliveryReviewKind = "capacity"
	DeliveryReviewRisk     DeliveryReviewKind = "risk"
	DeliveryReviewCost     DeliveryReviewKind = "cost"
)

type DeliveryReviewDecision string

const (
	DeliveryReviewPending  DeliveryReviewDecision = "pending"
	DeliveryReviewApproved DeliveryReviewDecision = "approved"
	DeliveryReviewRejected DeliveryReviewDecision = "rejected"
)

type DeliveryReview struct {
	Kind      DeliveryReviewKind     `json:"kind"`
	Decision  DeliveryReviewDecision `json:"decision"`
	Reviewer  string                 `json:"reviewer,omitempty"`
	Comment   string                 `json:"comment,omitempty"`
	DecidedAt *time.Time             `json:"decided_at,omitempty"`
}

type SolutionSpec struct {
	ID             string                                `json:"id"`
	ProjectID      string                                `json:"project_id"`
	RequestID      string                                `json:"request_id"`
	IntentID       string                                `json:"intent_id"`
	Title          string                                `json:"title"`
	Summary        string                                `json:"summary"`
	ADRRefs        []string                              `json:"adr_refs,omitempty"`
	AllowedPaths   []string                              `json:"allowed_paths,omitempty"`
	ForbiddenPaths []string                              `json:"forbidden_paths,omitempty"`
	TestStrategy   []string                              `json:"test_strategy"`
	RollbackPlan   string                                `json:"rollback_plan"`
	Status         SolutionStatus                        `json:"status"`
	Revision       uint64                                `json:"revision"`
	Reviews        map[DeliveryReviewKind]DeliveryReview `json:"reviews"`
	CreatedBy      string                                `json:"created_by"`
	CreatedAt      time.Time                             `json:"created_at"`
	UpdatedAt      time.Time                             `json:"updated_at"`
}

type FrozenPlan struct {
	ID             string    `json:"id"`
	ProjectID      string    `json:"project_id"`
	RequestID      string    `json:"request_id"`
	IntentID       string    `json:"intent_id"`
	SolutionID     string    `json:"solution_id"`
	BundleRevision uint64    `json:"bundle_revision"`
	BundleHash     string    `json:"bundle_hash"`
	WorkItemIDs    []string  `json:"work_item_ids"`
	FrozenBy       string    `json:"frozen_by"`
	FrozenAt       time.Time `json:"frozen_at"`
}

type ChangeIntentStatus string

const (
	ChangeIntentPending  ChangeIntentStatus = "pending"
	ChangeIntentApproved ChangeIntentStatus = "approved"
	ChangeIntentRejected ChangeIntentStatus = "rejected"
)

type ChangeIntent struct {
	ID              string             `json:"id"`
	ProjectID       string             `json:"project_id"`
	RequestID       string             `json:"request_id"`
	FrozenPlanID    string             `json:"frozen_plan_id"`
	Reason          string             `json:"reason"`
	Impact          string             `json:"impact"`
	ProposedGoal    string             `json:"proposed_goal,omitempty"`
	AcceptanceDelta []string           `json:"acceptance_delta,omitempty"`
	Status          ChangeIntentStatus `json:"status"`
	CreatedBy       string             `json:"created_by"`
	CreatedAt       time.Time          `json:"created_at"`
}

type DefectStatus string

const (
	DefectReported   DefectStatus = "reported"
	DefectConfirmed  DefectStatus = "confirmed"
	DefectReproduced DefectStatus = "reproduced"
	DefectClassified DefectStatus = "classified"
	DefectFixing     DefectStatus = "fixing"
	DefectVerifying  DefectStatus = "verifying"
	DefectVerified   DefectStatus = "verified"
	DefectReleased   DefectStatus = "released"
	DefectClosed     DefectStatus = "closed"
	DefectRejected   DefectStatus = "rejected"
	DefectReopened   DefectStatus = "reopened"
)

type Defect struct {
	ID            string        `json:"id"`
	ProjectID     string        `json:"project_id"`
	Title         string        `json:"title"`
	Description   string        `json:"description"`
	Status        DefectStatus  `json:"status"`
	Severity      IssueSeverity `json:"severity"`
	Priority      IssuePriority `json:"priority"`
	Environment   string        `json:"environment,omitempty"`
	Module        string        `json:"module,omitempty"`
	AffectedScope string        `json:"affected_scope,omitempty"`
	Reproduction  string        `json:"reproduction,omitempty"`
	Expected      string        `json:"expected,omitempty"`
	Actual        string        `json:"actual,omitempty"`
	Containment   string        `json:"containment,omitempty"`
	RootCause     string        `json:"root_cause,omitempty"`
	Resolution    string        `json:"resolution,omitempty"`
	WorkItemIDs   []string      `json:"work_item_ids,omitempty"`
	EvidenceIDs   []string      `json:"evidence_ids,omitempty"`
	ReporterID    string        `json:"reporter_id"`
	OwnerID       string        `json:"owner_id,omitempty"`
	Revision      uint64        `json:"revision"`
	ReopenCount   int           `json:"reopen_count"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

type RiskStatus string

const (
	RiskIdentified RiskStatus = "identified"
	RiskAssessed   RiskStatus = "assessed"
	RiskMonitoring RiskStatus = "monitoring"
	RiskMitigating RiskStatus = "mitigating"
	RiskReviewed   RiskStatus = "reviewed"
	RiskClosed     RiskStatus = "closed"
)

type RiskResponse string

const (
	RiskAvoid    RiskResponse = "avoid"
	RiskMitigate RiskResponse = "mitigate"
	RiskTransfer RiskResponse = "transfer"
	RiskAccept   RiskResponse = "accept"
	RiskMonitor  RiskResponse = "monitor"
)

type Risk struct {
	ID               string       `json:"id"`
	ProjectID        string       `json:"project_id"`
	Title            string       `json:"title"`
	Description      string       `json:"description"`
	Status           RiskStatus   `json:"status"`
	Probability      string       `json:"probability"`
	Impact           string       `json:"impact"`
	Trigger          string       `json:"trigger"`
	Response         RiskResponse `json:"response,omitempty"`
	ResponsePlan     string       `json:"response_plan,omitempty"`
	AcceptanceReason string       `json:"acceptance_reason,omitempty"`
	OwnerID          string       `json:"owner_id"`
	ReviewAt         *time.Time   `json:"review_at,omitempty"`
	WorkItemIDs      []string     `json:"work_item_ids,omitempty"`
	EvidenceIDs      []string     `json:"evidence_ids,omitempty"`
	Revision         uint64       `json:"revision"`
	CreatedBy        string       `json:"created_by"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

type DeliveryEvidence struct {
	ID           string       `json:"id"`
	ProjectID    string       `json:"project_id"`
	ResourceType ResourceType `json:"resource_type"`
	ResourceID   string       `json:"resource_id"`
	Kind         string       `json:"kind"`
	URI          string       `json:"uri"`
	SHA256       string       `json:"sha256,omitempty"`
	Summary      string       `json:"summary,omitempty"`
	RecordedBy   string       `json:"recorded_by"`
	RecordedAt   time.Time    `json:"recorded_at"`
}

type DeliveryProjection struct {
	ProjectID     string                      `json:"project_id"`
	Revision      uint64                      `json:"revision"`
	Requests      map[string]DeliveryRequest  `json:"requests"`
	Intents       map[string]IntentContract   `json:"intents"`
	Solutions     map[string]SolutionSpec     `json:"solutions"`
	FrozenPlans   map[string]FrozenPlan       `json:"frozen_plans"`
	ChangeIntents map[string]ChangeIntent     `json:"change_intents"`
	Defects       map[string]Defect           `json:"defects"`
	Risks         map[string]Risk             `json:"risks"`
	Evidence      map[string]DeliveryEvidence `json:"evidence"`
	UpdatedAt     time.Time                   `json:"updated_at"`
}

type CreateRequestPayload struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Source             string   `json:"source,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	NonGoals           []string `json:"non_goals,omitempty"`
	Constraints        []string `json:"constraints,omitempty"`
}

type ApproveIntentPayload struct {
	ID                 string   `json:"id"`
	RequestID          string   `json:"request_id"`
	Goal               string   `json:"goal"`
	Users              []string `json:"users,omitempty"`
	Scenarios          []string `json:"scenarios,omitempty"`
	Scope              []string `json:"scope,omitempty"`
	NonGoals           []string `json:"non_goals,omitempty"`
	Constraints        []string `json:"constraints,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	RiskBoundary       string   `json:"risk_boundary,omitempty"`
	CostBoundary       string   `json:"cost_boundary,omitempty"`
}

type CreateSolutionPayload struct {
	ID             string   `json:"id"`
	RequestID      string   `json:"request_id"`
	IntentID       string   `json:"intent_id"`
	Title          string   `json:"title"`
	Summary        string   `json:"summary"`
	ADRRefs        []string `json:"adr_refs,omitempty"`
	AllowedPaths   []string `json:"allowed_paths,omitempty"`
	ForbiddenPaths []string `json:"forbidden_paths,omitempty"`
	TestStrategy   []string `json:"test_strategy"`
	RollbackPlan   string   `json:"rollback_plan"`
}

type DecideSolutionReviewPayload struct {
	SolutionID string                 `json:"solution_id"`
	Kind       DeliveryReviewKind     `json:"kind"`
	Decision   DeliveryReviewDecision `json:"decision"`
	Comment    string                 `json:"comment,omitempty"`
}

type FreezePlanPayload struct {
	ID         string           `json:"id"`
	SolutionID string           `json:"solution_id"`
	WorkItems  []FrozenWorkItem `json:"work_items"`
}

type FrozenWorkItem struct {
	ID                   string        `json:"id"`
	Title                string        `json:"title"`
	Instructions         string        `json:"instructions"`
	BusinessDomain       string        `json:"business_domain,omitempty"`
	Priority             IssuePriority `json:"priority"`
	EstimatePoints       int           `json:"estimate_points,omitempty"`
	DependsOn            []string      `json:"depends_on,omitempty"`
	ComponentIDs         []string      `json:"component_ids,omitempty"`
	VerificationCommands [][]string    `json:"verification_commands"`
	EvidenceRequirements []string      `json:"evidence_requirements"`
	RiskLevel            IssueSeverity `json:"risk_level"`
}

type CreateChangeIntentPayload struct {
	ID              string   `json:"id"`
	RequestID       string   `json:"request_id"`
	FrozenPlanID    string   `json:"frozen_plan_id"`
	Reason          string   `json:"reason"`
	Impact          string   `json:"impact"`
	ProposedGoal    string   `json:"proposed_goal,omitempty"`
	AcceptanceDelta []string `json:"acceptance_delta,omitempty"`
}

type CreateDefectPayload struct {
	ID            string        `json:"id"`
	Title         string        `json:"title"`
	Description   string        `json:"description"`
	Severity      IssueSeverity `json:"severity"`
	Priority      IssuePriority `json:"priority"`
	Environment   string        `json:"environment,omitempty"`
	Module        string        `json:"module,omitempty"`
	AffectedScope string        `json:"affected_scope,omitempty"`
	OwnerID       string        `json:"owner_id,omitempty"`
}

type TransitionDefectPayload struct {
	DefectID     string       `json:"defect_id"`
	Status       DefectStatus `json:"status"`
	Reproduction string       `json:"reproduction,omitempty"`
	Expected     string       `json:"expected,omitempty"`
	Actual       string       `json:"actual,omitempty"`
	Containment  string       `json:"containment,omitempty"`
	RootCause    string       `json:"root_cause,omitempty"`
	Resolution   string       `json:"resolution,omitempty"`
	WorkItemIDs  []string     `json:"work_item_ids,omitempty"`
	EvidenceIDs  []string     `json:"evidence_ids,omitempty"`
}

type CreateRiskPayload struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Probability string `json:"probability"`
	Impact      string `json:"impact"`
	Trigger     string `json:"trigger"`
	OwnerID     string `json:"owner_id"`
}

type DecideRiskResponsePayload struct {
	RiskID           string       `json:"risk_id"`
	Response         RiskResponse `json:"response"`
	ResponsePlan     string       `json:"response_plan,omitempty"`
	AcceptanceReason string       `json:"acceptance_reason,omitempty"`
	ReviewAt         *time.Time   `json:"review_at"`
	WorkItemIDs      []string     `json:"work_item_ids,omitempty"`
}

type TransitionRiskPayload struct {
	RiskID      string     `json:"risk_id"`
	Status      RiskStatus `json:"status"`
	EvidenceIDs []string   `json:"evidence_ids,omitempty"`
}

type RecordDeliveryEvidencePayload struct {
	ID           string       `json:"id"`
	ResourceType ResourceType `json:"resource_type"`
	ResourceID   string       `json:"resource_id"`
	Kind         string       `json:"kind"`
	URI          string       `json:"uri"`
	SHA256       string       `json:"sha256,omitempty"`
	Summary      string       `json:"summary,omitempty"`
}
