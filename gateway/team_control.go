package gateway

import (
	"fmt"
	"math"
	"strings"
	"time"

	dev "github.com/smallnest/goclaw/orchestratorlite"
	"github.com/smallnest/goclaw/teamcontrol"
	"github.com/smallnest/goclaw/workstation"
)

func (h *Handler) registerTeamControlMethods() {
	h.registry.Register("team.user.create", h.rpcCreateTeamUser)
	h.registry.Register("team.user.status", h.rpcSetTeamUserStatus)
	h.registry.Register("team.users", h.rpcListTeamUsers)
	h.registry.Register("team.token.issue", h.rpcIssueTeamToken)
	h.registry.Register("team.token.list", h.rpcListTeamTokens)
	h.registry.Register("team.token.revoke", h.rpcRevokeTeamToken)
	h.registry.Register("team.create", h.rpcCreateTeam)
	h.registry.Register("team.get", h.rpcGetTeam)
	h.registry.Register("team.member.add", h.rpcAddTeamMember)
	h.registry.Register("team.members", h.rpcTeamMembers)

	h.registry.Register("project.create", h.rpcCreateProject)
	h.registry.Register("project.get", h.rpcGetProject)
	h.registry.Register("project.list", h.rpcListProjects)
	h.registry.Register("project.member.add", h.rpcAddProjectMember)
	h.registry.Register("project.members", h.rpcProjectMembers)

	h.registry.Register("repository.create", h.rpcCreateRepository)
	h.registry.Register("repository.get", h.rpcGetRepository)
	h.registry.Register("repository.list", h.rpcListRepositories)

	h.registry.Register("issue.create", h.rpcCreateIssue)
	h.registry.Register("issue.get", h.rpcGetIssue)
	h.registry.Register("issue.list", h.rpcListIssues)
	h.registry.Register("issue.transition", h.rpcTransitionIssue)

	h.registry.Register("work.create", h.rpcCreateWorkItem)
	h.registry.Register("work.get", h.rpcGetWorkItem)
	h.registry.Register("work.items", h.rpcListWorkItems)
	h.registry.Register("work.transition", h.rpcTransitionWorkItem)
	h.registry.Register("assignment.create", h.rpcCreateAssignment)
	h.registry.Register("assignment.list", h.rpcListAssignments)
	h.registry.Register("assignment.release", h.rpcReleaseAssignment)

	h.registry.Register("artifact.register", h.rpcRegisterArtifact)
	h.registry.Register("artifact.list", h.rpcListArtifacts)
	h.registry.Register("correlation.create", h.rpcCreateCorrelation)
	h.registry.Register("correlation.list", h.rpcListCorrelations)

	h.registry.Register("document.register", h.rpcRegisterDocument)
	h.registry.Register("document.list", h.rpcListDocuments)
	h.registry.Register("docs.summary", h.rpcDocumentsSummary)
	h.registry.Register("component.register", h.rpcRegisterComponent)
	h.registry.Register("component.list", h.rpcListComponents)
	h.registry.Register("components.summary", h.rpcComponentsSummary)

	h.registry.Register("policy.put", h.rpcPutPolicy)
	h.registry.Register("policy.list", h.rpcListPolicies)
	h.registry.Register("policy.resolve", h.rpcResolvePolicy)
	h.registry.Register("policy.status", h.rpcPolicyStatus)

	h.registry.Register("budget.put", h.rpcPutTokenBudget)
	h.registry.Register("budget.list", h.rpcListTokenBudgets)
	h.registry.Register("budget.usage.record", h.rpcRecordTokenUsage)
	h.registry.Register("budget.usage.list", h.rpcListTokenUsage)
	h.registry.Register("knowledge.source.put", h.rpcPutKnowledgeSource)
	h.registry.Register("knowledge.source.get", h.rpcGetKnowledgeSource)
	h.registry.Register("knowledge.source.list", h.rpcListKnowledgeSources)
	h.registry.Register("knowledge.source.delete", h.rpcDeleteKnowledgeSource)
	h.registry.Register("skill.release.put", h.rpcPutSkillRelease)
	h.registry.Register("skill.release.get", h.rpcGetSkillRelease)
	h.registry.Register("skill.release.list", h.rpcListSkillReleases)
	h.registry.Register("skill.release.delete", h.rpcDeleteSkillRelease)
	h.registry.Register("runner.release.put", h.rpcPutRunnerRelease)
	h.registry.Register("runner.release.get", h.rpcGetRunnerRelease)
	h.registry.Register("runner.release.list", h.rpcListRunnerReleases)
	h.registry.Register("runner.release.delete", h.rpcDeleteRunnerRelease)
	h.registry.Register("context.compile", h.rpcCompileContext)
	h.registry.Register("context.list", h.rpcListContextBundles)
	h.registry.Register("control.summary", h.rpcControlSummary)
	h.registry.Register("delivery.command", h.rpcExecuteDeliveryCommand)
	h.registry.Register("delivery.projection", h.rpcDeliveryProjection)
	h.registry.Register("delivery.events", h.rpcDeliveryEvents)
	h.registry.Register("delivery.integrity", h.rpcDeliveryIntegrity)
}

