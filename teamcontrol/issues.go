package teamcontrol

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

func (s *Service) CreateIssue(actorID string, input CreateIssueInput) (Issue, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return Issue{}, err
	}
	projectID, err := requireID(input.ProjectID, "project_id")
	if err != nil {
		return Issue{}, err
	}
	id, err := normalizeID(input.ID, "issue")
	if err != nil {
		return Issue{}, err
	}
	if !validIssueType(input.Type) {
		return Issue{}, fmt.Errorf("unsupported issue type %q", input.Type)
	}
	if !validSeverity(input.Severity) {
		return Issue{}, fmt.Errorf("unsupported issue severity %q", input.Severity)
	}
	priority := input.Priority
	if priority == "" {
		switch input.Severity {
		case SeverityCritical:
			priority = PriorityP0
		case SeverityHigh:
			priority = PriorityP1
		case SeverityMedium:
			priority = PriorityP2
		default:
			priority = PriorityP3
		}
	}
	if !validPriority(priority) {
		return Issue{}, fmt.Errorf("unsupported issue priority %q", priority)
	}
	title, err := requireText(input.Title, "title", 300)
	if err != nil {
		return Issue{}, err
	}
	description, err := optionalText(input.Description, "description", 100_000)
	if err != nil {
		return Issue{}, err
	}
	reproduction, err := optionalText(input.Reproduction, "reproduction", 50_000)
	if err != nil {
		return Issue{}, err
	}
	expected, err := optionalText(input.Expected, "expected", 50_000)
	if err != nil {
		return Issue{}, err
	}
	actual, err := optionalText(input.Actual, "actual", 50_000)
	if err != nil {
		return Issue{}, err
	}
	externalID, err := optionalText(input.ExternalIssueID, "external_issue_id", 300)
	if err != nil {
		return Issue{}, err
	}
	componentIDs, err := uniqueIDs(input.ComponentIDs, "component_id")
	if err != nil {
		return Issue{}, err
	}
	module, err := optionalText(input.Module, "module", 300)
	if err != nil {
		return Issue{}, err
	}
	environment, err := optionalText(input.Environment, "environment", 1000)
	if err != nil {
		return Issue{}, err
	}
	labels, err := cleanLabels(input.Labels)
	if err != nil {
		return Issue{}, err
	}
	var dueAt *time.Time
	if input.DueAt != nil {
		value := input.DueAt.UTC()
		dueAt = &value
	}
	if input.SLAMinutes < 0 || input.SLAMinutes > 525_600 {
		return Issue{}, fmt.Errorf("sla_minutes must be between 0 and 525600")
	}
	duplicateOf := strings.TrimSpace(input.DuplicateOf)
	if duplicateOf != "" {
		if duplicateOf, err = requireID(duplicateOf, "duplicate_of"); err != nil {
			return Issue{}, err
		}
	}
	regressionOf := strings.TrimSpace(input.RegressionOf)
	if regressionOf != "" {
		if regressionOf, err = requireID(regressionOf, "regression_of"); err != nil {
			return Issue{}, err
		}
	}
	if duplicateOf == id || regressionOf == id {
		return Issue{}, fmt.Errorf("issue cannot reference itself")
	}
	introducedByCommit, err := validateGitCommit(
		input.IntroducedByCommit,
		"introduced_by_commit",
	)
	if err != nil {
		return Issue{}, err
	}
	fixedByCommit, err := validateGitCommit(input.FixedByCommit, "fixed_by_commit")
	if err != nil {
		return Issue{}, err
	}
	releaseID := strings.TrimSpace(input.ReleaseID)
	if releaseID != "" {
		if releaseID, err = requireID(releaseID, "release_id"); err != nil {
			return Issue{}, err
		}
	}
	var created Issue
	err = s.store.update(func(st *state) error {
		if err := authorizeProject(st, actorID, projectID, ActionIssueWrite); err != nil {
			return err
		}
		if _, exists := st.Issues[id]; exists {
			return conflict("issue %q already exists", id)
		}
		if err := requireProjectComponents(st, projectID, componentIDs); err != nil {
			return err
		}
		for label, referencedID := range map[string]string{
			"duplicate_of":  duplicateOf,
			"regression_of": regressionOf,
		} {
			if referencedID == "" {
				continue
			}
			referenced, ok := st.Issues[referencedID]
			if !ok || referenced.ProjectID != projectID {
				return entityNotFound(label+" issue", referencedID)
			}
		}
		if releaseID != "" {
			if _, err := resourceProject(st, ResourceRelease, releaseID); err != nil {
				return fmt.Errorf("release_id: %w", err)
			}
			if project, _ := resourceProject(st, ResourceRelease, releaseID); project != projectID {
				return fmt.Errorf("%w: release belongs to another project", ErrForbidden)
			}
		}
		for _, issue := range st.Issues {
			if externalID != "" && issue.ProjectID == projectID &&
				issue.ExternalIssueID == externalID {
				return conflict("external issue %q is already registered", externalID)
			}
		}
		now := time.Now().UTC()
		var slaDeadline *time.Time
		if input.SLAMinutes > 0 {
			value := now.Add(time.Duration(input.SLAMinutes) * time.Minute)
			slaDeadline = &value
		}
		created = Issue{
			ID:                 id,
			ProjectID:          projectID,
			Type:               input.Type,
			Title:              title,
			Description:        description,
			Severity:           input.Severity,
			Priority:           priority,
			Status:             IssueNew,
			ReporterID:         actorID,
			Module:             module,
			Environment:        environment,
			Labels:             labels,
			ComponentIDs:       componentIDs,
			Reproduction:       reproduction,
			Expected:           expected,
			Actual:             actual,
			ExternalIssueID:    externalID,
			DueAt:              dueAt,
			SLAMinutes:         input.SLAMinutes,
			SLADeadline:        slaDeadline,
			DuplicateOf:        duplicateOf,
			RegressionOf:       regressionOf,
			IntroducedByCommit: introducedByCommit,
			FixedByCommit:      fixedByCommit,
			ReleaseID:          releaseID,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		st.Issues[id] = created
		return nil
	})
	return created, err
}

