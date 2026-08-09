package teamcontrol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
	"time"
)

type deliveryEventDraft struct {
	eventType DeliveryEventType
	streamID  string
	payload   any
}

type intentApprovedEvent struct {
	Request DeliveryRequest `json:"request"`
	Intent  IntentContract  `json:"intent"`
}

type solutionReviewEvent struct {
	Request  DeliveryRequest `json:"request"`
	Solution SolutionSpec    `json:"solution"`
}

type planFrozenEvent struct {
	Request   DeliveryRequest `json:"request"`
	Solution  SolutionSpec    `json:"solution"`
	Plan      FrozenPlan      `json:"plan"`
	WorkItems []WorkItem      `json:"work_items"`
}

type changeIntentEvent struct {
	Request DeliveryRequest `json:"request"`
	Change  ChangeIntent    `json:"change_intent"`
}

type DeliveryIntegrityReport struct {
	EventCount       int    `json:"event_count"`
	ProjectCount     int    `json:"project_count"`
	LastSequence     uint64 `json:"last_sequence"`
	LastHash         string `json:"last_hash,omitempty"`
	ProjectionStable bool   `json:"projection_stable"`
}

func (s *Service) ExecuteDeliveryCommand(
	actorID string,
	command DeliveryCommand,
) (DeliveryCommandResult, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return DeliveryCommandResult{}, err
	}
	command.ID, err = requireID(command.ID, "command_id")
	if err != nil {
		return DeliveryCommandResult{}, err
	}
	command.ProjectID, err = requireID(command.ProjectID, "project_id")
	if err != nil {
		return DeliveryCommandResult{}, err
	}
	if command.ActorID != "" && strings.TrimSpace(command.ActorID) != actorID {
		return DeliveryCommandResult{}, fmt.Errorf(
			"%w: command actor does not match the authenticated principal",
			ErrForbidden,
		)
	}
	command.ActorID = actorID
	if len(command.Payload) == 0 || !json.Valid(command.Payload) {
		return DeliveryCommandResult{}, fmt.Errorf("payload must be valid JSON")
	}
	action, err := deliveryAction(command.Type)
	if err != nil {
		return DeliveryCommandResult{}, err
	}
	commandHash, err := hashDeliveryCommand(command)
	if err != nil {
		return DeliveryCommandResult{}, err
	}

	var result DeliveryCommandResult
	err = s.store.updateWithChange(func(st *state) (bool, error) {
		if err := authorizeProject(st, actorID, command.ProjectID, action); err != nil {
			return false, err
		}
		commandKey := projectResourceKey(command.ProjectID, command.ID)
		if receipt, ok := st.Delivery.Commands[commandKey]; ok {
			if receipt.CommandHash != commandHash || receipt.CommandType != command.Type {
				return false, conflict("command id %q was already used with another payload", command.ID)
			}
			result.Receipt = receipt
			result.Events = deliveryEventsByID(st.Delivery.Events, receipt.EventIDs)
			return false, nil
		}

		projection := st.Delivery.Projects[command.ProjectID]
		normalizeDeliveryProjection(&projection, command.ProjectID)
		if projection.Revision != command.ExpectedRevision {
			return false, conflict(
				"delivery revision is %d, expected %d",
				projection.Revision,
				command.ExpectedRevision,
			)
		}
		draft, err := s.prepareDeliveryEvent(st, projection, command)
		if err != nil {
			return false, err
		}
		event, err := appendDeliveryEvent(&st.Delivery, command, draft)
		if err != nil {
			return false, err
		}
		projection = st.Delivery.Projects[command.ProjectID]
		receipt := DeliveryCommandReceipt{
			CommandID:       command.ID,
			ProjectID:       command.ProjectID,
			CommandType:     command.Type,
			CommandHash:     commandHash,
			EventIDs:        []string{event.ID},
			ProjectRevision: projection.Revision,
			RecordedAt:      event.OccurredAt,
		}
		st.Delivery.Commands[commandKey] = receipt
		result = DeliveryCommandResult{
			Receipt: receipt,
			Events:  []DeliveryEvent{event},
		}
		return true, nil
	})
	return result, err
}

func deliveryAction(commandType DeliveryCommandType) (Action, error) {
	switch commandType {
	case CommandDecideSolutionReview, CommandDecideRiskResponse:
		return ActionDeliveryReview, nil
	case CommandFreezePlan:
		return ActionDeliveryAccept, nil
	case CommandCreateRequest, CommandCreateSolution, CommandCreateChangeIntent,
		CommandCreateDefect, CommandTransitionDefect, CommandCreateRisk,
		CommandTransitionRisk, CommandRecordEvidence:
		return ActionDeliveryWrite, nil
	case CommandApproveIntent:
		return ActionDeliveryReview, nil
	default:
		return "", fmt.Errorf("unsupported delivery command %q", commandType)
	}
}

func hashDeliveryCommand(command DeliveryCommand) (string, error) {
	var payload any
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return "", err
	}
	return hashCanonical(struct {
		ID               string              `json:"id"`
		ProjectID        string              `json:"project_id"`
		Type             DeliveryCommandType `json:"type"`
		ActorID          string              `json:"actor_id"`
		ExpectedRevision uint64              `json:"expected_revision"`
		Payload          any                 `json:"payload"`
	}{
		ID:               command.ID,
		ProjectID:        command.ProjectID,
		Type:             command.Type,
		ActorID:          command.ActorID,
		ExpectedRevision: command.ExpectedRevision,
		Payload:          payload,
	})
}