func (h *Handler) rpcExecuteDeliveryCommand(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var command teamcontrol.DeliveryCommand
	if err := decodeDomainParams(params, &command); err != nil {
		return nil, err
	}
	command.ActorID = actorID
	return h.teamSvc.ExecuteDeliveryCommand(actorID, command)
}

func (h *Handler) rpcDeliveryProjection(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	return h.teamSvc.GetDeliveryProjection(
		actorID,
		stringParam(params["project_id"]),
	)
}

func (h *Handler) rpcDeliveryEvents(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	after := intParam(params["after_sequence"], 0)
	if after < 0 {
		after = 0
	}
	return h.teamSvc.ListDeliveryEvents(
		actorID,
		stringParam(params["project_id"]),
		uint64(after),
		intParam(params["limit"], 100),
	)
}

func (h *Handler) rpcDeliveryIntegrity(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	return h.teamSvc.VerifyDeliveryIntegrity(
		actorID,
		stringParam(params["project_id"]),
	)
}

func (h *Handler) rpcCreateTeamUser(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	teamID := stringParam(params["team_id"])
	if err := h.requireTeamAdmin(actorID, teamID); err != nil {
		return nil, err
	}
	var input teamcontrol.CreateUserInput
	if err := decodeDomainParams(params, &input); err != nil {
		return nil, err
	}
	return h.teamSvc.CreateUser(input)
}

func (h *Handler) rpcListTeamUsers(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	if err := h.requireTeamAdmin(actorID, stringParam(params["team_id"])); err != nil {
		return nil, err
	}
	return h.teamSvc.ListUsers(actorID)
}

func (h *Handler) rpcSetTeamUserStatus(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	if err := h.requireTeamAdmin(actorID, stringParam(params["team_id"])); err != nil {
		return nil, err
	}
	targetID := stringParam(params["user_id"])
	members, err := h.teamSvc.ListTeamMembers(
		actorID,
		stringParam(params["team_id"]),
	)
	if err != nil {
		return nil, err
	}
	targetIsMember := false
	actorRole := teamcontrol.TeamRole("")
	targetRole := teamcontrol.TeamRole("")
	for _, member := range members {
		if member.UserID == actorID && member.Status == teamcontrol.MembershipActive {
			actorRole = member.Role
		}
		if member.UserID == targetID &&
			member.Status != teamcontrol.MembershipRemoved {
			targetIsMember = true
			targetRole = member.Role
		}
	}
	if !targetIsMember {
		return nil, fmt.Errorf("%w: target user is not a team member", teamcontrol.ErrForbidden)
	}
	if err := h.teamSvc.AuthorizeUserAdministration(actorID, targetID); err != nil {
		return nil, err
	}
	status := teamcontrol.UserStatus(stringParam(params["status"]))
	if targetRole == teamcontrol.TeamOwner &&
		actorRole != teamcontrol.TeamOwner {
		return nil, fmt.Errorf("%w: only an owner may change an owner's status", teamcontrol.ErrForbidden)
	}
	if targetRole == teamcontrol.TeamOwner && status == teamcontrol.UserDisabled {
		activeOtherOwner := false
		for _, member := range members {
			if member.UserID == targetID ||
				member.Role != teamcontrol.TeamOwner ||
				member.Status != teamcontrol.MembershipActive {
				continue
			}
			user, getErr := h.teamSvc.GetUser(member.UserID)
			if getErr == nil && user.Status == teamcontrol.UserActive {
				activeOtherOwner = true
				break
			}
		}
		if !activeOtherOwner {
			return nil, fmt.Errorf("%w: cannot disable the last active team owner", teamcontrol.ErrConflict)
		}
	}
	return h.teamSvc.SetUserStatus(
		targetID,
		status,
	)
}

func (h *Handler) rpcIssueTeamToken(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var request struct {
		UserID    string     `json:"user_id"`
		Label     string     `json:"label"`
		Token     string     `json:"token"`
		ExpiresAt *time.Time `json:"expires_at"`
	}
	if err := decodeRPCParams(params, &request); err != nil {
		return nil, err
	}
	credential, err := h.teamSvc.RegisterAccessToken(
		actorID,
		request.UserID,
		request.Label,
		request.Token,
		request.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}
	return presentAccessCredential(credential), nil
}

func (h *Handler) rpcListTeamTokens(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	credentials, err := h.teamSvc.ListAccessCredentials(
		actorID,
		stringParam(params["user_id"]),
	)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(credentials))
	for _, credential := range credentials {
		result = append(result, presentAccessCredential(credential))
	}
	return result, nil
}

func (h *Handler) rpcRevokeTeamToken(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	credential, err := h.teamSvc.RevokeAccessToken(
		actorID,
		stringParam(params["credential_id"]),
	)
	if err != nil {
		return nil, err
	}
	return presentAccessCredential(credential), nil
}

func (h *Handler) rpcCreateTeam(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var input teamcontrol.CreateTeamInput
	if err := decodeDomainParams(params, &input); err != nil {
		return nil, err
	}
	return h.teamSvc.CreateTeam(actorID, input)
}

