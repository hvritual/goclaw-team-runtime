package protocol

// Event types published to workspace clients.
const (
	EventIssueCreated           = "issue:created"
	EventIssueUpdated           = "issue:updated"
	EventIssueDeleted           = "issue:deleted"
	EventIssueMetadataChanged   = "issue_metadata:changed"
	EventIssueReactionAdded     = "issue_reaction:added"
	EventIssueReactionRemoved   = "issue_reaction:removed"
	EventIssueLabelsChanged     = "issue_labels:changed"
	EventIssuePropertiesChanged = "issue_properties:changed"

	EventCommentCreated    = "comment:created"
	EventCommentUpdated    = "comment:updated"
	EventCommentDeleted    = "comment:deleted"
	EventCommentResolved   = "comment:resolved"
	EventCommentUnresolved = "comment:unresolved"
	EventReactionAdded     = "reaction:added"
	EventReactionRemoved   = "reaction:removed"

	EventWorkspaceUpdated = "workspace:updated"
	EventWorkspaceDeleted = "workspace:deleted"
	EventMemberAdded       = "member:added"
	EventMemberUpdated     = "member:updated"
	EventMemberRemoved     = "member:removed"

	EventSubscriberAdded   = "subscriber:added"
	EventSubscriberRemoved = "subscriber:removed"

	EventSkillCreated = "skill:created"
	EventSkillUpdated = "skill:updated"
	EventSkillDeleted = "skill:deleted"

	EventTaskCreated = "task:created"
	EventTaskUpdated = "task:updated"
	EventTaskDeleted = "task:deleted"

	EventProjectCreated = "project:created"
	EventProjectUpdated = "project:updated"
	EventProjectDeleted = "project:deleted"

	EventLabelCreated    = "label:created"
	EventLabelUpdated    = "label:updated"
	EventLabelDeleted    = "label:deleted"
	EventPropertyCreated = "property:created"
	EventPropertyUpdated = "property:updated"
	EventPinCreated      = "pin:created"
	EventPinDeleted      = "pin:deleted"
	EventPinReordered    = "pin:reordered"

	EventInvitationCreated  = "invitation:created"
	EventInvitationAccepted = "invitation:accepted"
	EventInvitationDeclined = "invitation:declined"
	EventInvitationRevoked  = "invitation:revoked"
)