func (s *Service) prepareDeliveryEvent(
	st *state,
	projection DeliveryProjection,
	command DeliveryCommand,
) (deliveryEventDraft, error) {
	now := time.Now().UTC()
	switch command.Type {
	case CommandCreateRequest:
		var input CreateRequestPayload
		if err := decodeDeliveryPayload(command.Payload, &input); err != nil {
			return deliveryEventDraft{}, err
		}
		id, err := normalizeID(input.ID, "request")
		if err != nil {
			return deliveryEventDraft{}, err
		}
		if _, exists := projection.Requests[id]; exists {
			return deliveryEventDraft{}, conflict("request %q already exists", id)
		}
		title, err := requireText(input.Title, "title", 300)
		if err != nil {
			return deliveryEventDraft{}, err
		}
		description, err := requireText(input.Description, "description", 12000)
		if err != nil {
			return deliveryEventDraft{}, err
		}
		criteria, err := cleanDeliveryTextList(input.AcceptanceCriteria, "acceptance_criteria", 100, true)
		if err != nil {
			return deliveryEventDraft{}, err
		}
		nonGoals, err := cleanDeliveryTextList(input.NonGoals, "non_goals", 100, false)
		if err != nil {
			return deliveryEventDraft{}, err
		}
		constraints, err := cleanDeliveryTextList(input.Constraints, "constraints", 100, false)
		if err != nil {
			return deliveryEventDraft{}, err
		}
		request := DeliveryRequest{
			ID:                 id,
			ProjectID:          command.ProjectID,
			Title:              title,
			Description:        description,
			Source:             strings.TrimSpace(input.Source),
			Status:             RequestDraft,
			Revision:           1,
			AcceptanceCriteria: criteria,
			NonGoals:           nonGoals,
			Constraints:        constraints,
			CreatedBy:          command.ActorID,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		return deliveryEventDraft{EventRequestCreated, "request:" + id, request}, nil

	case CommandApproveIntent:
		var input ApproveIntentPayload
		if err := decodeDeliveryPayload(command.Payload, &input); err != nil {
			return deliveryEventDraft{}, err
		}
		requestID, err := requireID(input.RequestID, "request_id")
		if err != nil {
			return deliveryEventDraft{}, err
		}
		request, ok := projection.Requests[requestID]
		if !ok {
			return deliveryEventDraft{}, entityNotFound("request", requestID)
		}
		if request.Status != RequestDraft && request.Status != RequestChangePending {
			return deliveryEventDraft{}, conflict("request %q cannot approve intent from %q", requestID, request.Status)
		}
		id, err := normalizeID(input.ID, "intent")
		if err != nil {
			return deliveryEventDraft{}, err
		}
		if _, exists := projection.Intents[id]; exists {
			return deliveryEventDraft{}, conflict("intent %q already exists", id)
		}
		goal, err := requireText(input.Goal, "goal", 4000)
		if err != nil {
			return deliveryEventDraft{}, err
		}
		criteria, err := cleanDeliveryTextList(input.AcceptanceCriteria, "acceptance_criteria", 100, true)
		if err != nil {
			return deliveryEventDraft{}, err
		}
		intent := IntentContract{
			ID:                 id,
			ProjectID:          command.ProjectID,
			RequestID:          requestID,
			Goal:               goal,
			Users:              cleanLooseTextList(input.Users),
			Scenarios:          cleanLooseTextList(input.Scenarios),
			Scope:              cleanLooseTextList(input.Scope),
			NonGoals:           cleanLooseTextList(input.NonGoals),
			Constraints:        cleanLooseTextList(input.Constraints),
			AcceptanceCriteria: criteria,
			RiskBoundary:       strings.TrimSpace(input.RiskBoundary),
			CostBoundary:       strings.TrimSpace(input.CostBoundary),
			Revision:           request.Revision,
			ApprovedBy:         command.ActorID,
			ApprovedAt:         now,
		}
		request.Status = RequestIntentApproved
		request.AcceptanceCriteria = slices.Clone(criteria)
		request.UpdatedAt = now
		return deliveryEventDraft{
			EventIntentApproved,
			"request:" + requestID,
			intentApprovedEvent{Request: request, Intent: intent},
		}, nil

	case CommandCreateSolution:
		var input CreateSolutionPayload
		if err := decodeDeliveryPayload(command.Payload, &input); err != nil {
			return deliveryEventDraft{}, err
		}
		request, intent, err := resolveRequestIntent(projection, input.RequestID, input.IntentID)
		if err != nil {
			return deliveryEventDraft{}, err
		}
		if request.Status != RequestIntentApproved {
			return deliveryEventDraft{}, conflict("request %q has no approved intent available for a solution", request.ID)
		}
		id, err := normalizeID(input.ID, "solution")
		if err != nil {
			return deliveryEventDraft{}, err
		}
		if _, exists := projection.Solutions[id]; exists {
			return deliveryEventDraft{}, conflict("solution %q already exists", id)
		}
		title, err := requireText(input.Title, "title", 300)
		if err != nil {
			return deliveryEventDraft{}, err
		}
		summary, err := requireText(input.Summary, "summary", 16000)
		if err != nil {
			return deliveryEventDraft{}, err
		}
		tests, err := cleanDeliveryTextList(input.TestStrategy, "test_strategy", 100, true)
		if err != nil {
			return deliveryEventDraft{}, err
		}
		rollback, err := requireText(input.RollbackPlan, "rollback_plan", 4000)
		if err != nil {
			return deliveryEventDraft{}, err
		}
		solution := SolutionSpec{
			ID:             id,
			ProjectID:      command.ProjectID,
			RequestID:      request.ID,
			IntentID:       intent.ID,
			Title:          title,
			Summary:        summary,
			ADRRefs:        cleanLooseTextList(input.ADRRefs),
			AllowedPaths:   cleanLooseTextList(input.AllowedPaths),
			ForbiddenPaths: cleanLooseTextList(input.ForbiddenPaths),
			TestStrategy:   tests,
			RollbackPlan:   rollback,
			Status:         SolutionProposed,
			Revision:       request.Revision,
			Reviews:        pendingDeliveryReviews(),
			CreatedBy:      command.ActorID,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		return deliveryEventDraft{EventSolutionCreated, "solution:" + id, solution}, nil

	case CommandDecideSolutionReview:
		var input DecideSolutionReviewPayload
		if err := decodeDeliveryPayload(command.Payload, &input); err != nil {
			return deliveryEventDraft{}, err
		}
		solutionID, err := requireID(input.SolutionID, "solution_id")
		if err != nil {
			return deliveryEventDraft{}, err
		}
		solution, ok := projection.Solutions[solutionID]
		if !ok {
			return deliveryEventDraft{}, entityNotFound("solution", solutionID)
		}
		if solution.Status == SolutionFrozen {
			return deliveryEventDraft{}, conflict("frozen solution cannot be reviewed in place")
		}
		if solution.CreatedBy == command.ActorID {
			return deliveryEventDraft{}, fmt.Errorf("%w: solution creator cannot review their own solution", ErrForbidden)
		}
		if !validDeliveryReviewKind(input.Kind) ||
			input.Decision != DeliveryReviewApproved && input.Decision != DeliveryReviewRejected {
			return deliveryEventDraft{}, fmt.Errorf("unsupported review kind or decision")
		}
		decidedAt := now
		solution.Reviews[input.Kind] = DeliveryReview{
			Kind: input.Kind, Decision: input.Decision, Reviewer: command.ActorID,
			Comment: strings.TrimSpace(input.Comment), DecidedAt: &decidedAt,
		}
		solution.Status = SolutionProposed
		if allDeliveryReviewsApproved(solution.Reviews) {
			solution.Status = SolutionApproved
		}
		solution.UpdatedAt = now
		request := projection.Requests[solution.RequestID]
		request.Status = RequestReviewPending
		request.UpdatedAt = now
		return deliveryEventDraft{
			EventSolutionReviewDecided,
			"solution:" + solutionID,
			solutionReviewEvent{Request: request, Solution: solution},
		}, nil

	case CommandFreezePlan:
		var input FreezePlanPayload
		if err := decodeDeliveryPayload(command.Payload, &input); err != nil {
			return deliveryEventDraft{}, err
		}
		solutionID, err := requireID(input.SolutionID, "solution_id")
		if err != nil {
			return deliveryEventDraft{}, err
		}
		solution, ok := projection.Solutions[solutionID]
		if !ok {
			return deliveryEventDraft{}, entityNotFound("solution", solutionID)
		}
		if solution.Status != SolutionApproved || !allDeliveryReviewsApproved(solution.Reviews) {
			return deliveryEventDraft{}, conflict("all four reviews must be approved before freeze")
		}
		if solution.CreatedBy == command.ActorID {
			return deliveryEventDraft{}, fmt.Errorf("%w: solution creator cannot freeze their own solution", ErrForbidden)
		}
		id, err := normalizeID(input.ID, "plan")
		if err != nil {
			return deliveryEventDraft{}, err
		}
		if _, exists := projection.FrozenPlans[id]; exists {
			return deliveryEventDraft{}, conflict("frozen plan %q already exists", id)
		}
		request := projection.Requests[solution.RequestID]
		intent := projection.Intents[solution.IntentID]
		workItems, err := createFrozenWorkItems(st, command, request, intent, solution, input.WorkItems, now)
		if err != nil {
			return deliveryEventDraft{}, err
		}
		material := struct {
			Request   DeliveryRequest `json:"request"`
			Intent    IntentContract  `json:"intent"`
			Solution  SolutionSpec    `json:"solution"`
			WorkItems []WorkItem      `json:"work_items"`
		}{request, intent, solution, workItems}
		bundleHash, err := hashCanonical(material)
		if err != nil {
			return deliveryEventDraft{}, err
		}
		workItemIDs := make([]string, 0, len(workItems))
		for _, item := range workItems {
			workItemIDs = append(workItemIDs, item.ID)
		}
		plan := FrozenPlan{
			ID: id, ProjectID: command.ProjectID, RequestID: request.ID,
			IntentID: intent.ID, SolutionID: solution.ID,
			BundleRevision: request.Revision, BundleHash: bundleHash,
			WorkItemIDs: workItemIDs, FrozenBy: command.ActorID, FrozenAt: now,
		}
		request.Status = RequestFrozen
		request.UpdatedAt = now
		solution.Status = SolutionFrozen
		solution.UpdatedAt = now
		return deliveryEventDraft{
			EventPlanFrozen,
			"solution:" + solutionID,
			planFrozenEvent{Request: request, Solution: solution, Plan: plan, WorkItems: workItems},
		}, nil

	case CommandCreateChangeIntent:
		var input CreateChangeIntentPayload
		if err := decodeDeliveryPayload(command.Payload, &input); err != nil {
			return deliveryEventDraft{}, err
		}
		requestID, err := requireID(input.RequestID, "request_id")
		if err != nil {
			return deliveryEventDraft{}, err
		}
		planID, err := requireID(input.FrozenPlanID, "frozen_plan_id")
		if err != nil {
			return deliveryEventDraft{}, err
		}
		request, ok := projection.Requests[requestID]
		if !ok {
			return deliveryEventDraft{}, entityNotFound("request", requestID)
		}
		plan, ok := projection.FrozenPlans[planID]
		if !ok || plan.RequestID != requestID {
			return deliveryEventDraft{}, entityNotFound("frozen_plan", planID)
		}
		if request.Status != RequestFrozen {
			return deliveryEventDraft{}, conflict("change intent requires a frozen request")
		}
		id, err := normalizeID(input.ID, "change")
		if err != nil {
			return deliveryEventDraft{}, err
		}
		reason, err := requireText(input.Reason, "reason", 4000)
		if err != nil {
			return deliveryEventDraft{}, err
		}
		impact, err := requireText(input.Impact, "impact", 8000)
		if err != nil {
			return deliveryEventDraft{}, err
		}
		change := ChangeIntent{
			ID: id, ProjectID: command.ProjectID, RequestID: requestID,
			FrozenPlanID: planID, Reason: reason, Impact: impact,
			ProposedGoal:    strings.TrimSpace(input.ProposedGoal),
			AcceptanceDelta: cleanLooseTextList(input.AcceptanceDelta),
			Status:          ChangeIntentPending, CreatedBy: command.ActorID, CreatedAt: now,
		}
		request.Status = RequestChangePending
		request.UpdatedAt = now
		return deliveryEventDraft{
			EventChangeIntentCreated, "request:" + requestID,
			changeIntentEvent{Request: request, Change: change},
		}, nil

	case CommandCreateDefect:
		var input CreateDefectPayload
		if err := decodeDeliveryPayload(command.Payload, &input); err != nil {
			return deliveryEventDraft{}, err
		}
		id, err := normalizeID(input.ID, "defect")
		if err != nil {
			return deliveryEventDraft{}, err
		}
		if _, exists := projection.Defects[id]; exists {
			return deliveryEventDraft{}, conflict("defect %q already exists", id)
		}
		if !validSeverity(input.Severity) || !validPriority(input.Priority) {
			return deliveryEventDraft{}, fmt.Errorf("invalid severity or priority")
		}
		title, err := requireText(input.Title, "title", 300)
		if err != nil {
			return deliveryEventDraft{}, err
		}
		description, err := requireText(input.Description, "description", 12000)
		if err != nil {
			return deliveryEventDraft{}, err
		}
		ownerID := strings.TrimSpace(input.OwnerID)
		if ownerID != "" {
			if err := requireActiveProjectMember(st, command.ProjectID, ownerID, "owner_id"); err != nil {
				return deliveryEventDraft{}, err
			}
		}
		defect := Defect{
			ID: id, ProjectID: command.ProjectID, Title: title,
			Description: description, Status: DefectReported,
			Severity: input.Severity, Priority: input.Priority,
			Environment:   strings.TrimSpace(input.Environment),
			Module:        strings.TrimSpace(input.Module),
			AffectedScope: strings.TrimSpace(input.AffectedScope),
			ReporterID:    command.ActorID, OwnerID: ownerID,
			Revision: 1, CreatedAt: now, UpdatedAt: now,
		}
		return deliveryEventDraft{EventDefectCreated, "defect:" + id, defect}, nil

	case CommandTransitionDefect:
		var input TransitionDefectPayload
		if err := decodeDeliveryPayload(command.Payload, &input); err != nil {
			return deliveryEventDraft{}, err
		}
		id, err := requireID(input.DefectID, "defect_id")
		if err != nil {
			return deliveryEventDraft{}, err
		}
		defect, ok := projection.Defects[id]
		if !ok {
			return deliveryEventDraft{}, entityNotFound("defect", id)
		}
		if !validDefectTransition(defect.Status, input.Status) {
			return deliveryEventDraft{}, fmt.Errorf("%w: defect %q cannot move from %q to %q", ErrInvalidTransition, id, defect.Status, input.Status)
		}
		if err := validateDeliveryWorkItems(st, command.ProjectID, input.WorkItemIDs); err != nil {
			return deliveryEventDraft{}, err
		}
		if err := validateDeliveryEvidenceRefs(projection, ResourceDefect, id, input.EvidenceIDs); err != nil {
			return deliveryEventDraft{}, err
		}
		mergeDefectTransition(&defect, input)
		if err := validateDefectGate(defect, input.Status); err != nil {
			return deliveryEventDraft{}, err
		}
		if input.Status == DefectReopened {
			defect.ReopenCount++
		}
		defect.Status = input.Status
		defect.Revision++
		defect.UpdatedAt = now
		return deliveryEventDraft{EventDefectTransitioned, "defect:" + id, defect}, nil

	case CommandCreateRisk:
		var input CreateRiskPayload
		if err := decodeDeliveryPayload(command.Payload, &input); err != nil {
			return deliveryEventDraft{}, err
		}
		id, err := normalizeID(input.ID, "risk")
		if err != nil {
			return deliveryEventDraft{}, err
		}
		if _, exists := projection.Risks[id]; exists {
			return deliveryEventDraft{}, conflict("risk %q already exists", id)
		}
		ownerID, err := requireID(input.OwnerID, "owner_id")
		if err != nil {
			return deliveryEventDraft{}, err
		}
		if err := requireActiveProjectMember(st, command.ProjectID, ownerID, "owner_id"); err != nil {
			return deliveryEventDraft{}, err
		}
		title, err := requireText(input.Title, "title", 300)
		if err != nil {
			return deliveryEventDraft{}, err
		}
		description, err := requireText(input.Description, "description", 12000)
		if err != nil {
			return deliveryEventDraft{}, err
		}
		trigger, err := requireText(input.Trigger, "trigger", 4000)
		if err != nil {
			return deliveryEventDraft{}, err
		}
		if !validRiskLevel(input.Probability) || !validRiskLevel(input.Impact) {
			return deliveryEventDraft{}, fmt.Errorf("probability and impact must be low, medium, or high")
		}
		risk := Risk{
			ID: id, ProjectID: command.ProjectID, Title: title,
			Description: description, Status: RiskIdentified,
			Probability: input.Probability, Impact: input.Impact, Trigger: trigger,
			OwnerID: ownerID, Revision: 1, CreatedBy: command.ActorID,
			CreatedAt: now, UpdatedAt: now,
		}
		return deliveryEventDraft{EventRiskCreated, "risk:" + id, risk}, nil

	case CommandDecideRiskResponse:
		var input DecideRiskResponsePayload
		if err := decodeDeliveryPayload(command.Payload, &input); err != nil {
			return deliveryEventDraft{}, err
		}
		id, err := requireID(input.RiskID, "risk_id")
		if err != nil {
			return deliveryEventDraft{}, err
		}
		risk, ok := projection.Risks[id]
		if !ok {
			return deliveryEventDraft{}, entityNotFound("risk", id)
		}
		if risk.Status != RiskAssessed {
			return deliveryEventDraft{}, conflict("risk response requires assessed status")
		}
		if risk.CreatedBy == command.ActorID {
			return deliveryEventDraft{}, fmt.Errorf("%w: risk creator cannot decide their own response", ErrForbidden)
		}
		if err := validateRiskResponse(input, now); err != nil {
			return deliveryEventDraft{}, err
		}
		if err := validateDeliveryWorkItems(st, command.ProjectID, input.WorkItemIDs); err != nil {
			return deliveryEventDraft{}, err
		}
		risk.Response = input.Response
		risk.ResponsePlan = strings.TrimSpace(input.ResponsePlan)
		risk.AcceptanceReason = strings.TrimSpace(input.AcceptanceReason)
		risk.ReviewAt = input.ReviewAt
		risk.WorkItemIDs, err = uniqueIDs(input.WorkItemIDs, "work_item_id")
		if err != nil {
			return deliveryEventDraft{}, err
		}
		if input.Response == RiskAvoid || input.Response == RiskMitigate {
			risk.Status = RiskMitigating
		} else {
			risk.Status = RiskMonitoring
		}
		risk.Revision++
		risk.UpdatedAt = now
		return deliveryEventDraft{EventRiskResponseDecided, "risk:" + id, risk}, nil

	case CommandTransitionRisk:
		var input TransitionRiskPayload
		if err := decodeDeliveryPayload(command.Payload, &input); err != nil {
			return deliveryEventDraft{}, err
		}
		id, err := requireID(input.RiskID, "risk_id")
		if err != nil {
			return deliveryEventDraft{}, err
		}
		risk, ok := projection.Risks[id]
		if !ok {
			return deliveryEventDraft{}, entityNotFound("risk", id)
		}
		if !validRiskTransition(risk.Status, input.Status) {
			return deliveryEventDraft{}, fmt.Errorf("%w: risk %q cannot move from %q to %q", ErrInvalidTransition, id, risk.Status, input.Status)
		}
		if input.Status == RiskReviewed && len(input.EvidenceIDs) == 0 {
			return deliveryEventDraft{}, conflict("reviewed risk requires evidence")
		}
		if err := validateDeliveryEvidenceRefs(projection, ResourceRisk, id, input.EvidenceIDs); err != nil {
			return deliveryEventDraft{}, err
		}
		risk.EvidenceIDs = appendUnique(risk.EvidenceIDs, input.EvidenceIDs...)
		risk.Status = input.Status
		risk.Revision++
		risk.UpdatedAt = now
		return deliveryEventDraft{EventRiskTransitioned, "risk:" + id, risk}, nil

	case CommandRecordEvidence:
		var input RecordDeliveryEvidencePayload
		if err := decodeDeliveryPayload(command.Payload, &input); err != nil {
			return deliveryEventDraft{}, err
		}
		id, err := normalizeID(input.ID, "evidence")
		if err != nil {
			return deliveryEventDraft{}, err
		}
		if _, exists := projection.Evidence[id]; exists {
			return deliveryEventDraft{}, conflict("evidence %q already exists", id)
		}
		resourceID, err := requireID(input.ResourceID, "resource_id")
		if err != nil {
			return deliveryEventDraft{}, err
		}
		if !deliveryResourceExists(st, projection, input.ResourceType, resourceID) {
			return deliveryEventDraft{}, entityNotFound(string(input.ResourceType), resourceID)
		}
		uri, err := validateRegistryURI(input.URI, "uri")
		if err != nil {
			return deliveryEventDraft{}, err
		}
		sha, err := validateOptionalSHA256(input.SHA256)
		if err != nil {
			return deliveryEventDraft{}, err
		}
		kind, err := requireKey(input.Kind, "kind")
		if err != nil {
			return deliveryEventDraft{}, err
		}
		evidence := DeliveryEvidence{
			ID: id, ProjectID: command.ProjectID, ResourceType: input.ResourceType,
			ResourceID: resourceID, Kind: kind, URI: uri, SHA256: sha,
			Summary:    strings.TrimSpace(input.Summary),
			RecordedBy: command.ActorID, RecordedAt: now,
		}
		return deliveryEventDraft{EventDeliveryEvidenceRecorded, string(input.ResourceType) + ":" + resourceID, evidence}, nil
	default:
		return deliveryEventDraft{}, fmt.Errorf("unsupported delivery command %q", command.Type)
	}
}

func appendDeliveryEvent(
	state *DeliveryState,
	command DeliveryCommand,
	draft deliveryEventDraft,
) (DeliveryEvent, error) {
	normalizeDeliveryState(state)
	payload, err := json.Marshal(draft.payload)
	if err != nil {
		return DeliveryEvent{}, err
	}
	streamVersion := uint64(1)
	for i := len(state.Events) - 1; i >= 0; i-- {
		if state.Events[i].ProjectID == command.ProjectID &&
			state.Events[i].StreamID == draft.streamID {
			streamVersion = state.Events[i].StreamVersion + 1
			break
		}
	}
	event := DeliveryEvent{
		ID:            newID("event"),
		ProjectID:     command.ProjectID,
		StreamID:      draft.streamID,
		StreamVersion: streamVersion,
		Sequence:      uint64(len(state.Events) + 1),
		SchemaVersion: DeliveryEventSchemaVersion,
		Type:          draft.eventType,
		ActorID:       command.ActorID,
		CommandID:     command.ID,
		Payload:       payload,
		OccurredAt:    time.Now().UTC(),
		PreviousHash:  state.LastHash,
	}
	event.CommandHash, err = hashDeliveryCommand(command)
	if err != nil {
		return DeliveryEvent{}, err
	}
	event.Hash, err = hashDeliveryEvent(event)
	if err != nil {
		return DeliveryEvent{}, err
	}
	projection := state.Projects[command.ProjectID]
	normalizeDeliveryProjection(&projection, command.ProjectID)
	if err := applyDeliveryEvent(&projection, event); err != nil {
		return DeliveryEvent{}, err
	}
	state.Events = append(state.Events, event)
	state.LastHash = event.Hash
	state.Projects[command.ProjectID] = projection
	return event, nil
}

func hashDeliveryEvent(event DeliveryEvent) (string, error) {
	var payload any
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return "", err
	}
	return hashCanonical(struct {
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
		Payload       any               `json:"payload"`
		OccurredAt    time.Time         `json:"occurred_at"`
		PreviousHash  string            `json:"previous_hash,omitempty"`
	}{
		event.ID, event.ProjectID, event.StreamID, event.StreamVersion,
		event.Sequence, event.SchemaVersion, event.Type, event.ActorID,
		event.CommandID, event.CommandHash, payload, event.OccurredAt, event.PreviousHash,
	})
}

func applyDeliveryEvent(projection *DeliveryProjection, event DeliveryEvent) error {
	if event.SchemaVersion != DeliveryEventSchemaVersion {
		return fmt.Errorf("unsupported delivery event schema %d", event.SchemaVersion)
	}
	normalizeDeliveryProjection(projection, event.ProjectID)
	switch event.Type {
	case EventRequestCreated:
		var value DeliveryRequest
		if err := decodeDeliveryPayload(event.Payload, &value); err != nil {
			return err
		}
		projection.Requests[value.ID] = value
	case EventIntentApproved:
		var value intentApprovedEvent
		if err := decodeDeliveryPayload(event.Payload, &value); err != nil {
			return err
		}
		projection.Requests[value.Request.ID] = value.Request
		projection.Intents[value.Intent.ID] = value.Intent
	case EventSolutionCreated:
		var value SolutionSpec
		if err := decodeDeliveryPayload(event.Payload, &value); err != nil {
			return err
		}
		projection.Solutions[value.ID] = value
	case EventSolutionReviewDecided:
		var value solutionReviewEvent
		if err := decodeDeliveryPayload(event.Payload, &value); err != nil {
			return err
		}
		projection.Requests[value.Request.ID] = value.Request
		projection.Solutions[value.Solution.ID] = value.Solution
	case EventPlanFrozen:
		var value planFrozenEvent
		if err := decodeDeliveryPayload(event.Payload, &value); err != nil {
			return err
		}
		projection.Requests[value.Request.ID] = value.Request
		projection.Solutions[value.Solution.ID] = value.Solution
		projection.FrozenPlans[value.Plan.ID] = value.Plan
	case EventChangeIntentCreated:
		var value changeIntentEvent
		if err := decodeDeliveryPayload(event.Payload, &value); err != nil {
			return err
		}
		projection.Requests[value.Request.ID] = value.Request
		projection.ChangeIntents[value.Change.ID] = value.Change
	case EventDefectCreated, EventDefectTransitioned:
		var value Defect
		if err := decodeDeliveryPayload(event.Payload, &value); err != nil {
			return err
		}
		projection.Defects[value.ID] = value
	case EventRiskCreated, EventRiskResponseDecided, EventRiskTransitioned:
		var value Risk
		if err := decodeDeliveryPayload(event.Payload, &value); err != nil {
			return err
		}
		projection.Risks[value.ID] = value
	case EventDeliveryEvidenceRecorded:
		var value DeliveryEvidence
		if err := decodeDeliveryPayload(event.Payload, &value); err != nil {
			return err
		}
		projection.Evidence[value.ID] = value
	default:
		return fmt.Errorf("unsupported delivery event %q", event.Type)
	}
	projection.Revision++
	projection.UpdatedAt = event.OccurredAt
	return nil
}

func validateDeliveryJournal(st *state) error {
	normalizeDeliveryState(&st.Delivery)
	replayed := make(map[string]DeliveryProjection)
	streamVersions := make(map[string]uint64)
	previousHash := ""
	for index, event := range st.Delivery.Events {
		expectedSequence := uint64(index + 1)
		if event.Sequence != expectedSequence {
			return fmt.Errorf("sequence gap at event %q: got %d want %d", event.ID, event.Sequence, expectedSequence)
		}
		if event.PreviousHash != previousHash {
			return fmt.Errorf("previous hash mismatch at event %q", event.ID)
		}
		expectedStreamVersion := streamVersions[event.ProjectID+"/"+event.StreamID] + 1
		if event.StreamVersion != expectedStreamVersion {
			return fmt.Errorf("stream version mismatch at event %q", event.ID)
		}
		hash, err := hashDeliveryEvent(event)
		if err != nil {
			return err
		}
		if hash != event.Hash {
			return fmt.Errorf("hash mismatch at event %q", event.ID)
		}
		projection := replayed[event.ProjectID]
		normalizeDeliveryProjection(&projection, event.ProjectID)
		if err := applyDeliveryEvent(&projection, event); err != nil {
			return err
		}
		replayed[event.ProjectID] = projection
		streamVersions[event.ProjectID+"/"+event.StreamID] = event.StreamVersion
		previousHash = event.Hash
	}
	if st.Delivery.LastHash != previousHash {
		return fmt.Errorf("delivery last hash mismatch")
	}
	for projectID, stored := range st.Delivery.Projects {
		normalizeDeliveryProjection(&stored, projectID)
		replay := replayed[projectID]
		normalizeDeliveryProjection(&replay, projectID)
		if !reflect.DeepEqual(stored, replay) {
			return fmt.Errorf("delivery projection mismatch for project %q", projectID)
		}
	}
	if len(replayed) != len(st.Delivery.Projects) {
		return fmt.Errorf("delivery projection project count mismatch")
	}
	for key, receipt := range st.Delivery.Commands {
		if key != projectResourceKey(receipt.ProjectID, receipt.CommandID) {
			return fmt.Errorf("command receipt key mismatch for %q", receipt.CommandID)
		}
		receiptEvents := deliveryEventsByID(st.Delivery.Events, receipt.EventIDs)
		if len(receiptEvents) != len(receipt.EventIDs) {
			return fmt.Errorf("command receipt %q references missing events", receipt.CommandID)
		}
		for _, event := range receiptEvents {
			if event.ProjectID != receipt.ProjectID || event.CommandID != receipt.CommandID ||
				event.CommandHash != receipt.CommandHash {
				return fmt.Errorf("command receipt %q does not match its event", receipt.CommandID)
			}
		}
	}
	for _, event := range st.Delivery.Events {
		receipt, ok := st.Delivery.Commands[projectResourceKey(event.ProjectID, event.CommandID)]
		if !ok || !slices.Contains(receipt.EventIDs, event.ID) {
			return fmt.Errorf("delivery event %q has no matching command receipt", event.ID)
		}
	}
	return nil
}

func (s *Service) VerifyDeliveryIntegrity(userID, projectID string) (DeliveryIntegrityReport, error) {
	var report DeliveryIntegrityReport
	err := s.readProject(userID, projectID, ActionDeliveryRead, func(st state, _ Project) error {
		if err := validateDeliveryJournal(&st); err != nil {
			return err
		}
		report.ProjectCount = 1
		report.ProjectionStable = true
		for _, event := range st.Delivery.Events {
			if event.ProjectID != projectID {
				continue
			}
			report.EventCount++
			report.LastSequence = event.Sequence
			report.LastHash = event.Hash
		}
		return nil
	})
	return report, err
}

func (s *Service) GetDeliveryProjection(
	userID, projectID string,
) (DeliveryProjection, error) {
	var result DeliveryProjection
	err := s.readProject(userID, projectID, ActionDeliveryRead, func(st state, _ Project) error {
		value := st.Delivery.Projects[projectID]
		normalizeDeliveryProjection(&value, projectID)
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, &result)
	})
	return result, err
}