func (s *Service) GetIssue(userID, projectID, issueID string) (Issue, error) {
	issueID, err := requireID(issueID, "issue_id")
	if err != nil {
		return Issue{}, err
	}
	var result Issue
	err = s.readProject(userID, projectID, ActionIssueRead, func(st state, _ Project) error {
		issue, ok := st.Issues[issueID]
		if !ok || issue.ProjectID != projectID {
			return entityNotFound("issue", issueID)
		}
		result = issue
		return nil
	})
	return result, err
}

func (s *Service) ListIssues(userID, projectID string, statuses ...IssueStatus) ([]Issue, error) {
	allowed := make(map[IssueStatus]struct{}, len(statuses))
	for _, status := range statuses {
		allowed[status] = struct{}{}
	}
	var result []Issue
	err := s.readProject(userID, projectID, ActionIssueRead, func(st state, _ Project) error {
		for _, issue := range st.Issues {
			if issue.ProjectID != projectID {
				continue
			}
			if len(allowed) > 0 {
				if _, ok := allowed[issue.Status]; !ok {
					continue
				}
			}
			result = append(result, issue)
		}
		slices.SortFunc(result, func(a, b Issue) int {
			if a.UpdatedAt.Equal(b.UpdatedAt) {
				return strings.Compare(a.ID, b.ID)
			}
			if a.UpdatedAt.After(b.UpdatedAt) {
				return -1
			}
			return 1
		})
		return nil
	})
	return result, err
}