func (h *Handler) rpcGetTeam(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	return h.teamSvc.GetTeam(userID, stringParam(params["team_id"]))
}

func (h *Handler) rpcAddTeamMember(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var input teamcontrol.AddTeamMemberInput
	if err := decodeDomainParams(params, &input); err != nil {
		return nil, err
	}
	return h.teamSvc.AddTeamMember(actorID, stringParam(params["team_id"]), input)
}

func (h *Handler) rpcCreateProject(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var input teamcontrol.CreateProjectInput
	if err := decodeDomainParams(params, &input); err != nil {
		return nil, err
	}
	return h.teamSvc.CreateProject(actorID, input)
}

func (h *Handler) rpcGetProject(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	return h.teamSvc.GetProject(userID, stringParam(params["project_id"]))
}

func (h *Handler) rpcListProjects(
	sessionID string,
	_ map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	return h.teamSvc.ListProjects(userID)
}

func (h *Handler) rpcAddProjectMember(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var input teamcontrol.AddProjectMemberInput
	if err := decodeDomainParams(params, &input); err != nil {
		return nil, err
	}
	return h.teamSvc.AddProjectMember(
		actorID,
		stringParam(params["project_id"]),
		input,
	)
}

func (h *Handler) rpcProjectMembers(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	return h.teamSvc.ListProjectMembers(userID, stringParam(params["project_id"]))
}

func (h *Handler) rpcCreateRepository(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var input teamcontrol.CreateRepositoryInput
	if err := decodeDomainParams(params, &input); err != nil {
		return nil, err
	}
	return h.teamSvc.CreateRepository(actorID, input)
}

func (h *Handler) rpcGetRepository(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	return h.teamSvc.GetRepository(
		userID,
		stringParam(params["project_id"]),
		stringParam(params["repository_id"]),
	)
}

func (h *Handler) rpcListRepositories(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	return h.teamSvc.ListRepositories(userID, stringParam(params["project_id"]))
}

func (h *Handler) rpcCreateIssue(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var input teamcontrol.CreateIssueInput
	if err := decodeDomainParams(params, &input); err != nil {
		return nil, err
	}
	return h.teamSvc.CreateIssue(actorID, input)
}

func (h *Handler) rpcGetIssue(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	return h.teamSvc.GetIssue(
		userID,
		stringParam(params["project_id"]),
		stringParam(params["issue_id"]),
	)
}

func (h *Handler) rpcListIssues(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	statuses := make([]teamcontrol.IssueStatus, 0)
	for _, status := range stringSliceParam(params["statuses"]) {
		statuses = append(statuses, teamcontrol.IssueStatus(status))
	}
	issues, err := h.teamSvc.ListIssues(
		userID,
		stringParam(params["project_id"]),
		statuses...,
	)
	if err != nil {
		return nil, err
	}
	assignments, _ := h.teamSvc.ListAssignments(
		userID,
		stringParam(params["project_id"]),
		teamcontrol.AssignmentIssue,
		"",
	)
	owners := assignmentOwners(assignments)
	issueTasks, _, err := h.developmentTaskProjection(
		stringParam(params["project_id"]),
	)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(issues))
	for _, issue := range limitSlice(issues, intParam(params["limit"], 0)) {
		presented := presentIssue(issue, owners[issue.ID])
		presented["task_id"] = issueTasks[issue.ID]
		result = append(result, presented)
	}
	return result, nil
}

func (h *Handler) rpcTransitionIssue(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	projectID := stringParam(params["project_id"])
	issueID := stringParam(params["issue_id"])
	next := teamcontrol.IssueStatus(stringParam(params["status"]))
	if _, err := h.teamSvc.GetIssue(actorID, projectID, issueID); err != nil {
		return nil, err
	}
	if err := h.guardLinkedIssueTerminal(projectID, issueID, next); err != nil {
		return nil, err
	}
	return h.teamSvc.TransitionIssue(
		actorID,
		projectID,
		issueID,
		next,
		stringParam(params["resolution"]),
	)
}

func (h *Handler) rpcCreateWorkItem(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var input teamcontrol.CreateWorkItemInput
	if err := decodeDomainParams(params, &input); err != nil {
		return nil, err
	}
	return h.teamSvc.CreateWorkItem(actorID, input)
}

func (h *Handler) rpcGetWorkItem(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	return h.teamSvc.GetWorkItem(
		userID,
		stringParam(params["project_id"]),
		stringParam(params["work_item_id"]),
	)
}

func (h *Handler) rpcListWorkItems(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	projectID := stringParam(params["project_id"])
	items, err := h.teamSvc.ListWorkItems(userID, projectID)
	if err != nil {
		return nil, err
	}
	assignments, _ := h.teamSvc.ListAssignments(
		userID,
		projectID,
		teamcontrol.AssignmentWorkItem,
		"",
	)
	owners := assignmentOwners(assignments)
	_, workItemTasks, err := h.developmentTaskProjection(projectID)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range limitSlice(items, intParam(params["limit"], 0)) {
		presented := presentWorkItem(item, owners[item.ID])
		presented["task_id"] = workItemTasks[item.ID]
		result = append(result, presented)
	}
	return result, nil
}