func (s *Service) ListDeliveryEvents(
	userID, projectID string,
	afterSequence uint64,
	limit int,
) ([]DeliveryEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var result []DeliveryEvent
	err := s.readProject(userID, projectID, ActionDeliveryRead, func(st state, _ Project) error {
		for _, event := range st.Delivery.Events {
			if event.ProjectID == projectID && event.Sequence > afterSequence {
				result = append(result, event)
				if len(result) == limit {
					break
				}
			}
		}
		return nil
	})
	return result, err
}

func decodeDeliveryPayload(data json.RawMessage, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode delivery payload: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("decode delivery payload: multiple JSON values")
		}
		return fmt.Errorf("decode delivery payload: %w", err)
	}
	return nil
}

func requireActiveProjectMember(
	st *state,
	projectID, userID, field string,
) error {
	if err := requireActiveUser(st, userID); err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}
	membership := findProjectMembership(st, projectID, userID)
	if membership == nil || membership.Status != MembershipActive {
		return fmt.Errorf("%s: %w: user %q is not an active project member", field, ErrForbidden, userID)
	}
	return nil
}

func validateDeliveryWorkItems(st *state, projectID string, ids []string) error {
	for _, rawID := range ids {
		id, err := requireID(rawID, "work_item_id")
		if err != nil {
			return err
		}
		item, ok := st.WorkItems[id]
		if !ok || item.ProjectID != projectID {
			return entityNotFound("work_item", id)
		}
	}
	return nil
}