func (s *Service) TransitionIssue(
	actorID, projectID, issueID string,
	next IssueStatus,
	resolution string,
) (Issue, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return Issue{}, err
	}
	projectID, err = requireID(projectID, "project_id")
	if err != nil {
		return Issue{}, err
	}
	issueID, err = requireID(issueID, "issue_id")
	if err != nil {
		return Issue{}, err
	}
	resolution, err = optionalText(resolution, "resolution", 50_000)
	if err != nil {
		return Issue{}, err
	}
	var updated Issue
	err = s.store.update(func(st *state) error {
		issue, ok := st.Issues[issueID]
		if !ok || issue.ProjectID != projectID {
			return entityNotFound("issue", issueID)
		}
		if err := authorizeIssueTransition(
			st,
			actorID,
			projectID,
			issue,
		); err != nil {
			return err
		}
		if !canTransitionIssue(issue.Status, next) {
			return fmt.Errorf(
				"%w: issue %q cannot move from %q to %q",
				ErrInvalidTransition,
				issueID,
				issue.Status,
				next,
			)
		}
		if (next == IssueResolved || next == IssueClosed) &&
			resolution == "" && issue.Resolution == "" && !hasIssueFixEvidence(st, issue) {
			return fmt.Errorf(
				"resolution or linked fix evidence is required when resolving or closing an issue",
			)
		}
		issue.Status = next
		if next == IssueResolved || next == IssueClosed {
			if resolution != "" {
				issue.Resolution = resolution
			}
		}
		if next == IssueReopened {
			issue.Resolution = ""
			issue.ReopenCount++
		}
		issue.UpdatedAt = time.Now().UTC()
		st.Issues[issueID] = issue
		updated = issue
		return nil
	})
	return updated, err
}

func hasIssueFixEvidence(st *state, issue Issue) bool {
	if issue.FixedByCommit != "" || issue.ReleaseID != "" {
		return true
	}
	fixTypes := map[ResourceType]bool{
		ResourceCommit:      true,
		ResourcePullRequest: true,
		ResourceCI:          true,
		ResourceRelease:     true,
		ResourceArtifact:    true,
	}
	for _, link := range st.Links {
		if link.ProjectID != issue.ProjectID {
			continue
		}
		if link.SourceType == ResourceIssue && link.SourceID == issue.ID &&
			fixTypes[link.TargetType] {
			return true
		}
		if link.TargetType == ResourceIssue && link.TargetID == issue.ID &&
			fixTypes[link.SourceType] {
			return true
		}
	}
	return false
}

func canTransitionIssue(current, next IssueStatus) bool {
	allowed := map[IssueStatus][]IssueStatus{
		IssueNew:        {IssueTriaged, IssueCancelled},
		IssueTriaged:    {IssueAssigned, IssueInProgress, IssueBlocked, IssueCancelled},
		IssueAssigned:   {IssueInProgress, IssueBlocked, IssueCancelled},
		IssueInProgress: {IssueVerifying, IssueBlocked, IssueCancelled},
		IssueBlocked:    {IssueTriaged, IssueAssigned, IssueInProgress, IssueCancelled},
		IssueVerifying:  {IssueResolved, IssueInProgress, IssueBlocked},
		IssueResolved:   {IssueClosed, IssueReopened},
		IssueClosed:     {IssueReopened},
		IssueReopened:   {IssueTriaged, IssueAssigned, IssueInProgress, IssueCancelled},
	}
	return slices.Contains(allowed[current], next)
}

func authorizeIssueTransition(
	st *state,
	actorID, projectID string,
	issue Issue,
) error {
	if authorizeProject(
		st,
		actorID,
		projectID,
		ActionProjectManage,
	) == nil {
		return nil
	}
	if err := authorizeProject(
		st,
		actorID,
		projectID,
		ActionIssueTransition,
	); err != nil {
		return err
	}
	hasOwner := false
	for _, assignment := range st.Assignments {
		if assignment.ProjectID != projectID ||
			assignment.TargetType != AssignmentIssue ||
			assignment.TargetID != issue.ID ||
			assignment.Role != AssignmentOwner ||
			assignment.Status != AssignmentActive {
			continue
		}
		hasOwner = true
		if assignment.UserID == actorID {
			return nil
		}
	}
	if hasOwner {
		return fmt.Errorf(
			"%w: only the active issue owner or a project manager may transition issue %q",
			ErrForbidden,
			issue.ID,
		)
	}
	return nil
}