func (h *Handler) rpcTransitionWorkItem(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	projectID := stringParam(params["project_id"])
	workItemID := stringParam(params["work_item_id"])
	next := teamcontrol.WorkItemStatus(stringParam(params["status"]))
	if _, err := h.teamSvc.GetWorkItem(
		actorID,
		projectID,
		workItemID,
	); err != nil {
		return nil, err
	}
	if err := h.guardLinkedWorkItemTerminal(
		projectID,
		workItemID,
		next,
	); err != nil {
		return nil, err
	}
	return h.teamSvc.TransitionWorkItem(
		actorID,
		projectID,
		workItemID,
		next,
	)
}

func (h *Handler) rpcCreateAssignment(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var input teamcontrol.AssignInput
	if err := decodeDomainParams(params, &input); err != nil {
		return nil, err
	}
	return h.teamSvc.Assign(actorID, input)
}

func (h *Handler) rpcListAssignments(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	return h.teamSvc.ListAssignments(
		userID,
		stringParam(params["project_id"]),
		teamcontrol.AssignmentTarget(stringParam(params["target_type"])),
		stringParam(params["target_id"]),
	)
}

func (h *Handler) rpcReleaseAssignment(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	return h.teamSvc.ReleaseAssignment(
		actorID,
		stringParam(params["project_id"]),
		stringParam(params["assignment_id"]),
	)
}

func (h *Handler) rpcRegisterArtifact(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var input teamcontrol.RegisterArtifactInput
	if err := decodeDomainParams(params, &input); err != nil {
		return nil, err
	}
	return h.teamSvc.RegisterArtifact(actorID, input)
}

func (h *Handler) rpcListArtifacts(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	return h.teamSvc.ListArtifacts(userID, stringParam(params["project_id"]))
}

func (h *Handler) rpcCreateCorrelation(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var input teamcontrol.CreateLinkInput
	if err := decodeDomainParams(params, &input); err != nil {
		return nil, err
	}
	return h.teamSvc.CreateLink(actorID, input)
}

func (h *Handler) rpcListCorrelations(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	return h.teamSvc.ListLinks(
		userID,
		stringParam(params["project_id"]),
		teamcontrol.ResourceType(stringParam(params["resource_type"])),
		stringParam(params["resource_id"]),
	)
}

func (h *Handler) rpcRegisterDocument(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var input teamcontrol.RegisterDocumentInput
	if err := decodeDomainParams(params, &input); err != nil {
		return nil, err
	}
	return h.teamSvc.RegisterDocument(actorID, input)
}

func (h *Handler) rpcListDocuments(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	return h.teamSvc.ListDocuments(userID, stringParam(params["project_id"]))
}

func (h *Handler) rpcRegisterComponent(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var input teamcontrol.RegisterComponentInput
	if err := decodeDomainParams(params, &input); err != nil {
		return nil, err
	}
	return h.teamSvc.RegisterComponent(actorID, input)
}

func (h *Handler) rpcListComponents(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	return h.teamSvc.ListComponents(userID, stringParam(params["project_id"]))
}

func (h *Handler) rpcPutPolicy(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var input teamcontrol.PutPolicyBundleInput
	if err := decodeDomainParams(params, &input); err != nil {
		return nil, err
	}
	return h.teamSvc.PutPolicyBundle(actorID, input)
}

func (h *Handler) rpcListPolicies(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	return h.teamSvc.ListPolicyBundles(userID, stringParam(params["project_id"]))
}

func (h *Handler) rpcResolvePolicy(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	return h.teamSvc.ResolvePolicy(
		userID,
		stringParam(params["project_id"]),
		stringParam(params["repository_id"]),
		stringParam(params["component_id"]),
	)
}

func (h *Handler) rpcPutTokenBudget(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var input teamcontrol.PutTokenBudgetInput
	if err := decodeDomainParams(params, &input); err != nil {
		return nil, err
	}
	return h.teamSvc.PutTokenBudget(actorID, input)
}

func (h *Handler) rpcListTokenBudgets(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	return h.teamSvc.ListTokenBudgets(userID, stringParam(params["project_id"]))
}

func (h *Handler) rpcRecordTokenUsage(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var input teamcontrol.RecordTokenUsageInput
	if err := decodeDomainParams(params, &input); err != nil {
		return nil, err
	}
	return h.teamSvc.RecordTokenUsage(actorID, input)
}

func (h *Handler) rpcListTokenUsage(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	return h.teamSvc.ListTokenUsage(
		userID,
		stringParam(params["project_id"]),
		stringParam(params["budget_id"]),
	)
}

func (h *Handler) rpcPutKnowledgeSource(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var input teamcontrol.PutKnowledgeSourceInput
	if err := decodeDomainParams(params, &input); err != nil {
		return nil, err
	}
	value, err := h.teamSvc.PutKnowledgeSource(actorID, input)
	if err != nil {
		return nil, err
	}
	return presentKnowledgeSource(value), nil
}