func validateDeliveryEvidenceRefs(
	projection DeliveryProjection,
	resourceType ResourceType,
	resourceID string,
	ids []string,
) error {
	for _, rawID := range ids {
		id, err := requireID(rawID, "evidence_id")
		if err != nil {
			return err
		}
		evidence, ok := projection.Evidence[id]
		if !ok || evidence.ResourceType != resourceType || evidence.ResourceID != resourceID {
			return entityNotFound("evidence", id)
		}
	}
	return nil
}

func pendingDeliveryReviews() map[DeliveryReviewKind]DeliveryReview {
	return map[DeliveryReviewKind]DeliveryReview{
		DeliveryReviewScenario: {Kind: DeliveryReviewScenario, Decision: DeliveryReviewPending},
		DeliveryReviewCapacity: {Kind: DeliveryReviewCapacity, Decision: DeliveryReviewPending},
		DeliveryReviewRisk:     {Kind: DeliveryReviewRisk, Decision: DeliveryReviewPending},
		DeliveryReviewCost:     {Kind: DeliveryReviewCost, Decision: DeliveryReviewPending},
	}
}

func validDeliveryReviewKind(value DeliveryReviewKind) bool {
	switch value {
	case DeliveryReviewScenario, DeliveryReviewCapacity, DeliveryReviewRisk, DeliveryReviewCost:
		return true
	default:
		return false
	}
}