func (s *Service) CreateWorkItem(actorID string, input CreateWorkItemInput) (WorkItem, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return WorkItem{}, err
	}
	projectID, err := requireID(input.ProjectID, "project_id")
	if err != nil {
		return WorkItem{}, err
	}
	id, err := normalizeID(input.ID, "work")
	if err != nil {
		return WorkItem{}, err
	}
	title, err := requireText(input.Title, "title", 300)
	if err != nil {
		return WorkItem{}, err
	}
	instructions, err := requireText(input.Instructions, "instructions", 100_000)
	if err != nil {
		return WorkItem{}, err
	}
	businessDomain, err := normalizeBusinessDomain(input.BusinessDomain)
	if err != nil {
		return WorkItem{}, err
	}
	priority := input.Priority
	if priority == "" {
		priority = PriorityP2
	}
	if !validPriority(priority) {
		return WorkItem{}, fmt.Errorf("unsupported work item priority %q", priority)
	}
	if input.EstimatePoints < 0 || input.EstimatePoints > 10_000 {
		return WorkItem{}, fmt.Errorf(
			"estimate_points must be between 0 and 10000",
		)
	}
	var dueAt *time.Time
	if input.DueAt != nil {
		value := input.DueAt.UTC()
		dueAt = &value
	}
	issueID := strings.TrimSpace(input.IssueID)
	if issueID != "" {
		if issueID, err = requireID(issueID, "issue_id"); err != nil {
			return WorkItem{}, err
		}
	}
	dependsOn, err := uniqueIDs(input.DependsOn, "depends_on")
	if err != nil {
		return WorkItem{}, err
	}
	if slices.Contains(dependsOn, id) {
		return WorkItem{}, fmt.Errorf("work item cannot depend on itself")
	}
	componentIDs, err := uniqueIDs(input.ComponentIDs, "component_id")
	if err != nil {
		return WorkItem{}, err
	}
	commands, err := normalizeCommands(input.VerificationCommands)
	if err != nil {
		return WorkItem{}, err
	}
	var created WorkItem
	err = s.store.update(func(st *state) error {
		if err := authorizeProject(st, actorID, projectID, ActionWorkItemWrite); err != nil {
			return err
		}
		if _, exists := st.WorkItems[id]; exists {
			return conflict("work item %q already exists", id)
		}
		if issueID != "" {
			issue, ok := st.Issues[issueID]
			if !ok || issue.ProjectID != projectID {
				return entityNotFound("issue", issueID)
			}
		}
		for _, dependencyID := range dependsOn {
			dependency, ok := st.WorkItems[dependencyID]
			if !ok || dependency.ProjectID != projectID {
				return entityNotFound("work item dependency", dependencyID)
			}
		}
		if err := requireProjectComponents(st, projectID, componentIDs); err != nil {
			return err
		}
		now := time.Now().UTC()
		status := WorkItemReady
		if len(dependsOn) > 0 {
			status = WorkItemPending
		}
		created = WorkItem{
			ID:                   id,
			ProjectID:            projectID,
			IssueID:              issueID,
			Title:                title,
			Instructions:         instructions,
			BusinessDomain:       businessDomain,
			Priority:             priority,
			EstimatePoints:       input.EstimatePoints,
			DueAt:                dueAt,
			Status:               status,
			DependsOn:            dependsOn,
			ComponentIDs:         componentIDs,
			VerificationCommands: commands,
			CreatedBy:            actorID,
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		st.WorkItems[id] = created
		return nil
	})
	return created, err
}

func normalizeCommands(commands [][]string) ([][]string, error) {
	if len(commands) > 100 {
		return nil, fmt.Errorf("verification_commands exceeds 100 commands")
	}
	result := make([][]string, 0, len(commands))
	for index, command := range commands {
		if len(command) == 0 || strings.TrimSpace(command[0]) == "" {
			return nil, fmt.Errorf("verification command %d has no executable", index)
		}
		normalized := make([]string, len(command))
		for i, argument := range command {
			if strings.ContainsRune(argument, '\x00') {
				return nil, fmt.Errorf("verification command %d contains NUL", index)
			}
			normalized[i] = argument
		}
		result = append(result, normalized)
	}
	return result, nil
}

func (s *Service) GetWorkItem(userID, projectID, workItemID string) (WorkItem, error) {
	workItemID, err := requireID(workItemID, "work_item_id")
	if err != nil {
		return WorkItem{}, err
	}
	var result WorkItem
	err = s.readProject(userID, projectID, ActionWorkItemRead, func(st state, _ Project) error {
		workItem, ok := st.WorkItems[workItemID]
		if !ok || workItem.ProjectID != projectID {
			return entityNotFound("work item", workItemID)
		}
		result = workItem
		return nil
	})
	return result, err
}

