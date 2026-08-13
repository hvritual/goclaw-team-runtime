package contract

// WorkspaceEventPublisher publishes committed Workspace state to authorized
// realtime clients. Callers invoke it only after a successful repository write.
type WorkspaceEventPublisher interface {
	Publish(workspaceID, eventType string, payload any, actorID, actorType string)
}
