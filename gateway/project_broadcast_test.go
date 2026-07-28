package gateway

import "testing"

func TestResolveProjectBroadcastScopeRejectsMissingAndConflictingScope(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name     string
		channel  string
		chatID   string
		metadata interface{}
		want     projectBroadcastScope
		ok       bool
	}{
		{
			name:    "gateway route is authoritative",
			channel: "gateway",
			chatID:  "project:alpha:topic:inbox",
			metadata: map[string]interface{}{
				"project_id": "alpha",
				"topic_id":   "inbox",
			},
			want: projectBroadcastScope{ProjectID: "alpha", TopicID: "inbox"},
			ok:   true,
		},
		{
			name:    "gateway route supplies omitted metadata",
			channel: "gateway",
			chatID:  "project:alpha:topic:inbox",
			want:    projectBroadcastScope{ProjectID: "alpha", TopicID: "inbox"},
			ok:      true,
		},
		{
			name:    "missing gateway topic",
			channel: "gateway",
			chatID:  "project:alpha",
		},
		{
			name:    "empty gateway topic",
			channel: "gateway",
			chatID:  "project:alpha:topic:",
		},
		{
			name:    "metadata cannot redirect gateway project",
			channel: "gateway",
			chatID:  "project:alpha:topic:inbox",
			metadata: map[string]interface{}{
				"project_id": "other",
				"topic_id":   "inbox",
			},
		},
		{
			name:    "metadata cannot redirect gateway topic",
			channel: "gateway",
			chatID:  "project:alpha:topic:inbox",
			metadata: map[string]interface{}{
				"project_id": "alpha",
				"topic_id":   "release",
			},
		},
		{
			name:    "external channel requires project",
			channel: "feishu",
			chatID:  "chat-1",
			metadata: map[string]interface{}{
				"topic_id": "inbox",
			},
		},
		{
			name:    "external channel requires topic",
			channel: "feishu",
			chatID:  "chat-1",
			metadata: map[string]interface{}{
				"project_id": "alpha",
			},
		},
		{
			name:    "external channel accepts complete metadata scope",
			channel: "feishu",
			chatID:  "chat-1",
			metadata: map[string]string{
				"project_id": "alpha",
				"topic_id":   "inbox",
			},
			want: projectBroadcastScope{ProjectID: "alpha", TopicID: "inbox"},
			ok:   true,
		},
		{
			name:    "invalid project segment",
			channel: "feishu",
			chatID:  "chat-1",
			metadata: map[string]interface{}{
				"project_id": "alpha:other",
				"topic_id":   "inbox",
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, ok := resolveProjectBroadcastScope(
				test.channel,
				test.chatID,
				test.metadata,
			)
			if ok != test.ok || got != test.want {
				t.Fatalf(
					"resolve scope = (%+v, %v), want (%+v, %v)",
					got,
					ok,
					test.want,
					test.ok,
				)
			}
		})
	}
}

func TestProjectBroadcastAllowedRequiresAuthenticatedAuthorizedPrincipal(
	t *testing.T,
) {
	fixture := newGatewayTeamFixture(t)
	scope := projectBroadcastScope{
		ProjectID: fixture.project.ID,
		TopicID:   "inbox",
	}

	if !projectBroadcastAllowed(
		&fixture.service,
		&Connection{PrincipalID: fixture.alice.ID},
		false,
		scope,
		true,
	) {
		t.Fatal("authorized project principal was denied")
	}
	if !projectBroadcastAllowed(
		&fixture.service,
		&Connection{PrincipalID: fixture.bob.ID},
		false,
		scope,
		true,
	) {
		t.Fatal("authorized project member was denied")
	}

	tests := []struct {
		name    string
		conn    *Connection
		scope   projectBroadcastScope
		scopeOK bool
	}{
		{
			name:    "nil connection",
			scope:   scope,
			scopeOK: true,
		},
		{
			name:    "missing authenticated principal",
			conn:    &Connection{},
			scope:   scope,
			scopeOK: true,
		},
		{
			name:    "principal belongs to another project",
			conn:    &Connection{PrincipalID: fixture.viewer.ID},
			scope:   scope,
			scopeOK: true,
		},
		{
			name:    "invalid scope cannot use legacy session match",
			conn:    &Connection{PrincipalID: fixture.alice.ID},
			scope:   scope,
			scopeOK: false,
		},
		{
			name: "missing project",
			conn: &Connection{PrincipalID: fixture.alice.ID},
			scope: projectBroadcastScope{
				TopicID: "inbox",
			},
			scopeOK: true,
		},
		{
			name: "missing topic",
			conn: &Connection{PrincipalID: fixture.alice.ID},
			scope: projectBroadcastScope{
				ProjectID: fixture.project.ID,
			},
			scopeOK: true,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if projectBroadcastAllowed(
				&fixture.service,
				test.conn,
				true,
				test.scope,
				test.scopeOK,
			) {
				t.Fatal("team project broadcast was allowed")
			}
		})
	}
}

func TestProjectBroadcastAllowedPreservesLegacySessionRouting(t *testing.T) {
	t.Parallel()
	conn := &Connection{}
	if !projectBroadcastAllowed(
		nil,
		conn,
		true,
		projectBroadcastScope{},
		false,
	) {
		t.Fatal("legacy matching session was denied")
	}
	if projectBroadcastAllowed(
		nil,
		conn,
		false,
		projectBroadcastScope{},
		false,
	) {
		t.Fatal("legacy non-matching session was allowed")
	}
}