func (s *Service) TransitionWorkItem(
	actorID, projectID, workItemID string,
	next WorkItemStatus,
) (WorkItem, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return WorkItem{}, err
	}
	projectID, err = requireID(projectID, "project_id")
	if err != nil {
		return WorkItem{}, err
	}
	workItemID, err = requireID(workItemID, "work_item_id")
	if err != nil {
		return WorkItem{}, err
	}
	var updated WorkItem
	err = s.store.update(func(st *state) error {
		item, ok := st.WorkItems[workItemID]
		if !ok || item.ProjectID != projectID {
			return entityNotFound("work item", workItemID)
		}
		if err := authorizeWorkItemTransition(
			st,
			actorID,
			projectID,
			item,
		); err != nil {
			return err
		}
		if !canTransitionWorkItem(item.Status, next) {
			return fmt.Errorf(
				"%w: work item %q cannot move from %q to %q",
				ErrInvalidTransition,
				workItemID,
				item.Status,
				next,
			)
		}
		if next == WorkItemReady || next == WorkItemInProgress {
			for _, dependencyID := range item.DependsOn {
				if st.WorkItems[dependencyID].Status != WorkItemDone {
					return conflict("dependency %q is not done", dependencyID)
				}
			}
		}
		item.Status = next
		item.UpdatedAt = time.Now().UTC()
		st.WorkItems[workItemID] = item
		updated = item
		return nil
	})
	return updated, err
}

func canTransitionWorkItem(current, next WorkItemStatus) bool {
	allowed := map[WorkItemStatus][]WorkItemStatus{
		WorkItemPending:    {WorkItemReady, WorkItemCancelled},
		WorkItemReady:      {WorkItemInProgress, WorkItemBlocked, WorkItemCancelled},
		WorkItemInProgress: {WorkItemVerifying, WorkItemBlocked, WorkItemCancelled},
		WorkItemBlocked:    {WorkItemReady, WorkItemInProgress, WorkItemCancelled},
		WorkItemVerifying:  {WorkItemDone, WorkItemInProgress, WorkItemBlocked},
	}
	return slices.Contains(allowed[current], next)
}

func authorizeWorkItemTransition(
	st *state,
	actorID, projectID string,
	item WorkItem,
) error {
	if authorizeProject(
		st,
		actorID,
		projectID,
		ActionProjectManage,
	) == nil {
		return nil
	}
	if err := authorizeProject(
		st,
		actorID,
		projectID,
		ActionWorkItemWrite,
	); err != nil {
		return err
	}
	hasOwner := false
	for _, assignment := range st.Assignments {
		if assignment.ProjectID != projectID ||
			assignment.TargetType != AssignmentWorkItem ||
			assignment.TargetID != item.ID ||
			assignment.Role != AssignmentOwner ||
			assignment.Status != AssignmentActive {
			continue
		}
		hasOwner = true
		if assignment.UserID == actorID {
			return nil
		}
	}
	if hasOwner || item.CreatedBy != actorID {
		return fmt.Errorf(
			"%w: only the active work item owner, unassigned creator, or a project manager may transition work item %q",
			ErrForbidden,
			item.ID,
		)
	}
	return nil
}