func (h *Handler) rpcGetKnowledgeSource(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	value, err := h.teamSvc.GetKnowledgeSource(
		userID,
		stringParam(params["project_id"]),
		stringParam(params["knowledge_id"]),
	)
	if err != nil {
		return nil, err
	}
	return presentKnowledgeSource(value), nil
}

func (h *Handler) rpcListKnowledgeSources(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	values, err := h.teamSvc.ListKnowledgeSources(userID, stringParam(params["project_id"]))
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(values))
	for _, value := range values {
		result = append(result, presentKnowledgeSource(value))
	}
	return result, nil
}

func (h *Handler) rpcDeleteKnowledgeSource(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	id := stringParam(params["knowledge_id"])
	if err := h.teamSvc.DeleteKnowledgeSource(
		actorID, stringParam(params["project_id"]), id,
	); err != nil {
		return nil, err
	}
	return map[string]interface{}{"id": id, "deleted": true}, nil
}

func (h *Handler) rpcPutSkillRelease(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var input teamcontrol.PutSkillReleaseInput
	if err := decodeDomainParams(params, &input); err != nil {
		return nil, err
	}
	value, err := h.teamSvc.PutSkillRelease(actorID, input)
	if err != nil {
		return nil, err
	}
	return presentSkillRelease(value), nil
}

func (h *Handler) rpcGetSkillRelease(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	value, err := h.teamSvc.GetSkillRelease(
		userID,
		stringParam(params["project_id"]),
		stringParam(params["skill_id"]),
	)
	if err != nil {
		return nil, err
	}
	return presentSkillRelease(value), nil
}

func (h *Handler) rpcListSkillReleases(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	values, err := h.teamSvc.ListSkillReleases(userID, stringParam(params["project_id"]))
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(values))
	for _, value := range values {
		result = append(result, presentSkillRelease(value))
	}
	return result, nil
}

func (h *Handler) rpcDeleteSkillRelease(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	id := stringParam(params["skill_id"])
	if err := h.teamSvc.DeleteSkillRelease(
		actorID, stringParam(params["project_id"]), id,
	); err != nil {
		return nil, err
	}
	return map[string]interface{}{"id": id, "deleted": true}, nil
}

func (h *Handler) rpcPutRunnerRelease(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var input teamcontrol.PutRunnerReleaseInput
	if err := decodeDomainParams(params, &input); err != nil {
		return nil, err
	}
	value, err := h.teamSvc.PutRunnerRelease(actorID, input)
	if err != nil {
		return nil, err
	}
	return presentRunnerRelease(value), nil
}

func (h *Handler) rpcGetRunnerRelease(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	value, err := h.teamSvc.GetRunnerRelease(
		userID,
		stringParam(params["project_id"]),
		stringParam(params["runner_release_id"]),
	)
	if err != nil {
		return nil, err
	}
	return presentRunnerRelease(value), nil
}

func (h *Handler) rpcListRunnerReleases(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	values, err := h.teamSvc.ListRunnerReleases(userID, stringParam(params["project_id"]))
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(values))
	for _, value := range values {
		result = append(result, presentRunnerRelease(value))
	}
	return result, nil
}

func (h *Handler) rpcDeleteRunnerRelease(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	id := stringParam(params["runner_release_id"])
	if err := h.teamSvc.DeleteRunnerRelease(
		actorID, stringParam(params["project_id"]), id,
	); err != nil {
		return nil, err
	}
	return map[string]interface{}{"id": id, "deleted": true}, nil
}

func (h *Handler) rpcCompileContext(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	actorID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var input teamcontrol.CompileContextInput
	if err := decodeDomainParams(params, &input); err != nil {
		return nil, err
	}
	return h.teamSvc.CompileContext(actorID, input)
}

func (h *Handler) rpcListContextBundles(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	return h.teamSvc.ListContextBundles(userID, stringParam(params["project_id"]))
}

func (h *Handler) rpcControlSummary(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	projectID := stringParam(params["project_id"])
	budgets, err := h.teamSvc.ListTokenBudgets(userID, projectID)
	if err != nil {
		return nil, err
	}
	knowledge, err := h.teamSvc.ListKnowledgeSources(userID, projectID)
	if err != nil {
		return nil, err
	}
	skills, err := h.teamSvc.ListSkillReleases(userID, projectID)
	if err != nil {
		return nil, err
	}
	releases, err := h.teamSvc.ListRunnerReleases(userID, projectID)
	if err != nil {
		return nil, err
	}
	contexts, err := h.teamSvc.ListContextBundles(userID, projectID)
	if err != nil {
		return nil, err
	}
	var limit, used int64
	for _, budget := range budgets {
		if budget.LimitTokens > teamcontrol.MaxProjectTokenTotal-limit ||
			budget.UsedTokens > teamcontrol.MaxProjectTokenTotal-used {
			return nil, fmt.Errorf("budget summary exceeds JavaScript safe integer")
		}
		limit += budget.LimitTokens
		used += budget.UsedTokens
	}
	approvedKnowledge := 0
	for _, value := range knowledge {
		if value.Status == teamcontrol.RegistryApproved {
			approvedKnowledge++
		}
	}
	approvedSkills := 0
	for _, value := range skills {
		if value.Status == teamcontrol.RegistryApproved {
			approvedSkills++
		}
	}
	return map[string]interface{}{
		"project_id":           projectID,
		"budget_count":         len(budgets),
		"limit_tokens":         limit,
		"used_tokens":          used,
		"knowledge_count":      len(knowledge),
		"approved_knowledge":   approvedKnowledge,
		"skill_count":          len(skills),
		"approved_skills":      approvedSkills,
		"runner_release_count": len(releases),
		"context_bundle_count": len(contexts),
	}, nil
}