func allDeliveryReviewsApproved(values map[DeliveryReviewKind]DeliveryReview) bool {
	for _, kind := range []DeliveryReviewKind{
		DeliveryReviewScenario, DeliveryReviewCapacity, DeliveryReviewRisk, DeliveryReviewCost,
	} {
		if values[kind].Decision != DeliveryReviewApproved {
			return false
		}
	}
	return true
}

func resolveRequestIntent(
	projection DeliveryProjection,
	requestID, intentID string,
) (DeliveryRequest, IntentContract, error) {
	requestID, err := requireID(requestID, "request_id")
	if err != nil {
		return DeliveryRequest{}, IntentContract{}, err
	}
	intentID, err = requireID(intentID, "intent_id")
	if err != nil {
		return DeliveryRequest{}, IntentContract{}, err
	}
	request, ok := projection.Requests[requestID]
	if !ok {
		return DeliveryRequest{}, IntentContract{}, entityNotFound("request", requestID)
	}
	intent, ok := projection.Intents[intentID]
	if !ok || intent.RequestID != requestID {
		return DeliveryRequest{}, IntentContract{}, entityNotFound("intent", intentID)
	}
	return request, intent, nil
}

func createFrozenWorkItems(
	st *state,
	command DeliveryCommand,
	request DeliveryRequest,
	intent IntentContract,
	solution SolutionSpec,
	input []FrozenWorkItem,
	now time.Time,
) ([]WorkItem, error) {
	if len(input) == 0 || len(input) > 200 {
		return nil, fmt.Errorf("work_items must contain 1-200 entries")
	}
	ids := make(map[string]struct{}, len(input))
	result := make([]WorkItem, 0, len(input))
	for _, candidate := range input {
		id, err := normalizeID(candidate.ID, "work")
		if err != nil {
			return nil, err
		}
		if _, exists := ids[id]; exists {
			return nil, conflict("duplicate work item %q", id)
		}
		if _, exists := st.WorkItems[id]; exists {
			return nil, conflict("work item %q already exists", id)
		}
		ids[id] = struct{}{}
		if !validPriority(candidate.Priority) || !validSeverity(candidate.RiskLevel) {
			return nil, fmt.Errorf("work item %q has invalid priority or risk level", id)
		}
		if candidate.EstimatePoints < 0 || candidate.EstimatePoints > 10_000 {
			return nil, fmt.Errorf("work item %q estimate_points must be between 0 and 10000", id)
		}
		title, err := requireText(candidate.Title, "work item title", 300)
		if err != nil {
			return nil, err
		}
		instructions, err := requireText(candidate.Instructions, "work item instructions", 12000)
		if err != nil {
			return nil, err
		}
		dependsOn, err := uniqueIDs(candidate.DependsOn, "depends_on")
		if err != nil {
			return nil, err
		}
		componentIDs, err := uniqueIDs(candidate.ComponentIDs, "component_id")
		if err != nil {
			return nil, err
		}
		for _, componentID := range componentIDs {
			component, ok := st.Components[componentID]
			if !ok || component.ProjectID != command.ProjectID {
				return nil, entityNotFound("component", componentID)
			}
		}
		commands, err := normalizeCommands(candidate.VerificationCommands)
		if err != nil || len(commands) == 0 {
			return nil, conflict("work item %q requires valid verification commands", id)
		}
		evidenceRequirements, err := cleanDeliveryTextList(
			candidate.EvidenceRequirements,
			"evidence_requirements",
			100,
			true,
		)
		if err != nil {
			return nil, err
		}
		status := WorkItemReady
		if len(dependsOn) > 0 {
			status = WorkItemPending
		}
		item := WorkItem{
			ID: id, ProjectID: command.ProjectID, Title: title,
			Instructions: instructions, BusinessDomain: strings.TrimSpace(candidate.BusinessDomain),
			Priority: candidate.Priority, EstimatePoints: candidate.EstimatePoints,
			Status: status, DependsOn: dependsOn,
			ComponentIDs:         componentIDs,
			VerificationCommands: commands,
			SourceType:           ResourceRequest, SourceID: request.ID, ContractID: intent.ID,
			ObjectiveRef:         solution.ID,
			EvidenceRequirements: evidenceRequirements,
			RiskLevel:            candidate.RiskLevel, Revision: 1,
			CreatedBy: command.ActorID, CreatedAt: now, UpdatedAt: now,
		}
		result = append(result, item)
	}
	if err := validateFrozenWorkItemDependencies(st, command.ProjectID, result); err != nil {
		return nil, err
	}
	for _, item := range result {
		st.WorkItems[item.ID] = item
	}
	return result, nil
}