func (s *Service) Assign(actorID string, input AssignInput) (Assignment, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return Assignment{}, err
	}
	projectID, err := requireID(input.ProjectID, "project_id")
	if err != nil {
		return Assignment{}, err
	}
	targetID, err := requireID(input.TargetID, "target_id")
	if err != nil {
		return Assignment{}, err
	}
	userID, err := requireID(input.UserID, "user_id")
	if err != nil {
		return Assignment{}, err
	}
	id, err := normalizeID(input.ID, "assign")
	if err != nil {
		return Assignment{}, err
	}
	if input.TargetType != AssignmentIssue && input.TargetType != AssignmentWorkItem {
		return Assignment{}, fmt.Errorf("unsupported assignment target %q", input.TargetType)
	}
	switch input.Role {
	case AssignmentOwner, AssignmentContributor, AssignmentReviewer:
	default:
		return Assignment{}, fmt.Errorf("unsupported assignment role %q", input.Role)
	}
	var created Assignment
	err = s.store.update(func(st *state) error {
		action := ActionIssueAssign
		selfAction := ActionIssueWrite
		if input.TargetType == AssignmentWorkItem {
			action = ActionWorkItemAssign
			selfAction = ActionWorkItemWrite
		}
		if actorID == userID {
			action = selfAction
		}
		if err := authorizeProject(st, actorID, projectID, action); err != nil {
			return err
		}
		if _, exists := st.Assignments[id]; exists {
			return conflict("assignment %q already exists", id)
		}
		membership := findProjectMembership(st, projectID, userID)
		if membership == nil || membership.Status != MembershipActive {
			return fmt.Errorf("%w: assignee is not an active project member", ErrForbidden)
		}
		if err := requireActiveUser(st, userID); err != nil {
			return err
		}
		switch input.TargetType {
		case AssignmentIssue:
			issue, ok := st.Issues[targetID]
			if !ok || issue.ProjectID != projectID {
				return entityNotFound("issue", targetID)
			}
		case AssignmentWorkItem:
			item, ok := st.WorkItems[targetID]
			if !ok || item.ProjectID != projectID {
				return entityNotFound("work item", targetID)
			}
		}
		for _, assignment := range st.Assignments {
			if input.Role == AssignmentOwner &&
				assignment.ProjectID == projectID &&
				assignment.TargetType == input.TargetType &&
				assignment.TargetID == targetID &&
				assignment.Role == AssignmentOwner &&
				assignment.Status == AssignmentActive {
				return conflict(
					"target %q already has active owner %q",
					targetID,
					assignment.UserID,
				)
			}
			if assignment.ProjectID == projectID &&
				assignment.TargetType == input.TargetType &&
				assignment.TargetID == targetID &&
				assignment.UserID == userID &&
				assignment.Role == input.Role &&
				assignment.Status == AssignmentActive {
				return conflict("equivalent active assignment already exists")
			}
		}
		now := time.Now().UTC()
		created = Assignment{
			ID:         id,
			ProjectID:  projectID,
			TargetType: input.TargetType,
			TargetID:   targetID,
			UserID:     userID,
			Role:       input.Role,
			Status:     AssignmentActive,
			AssignedBy: actorID,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		st.Assignments[id] = created
		if input.TargetType == AssignmentIssue && input.Role == AssignmentOwner {
			issue := st.Issues[targetID]
			if issue.Status == IssueTriaged || issue.Status == IssueReopened {
				issue.Status = IssueAssigned
				issue.UpdatedAt = now
				st.Issues[targetID] = issue
			}
		}
		return nil
	})
	return created, err
}

func (s *Service) ReleaseAssignment(
	actorID, projectID, assignmentID string,
) (Assignment, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return Assignment{}, err
	}
	projectID, err = requireID(projectID, "project_id")
	if err != nil {
		return Assignment{}, err
	}
	assignmentID, err = requireID(assignmentID, "assignment_id")
	if err != nil {
		return Assignment{}, err
	}
	var updated Assignment
	err = s.store.update(func(st *state) error {
		assignment, ok := st.Assignments[assignmentID]
		if !ok || assignment.ProjectID != projectID {
			return entityNotFound("assignment", assignmentID)
		}
		action := ActionIssueAssign
		selfAction := ActionIssueWrite
		if assignment.TargetType == AssignmentWorkItem {
			action = ActionWorkItemAssign
			selfAction = ActionWorkItemWrite
		}
		if assignment.UserID == actorID {
			action = selfAction
		}
		if err := authorizeProject(st, actorID, projectID, action); err != nil {
			return err
		}
		if assignment.Status != AssignmentActive {
			return conflict("assignment %q is not active", assignmentID)
		}
		assignment.Status = AssignmentReleased
		assignment.UpdatedAt = time.Now().UTC()
		st.Assignments[assignmentID] = assignment
		updated = assignment
		return nil
	})
	return updated, err
}

func requireProjectComponents(st *state, projectID string, ids []string) error {
	for _, id := range ids {
		component, ok := st.Components[id]
		if !ok || component.ProjectID != projectID {
			return entityNotFound("component", id)
		}
	}
	return nil
}