func (h *Handler) rpcTeamMembers(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	projectID := stringParam(params["project_id"])
	memberships, err := h.teamSvc.ListProjectMembers(userID, projectID)
	if err != nil {
		return nil, err
	}
	assignments, _ := h.teamSvc.ListAssignments(userID, projectID, "", "")
	workItems, _ := h.teamSvc.ListWorkItems(userID, projectID)
	runners := h.projectRunners(projectID)

	result := make([]map[string]interface{}, 0, len(memberships))
	for _, membership := range memberships {
		user, getErr := h.teamSvc.GetUser(membership.UserID)
		if getErr != nil {
			continue
		}
		active, queued, blocked := memberWorkload(membership.UserID, assignments, workItems)
		memberRunners := make([]string, 0)
		status := "offline"
		var lastSeen time.Time
		for _, runner := range runners {
			if runner.OwnerUserID != membership.UserID {
				continue
			}
			memberRunners = append(memberRunners, runner.ID)
			if runner.Status == workstation.RunnerOnline {
				status = "active"
			}
			if runner.LastHeartbeatAt.After(lastSeen) {
				lastSeen = runner.LastHeartbeatAt
			}
		}
		if user.Status == teamcontrol.UserDisabled {
			status = "disabled"
		}
		utilization := 0
		if membership.CapacityPoints > 0 {
			utilization = int(math.Round(
				float64(active) / float64(membership.CapacityPoints) * 100,
			))
		}
		item := map[string]interface{}{
			"id":               user.ID,
			"display_name":     user.DisplayName,
			"role":             membership.Role,
			"status":           status,
			"business_domains": membership.BusinessDomains,
			"project_ids":      []string{projectID},
			"capacity": map[string]interface{}{
				"planned_points":      membership.CapacityPoints,
				"active_work":         active,
				"queued_work":         queued,
				"blocked_work":        blocked,
				"utilization_percent": utilization,
			},
			"runner_ids": memberRunners,
		}
		if !lastSeen.IsZero() {
			item["last_seen_at"] = lastSeen
		}
		result = append(result, item)
	}
	return result, nil
}

func (h *Handler) rpcPolicyStatus(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	projectID := stringParam(params["project_id"])
	resolved, err := h.teamSvc.ResolvePolicy(
		userID,
		projectID,
		stringParam(params["repository_id"]),
		stringParam(params["component_id"]),
	)
	if err != nil {
		return nil, err
	}
	bundles, err := h.teamSvc.ListPolicyBundles(userID, projectID)
	if err != nil {
		return nil, err
	}
	layers := make([]map[string]interface{}, 0, len(bundles))
	for _, bundle := range bundles {
		layers = append(layers, map[string]interface{}{
			"scope":     presentPolicyScope(bundle.Scope),
			"id":        bundle.ID,
			"version":   fmt.Sprint(bundle.Version),
			"checksum":  bundle.Hash,
			"compliant": bundle.Enabled,
		})
	}
	return map[string]interface{}{
		"project_id":        projectID,
		"effective_version": resolved.Hash,
		"compliant":         true,
		"drift_count":       0,
		"checked_at":        time.Now().UTC(),
		"layers":            layers,
		"violations":        []interface{}{},
	}, nil
}

func (h *Handler) rpcDocumentsSummary(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	projectID := stringParam(params["project_id"])
	documents, err := h.teamSvc.ListDocuments(userID, projectID)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]interface{}, 0, len(documents))
	approved, stale := 0, 0
	for _, document := range documents {
		switch document.Status {
		case teamcontrol.DocumentActive:
			approved++
		case teamcontrol.DocumentSuperseded, teamcontrol.DocumentArchived:
			stale++
		}
	}
	for _, document := range limitSlice(documents, intParam(params["limit"], 0)) {
		status := "draft"
		switch document.Status {
		case teamcontrol.DocumentActive:
			status = "approved"
		case teamcontrol.DocumentSuperseded, teamcontrol.DocumentArchived:
			status = "stale"
		}
		items = append(items, map[string]interface{}{
			"id":         document.ID,
			"title":      document.Title,
			"path":       document.URI,
			"kind":       document.Kind,
			"owner_id":   document.OwnerID,
			"status":     status,
			"updated_at": document.UpdatedAt,
		})
	}
	return map[string]interface{}{
		"project_id": projectID,
		"total":      len(documents),
		"approved":   approved,
		"review_due": 0,
		"stale":      stale,
		"unlinked":   0,
		"items":      items,
	}, nil
}