func validateFrozenWorkItemDependencies(st *state, projectID string, items []WorkItem) error {
	bundle := make(map[string]WorkItem, len(items))
	for _, item := range items {
		bundle[item.ID] = item
	}
	for _, item := range items {
		for _, dependencyID := range item.DependsOn {
			if dependencyID == item.ID {
				return conflict("work item %q cannot depend on itself", item.ID)
			}
			if _, inBundle := bundle[dependencyID]; inBundle {
				continue
			}
			dependency, exists := st.WorkItems[dependencyID]
			if !exists || dependency.ProjectID != projectID {
				return entityNotFound("work_item dependency", dependencyID)
			}
		}
	}
	visiting := make(map[string]bool, len(bundle))
	visited := make(map[string]bool, len(bundle))
	var visit func(string) error
	visit = func(id string) error {
		if visiting[id] {
			return conflict("work item dependency cycle includes %q", id)
		}
		if visited[id] {
			return nil
		}
		visiting[id] = true
		for _, dependencyID := range bundle[id].DependsOn {
			if _, inBundle := bundle[dependencyID]; inBundle {
				if err := visit(dependencyID); err != nil {
					return err
				}
			}
		}
		visiting[id] = false
		visited[id] = true
		return nil
	}
	for id := range bundle {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

func cleanDeliveryTextList(
	values []string,
	field string,
	maximum int,
	required bool,
) ([]string, error) {
	result := cleanLooseTextList(values)
	if required && len(result) == 0 {
		return nil, fmt.Errorf("%s requires at least one value", field)
	}
	if len(result) > maximum {
		return nil, fmt.Errorf("%s exceeds %d values", field, maximum)
	}
	for _, value := range result {
		if _, err := requireText(value, field, 4000); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func cleanLooseTextList(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func mergeDefectTransition(defect *Defect, input TransitionDefectPayload) {
	if input.Reproduction != "" {
		defect.Reproduction = strings.TrimSpace(input.Reproduction)
	}
	if input.Expected != "" {
		defect.Expected = strings.TrimSpace(input.Expected)
	}
	if input.Actual != "" {
		defect.Actual = strings.TrimSpace(input.Actual)
	}
	if input.Containment != "" {
		defect.Containment = strings.TrimSpace(input.Containment)
	}
	if input.RootCause != "" {
		defect.RootCause = strings.TrimSpace(input.RootCause)
	}
	if input.Resolution != "" {
		defect.Resolution = strings.TrimSpace(input.Resolution)
	}
	defect.WorkItemIDs = appendUnique(defect.WorkItemIDs, input.WorkItemIDs...)
	defect.EvidenceIDs = appendUnique(defect.EvidenceIDs, input.EvidenceIDs...)
}

func validDefectTransition(from, to DefectStatus) bool {
	allowed := map[DefectStatus][]DefectStatus{
		DefectReported:   {DefectConfirmed, DefectRejected},
		DefectConfirmed:  {DefectReproduced, DefectRejected},
		DefectReproduced: {DefectClassified},
		DefectClassified: {DefectFixing},
		DefectFixing:     {DefectVerifying},
		DefectVerifying:  {DefectVerified, DefectFixing},
		DefectVerified:   {DefectReleased, DefectReopened},
		DefectReleased:   {DefectClosed, DefectReopened},
		DefectClosed:     {DefectReopened},
		DefectReopened:   {DefectFixing},
	}
	return slices.Contains(allowed[from], to)
}

func validateDefectGate(defect Defect, target DefectStatus) error {
	switch target {
	case DefectReproduced:
		if defect.Reproduction == "" || defect.Expected == "" || defect.Actual == "" {
			return conflict("reproduced defect requires reproduction, expected, and actual evidence")
		}
	case DefectFixing:
		if len(defect.WorkItemIDs) == 0 {
			return conflict("fixing defect requires at least one work item")
		}
	case DefectVerifying:
		if defect.RootCause == "" || defect.Resolution == "" {
			return conflict("verifying defect requires root cause and resolution")
		}
	case DefectVerified:
		if len(defect.EvidenceIDs) == 0 {
			return conflict("verified defect requires verification evidence")
		}
	}
	return nil
}

func validRiskLevel(value string) bool {
	return value == "low" || value == "medium" || value == "high"
}

func validateRiskResponse(input DecideRiskResponsePayload, now time.Time) error {
	switch input.Response {
	case RiskAccept:
		if strings.TrimSpace(input.AcceptanceReason) == "" ||
			input.ReviewAt == nil || !input.ReviewAt.After(now) {
			return conflict("accepted risk requires a reason and future review_at")
		}
	case RiskAvoid, RiskMitigate:
		if strings.TrimSpace(input.ResponsePlan) == "" || len(input.WorkItemIDs) == 0 {
			return conflict("%s risk response requires a plan and work items", input.Response)
		}
	case RiskTransfer:
		if strings.TrimSpace(input.ResponsePlan) == "" {
			return conflict("transferred risk requires a response plan")
		}
	case RiskMonitor:
		if input.ReviewAt == nil || !input.ReviewAt.After(now) {
			return conflict("monitored risk requires future review_at")
		}
	default:
		return fmt.Errorf("unsupported risk response %q", input.Response)
	}
	return nil
}

func validRiskTransition(from, to RiskStatus) bool {
	allowed := map[RiskStatus][]RiskStatus{
		RiskIdentified: {RiskAssessed},
		RiskMonitoring: {RiskReviewed},
		RiskMitigating: {RiskReviewed},
		RiskReviewed:   {RiskClosed, RiskMonitoring, RiskMitigating},
	}
	return slices.Contains(allowed[from], to)
}

func deliveryResourceExists(
	st *state,
	projection DeliveryProjection,
	resourceType ResourceType,
	resourceID string,
) bool {
	switch resourceType {
	case ResourceRequest:
		_, ok := projection.Requests[resourceID]
		return ok
	case ResourceIntentContract:
		_, ok := projection.Intents[resourceID]
		return ok
	case ResourceSolution:
		_, ok := projection.Solutions[resourceID]
		return ok
	case ResourceFrozenPlan:
		_, ok := projection.FrozenPlans[resourceID]
		return ok
	case ResourceChangeIntent:
		_, ok := projection.ChangeIntents[resourceID]
		return ok
	case ResourceDefect:
		_, ok := projection.Defects[resourceID]
		return ok
	case ResourceRisk:
		_, ok := projection.Risks[resourceID]
		return ok
	case ResourceWorkItem:
		item, ok := st.WorkItems[resourceID]
		return ok && item.ProjectID == projection.ProjectID
	default:
		return false
	}
}

func deliveryEventsByID(events []DeliveryEvent, ids []string) []DeliveryEvent {
	wanted := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		wanted[id] = struct{}{}
	}
	result := make([]DeliveryEvent, 0, len(ids))
	for _, event := range events {
		if _, ok := wanted[event.ID]; ok {
			result = append(result, event)
		}
	}
	return result
}

func appendUnique(existing []string, values ...string) []string {
	seen := make(map[string]struct{}, len(existing)+len(values))
	result := make([]string, 0, len(existing)+len(values))
	for _, value := range append(slices.Clone(existing), values...) {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