func (h *Handler) rpcComponentsSummary(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	projectID := stringParam(params["project_id"])
	components, err := h.teamSvc.ListComponents(userID, projectID)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]interface{}, 0, len(components))
	for _, component := range limitSlice(components, intParam(params["limit"], 0)) {
		ownerID := ""
		if len(component.OwnerIDs) > 0 {
			ownerID = component.OwnerIDs[0]
		}
		items = append(items, map[string]interface{}{
			"id":         component.ID,
			"name":       component.Name,
			"kind":       component.Kind,
			"owner_id":   ownerID,
			"status":     "active",
			"updated_at": component.UpdatedAt,
		})
	}
	return map[string]interface{}{
		"project_id":     projectID,
		"total":          len(components),
		"reusable":       len(components),
		"deprecated":     0,
		"pending_review": 0,
		"items":          items,
	}, nil
}

func (h *Handler) requireTeamAdmin(actorID, teamID string) error {
	members, err := h.teamSvc.ListTeamMembers(actorID, teamID)
	if err != nil {
		return err
	}
	for _, member := range members {
		if member.UserID == actorID && member.Status == teamcontrol.MembershipActive &&
			(member.Role == teamcontrol.TeamOwner || member.Role == teamcontrol.TeamAdmin) {
			return nil
		}
	}
	return fmt.Errorf("%w: team administration is required", teamcontrol.ErrForbidden)
}

func decodeDomainParams(params map[string]interface{}, target interface{}) error {
	normalized := make(map[string]interface{}, len(params))
	for key, value := range params {
		normalized[snakeToPascal(key)] = value
	}
	return decodeRPCParams(normalized, target)
}

func snakeToPascal(value string) string {
	parts := strings.Split(value, "_")
	var result strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		result.WriteString(strings.ToUpper(part[:1]))
		result.WriteString(part[1:])
	}
	return result.String()
}

func intParam(value interface{}, fallback int) int {
	switch number := value.(type) {
	case float64:
		return int(number)
	case int:
		return number
	default:
		return fallback
	}
}

func limitSlice[T any](values []T, limit int) []T {
	if limit <= 0 || limit >= len(values) {
		return values
	}
	return values[:limit]
}

func assignmentOwners(assignments []teamcontrol.Assignment) map[string]string {
	result := make(map[string]string)
	for _, assignment := range assignments {
		if assignment.Status == teamcontrol.AssignmentActive &&
			assignment.Role == teamcontrol.AssignmentOwner {
			result[assignment.TargetID] = assignment.UserID
		}
	}
	return result
}

func presentIssue(issue teamcontrol.Issue, ownerID string) map[string]interface{} {
	status := string(issue.Status)
	return map[string]interface{}{
		"id":               issue.ID,
		"project_id":       issue.ProjectID,
		"title":            issue.Title,
		"status":           status,
		"lifecycle_status": status,
		"severity":         issue.Severity,
		"priority":         presentPriority(issue.Priority),
		"owner_id":         ownerID,
		"task_id":          "",
		"updated_at":       issue.UpdatedAt,
	}
}

func presentWorkItem(item teamcontrol.WorkItem, ownerID string) map[string]interface{} {
	status := string(item.Status)
	kind := "task"
	if item.IssueID != "" {
		kind = "bug"
	}
	return map[string]interface{}{
		"id":               item.ID,
		"project_id":       item.ProjectID,
		"title":            item.Title,
		"kind":             kind,
		"status":           status,
		"lifecycle_status": status,
		"priority":         presentPriority(item.Priority),
		"assignee_id":      ownerID,
		"business_domain":  item.BusinessDomain,
		"issue_id":         item.IssueID,
		"task_id":          "",
		"updated_at":       item.UpdatedAt,
	}
}

func (h *Handler) developmentTaskProjection(
	projectID string,
) (map[string]string, map[string]string, error) {
	issueTasks := make(map[string]string)
	workItemTasks := make(map[string]string)
	if h.devSvc == nil {
		return issueTasks, workItemTasks, nil
	}
	tasks, err := h.devSvc.ListTasks(projectID)
	if err != nil {
		return nil, nil, fmt.Errorf("list linked development tasks: %w", err)
	}
	// ListTasks is newest first. Keep the latest revision/task projection when
	// a shared Issue has more than one development task.
	for _, task := range tasks {
		for _, issueID := range task.IssueIDs {
			if _, exists := issueTasks[issueID]; !exists {
				issueTasks[issueID] = task.ID
			}
		}
		for _, workItemID := range developmentWorkItemIDs(task) {
			if _, exists := workItemTasks[workItemID]; !exists {
				workItemTasks[workItemID] = task.ID
			}
		}
	}
	return issueTasks, workItemTasks, nil
}

func (h *Handler) guardLinkedIssueTerminal(
	projectID string,
	issueID string,
	next teamcontrol.IssueStatus,
) error {
	var required dev.TaskStatus
	switch next {
	case teamcontrol.IssueResolved, teamcontrol.IssueClosed:
		required = dev.TaskDone
	case teamcontrol.IssueCancelled:
		required = dev.TaskCancelled
	default:
		return nil
	}
	return h.guardLinkedDevelopmentTasks(
		projectID,
		func(task dev.Task) bool {
			return containsDevelopmentID(task.IssueIDs, issueID)
		},
		required,
		"issue",
		issueID,
	)
}

func (h *Handler) guardLinkedWorkItemTerminal(
	projectID string,
	workItemID string,
	next teamcontrol.WorkItemStatus,
) error {
	var required dev.TaskStatus
	switch next {
	case teamcontrol.WorkItemDone:
		required = dev.TaskDone
	case teamcontrol.WorkItemCancelled:
		required = dev.TaskCancelled
	default:
		return nil
	}
	return h.guardLinkedDevelopmentTasks(
		projectID,
		func(task dev.Task) bool {
			return containsDevelopmentID(
				developmentWorkItemIDs(task),
				workItemID,
			)
		},
		required,
		"work item",
		workItemID,
	)
}

func (h *Handler) guardLinkedDevelopmentTasks(
	projectID string,
	linked func(dev.Task) bool,
	required dev.TaskStatus,
	resourceKind string,
	resourceID string,
) error {
	if h.devSvc == nil {
		return nil
	}
	tasks, err := h.devSvc.ListTasks(projectID)
	if err != nil {
		return fmt.Errorf("list linked development tasks: %w", err)
	}
	for _, task := range tasks {
		if !linked(task) {
			continue
		}
		if task.Status != required {
			return fmt.Errorf(
				"%s %q cannot enter a terminal state while linked development task %q is %q; required %q",
				resourceKind,
				resourceID,
				task.ID,
				task.Status,
				required,
			)
		}
	}
	return nil
}

func presentPriority(priority teamcontrol.IssuePriority) string {
	switch priority {
	case teamcontrol.PriorityP0:
		return "critical"
	case teamcontrol.PriorityP1:
		return "high"
	case teamcontrol.PriorityP2:
		return "medium"
	default:
		return "low"
	}
}

func presentAccessCredential(
	credential teamcontrol.AccessCredential,
) map[string]interface{} {
	result := map[string]interface{}{
		"id":         credential.ID,
		"user_id":    credential.UserID,
		"label":      credential.Label,
		"created_by": credential.CreatedBy,
		"created_at": credential.CreatedAt,
		"updated_at": credential.UpdatedAt,
	}
	if credential.ExpiresAt != nil {
		result["expires_at"] = credential.ExpiresAt
	}
	if credential.RevokedAt != nil {
		result["revoked_at"] = credential.RevokedAt
	}
	return result
}

func presentKnowledgeSource(value teamcontrol.KnowledgeSource) map[string]interface{} {
	return map[string]interface{}{
		"id": value.ID, "project_id": value.ProjectID, "name": value.Name,
		"uri": value.URI, "revision": value.Revision, "sha256": value.SHA256,
		"status": value.Status, "created_by": value.CreatedBy,
		"updated_by": value.UpdatedBy, "created_at": value.CreatedAt,
		"updated_at": value.UpdatedAt,
	}
}

func presentSkillRelease(value teamcontrol.SkillRelease) map[string]interface{} {
	return map[string]interface{}{
		"id": value.ID, "project_id": value.ProjectID, "name": value.Name,
		"version": value.Version, "uri": value.URI, "sha256": value.SHA256,
		"min_runner_version": value.MinRunnerVersion, "status": value.Status,
		"created_by": value.CreatedBy, "updated_by": value.UpdatedBy,
		"created_at": value.CreatedAt, "updated_at": value.UpdatedAt,
	}
}

func presentRunnerRelease(value teamcontrol.RunnerRelease) map[string]interface{} {
	return map[string]interface{}{
		"id": value.ID, "project_id": value.ProjectID, "channel": value.Channel,
		"version": value.Version, "os": value.OS, "arch": value.Arch,
		"uri": value.URI, "sha256": value.SHA256,
		"min_protocol": value.MinProtocol, "status": value.Status,
		"created_by": value.CreatedBy, "updated_by": value.UpdatedBy,
		"created_at": value.CreatedAt, "updated_at": value.UpdatedAt,
	}
}

func presentPolicyScope(scope teamcontrol.PolicyScope) string {
	switch scope {
	case teamcontrol.PolicyTeam:
		return "global"
	case teamcontrol.PolicyProject:
		return "project"
	default:
		return "domain"
	}
}

func memberWorkload(
	userID string,
	assignments []teamcontrol.Assignment,
	items []teamcontrol.WorkItem,
) (active, queued, blocked int) {
	owned := assignmentOwners(assignments)
	for _, item := range items {
		if owned[item.ID] != userID {
			continue
		}
		switch item.Status {
		case teamcontrol.WorkItemInProgress, teamcontrol.WorkItemVerifying:
			active += max(1, item.EstimatePoints)
		case teamcontrol.WorkItemBlocked:
			blocked += max(1, item.EstimatePoints)
		case teamcontrol.WorkItemPending, teamcontrol.WorkItemReady:
			queued += max(1, item.EstimatePoints)
		}
	}
	return active, queued, blocked
}
