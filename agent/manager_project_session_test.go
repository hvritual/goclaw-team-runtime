package agent

import (
	"testing"
	"time"

	"github.com/smallnest/goclaw/bus"
	"github.com/smallnest/goclaw/session"
)

func TestConversationSessionKeySharesProjectAcrossChannels(t *testing.T) {
	at := time.Unix(1_700_000_000, 0)
	obsidian := &bus.InboundMessage{
		Channel: "gateway", ChatID: "user:alice", Timestamp: at,
	}
	feishu := &bus.InboundMessage{
		Channel: "feishu", AccountID: "main", ChatID: "oc_team", Timestamp: at,
	}
	first := conversationSessionKey(obsidian, "project-alpha", "inbox", true)
	second := conversationSessionKey(feishu, "project-alpha", "inbox", true)
	want := "project-v2.cHJvamVjdC1hbHBoYQ.aW5ib3g"
	canonical, _, err := session.ProjectConversationKey(
		"project-alpha",
		"inbox",
	)
	if err != nil {
		t.Fatal(err)
	}
	if canonical != want || first != canonical || second != first {
		t.Fatalf("project session keys differ: %q != %q", first, second)
	}
	if got := conversationSessionKey(feishu, "project-alpha", "release", true); got == first {
		t.Fatalf("different topic unexpectedly reused %q", got)
	}
}

func TestConversationSessionKeyAvoidsLegacyFilenameCollision(t *testing.T) {
	message := &bus.InboundMessage{Channel: "gateway"}
	first := conversationSessionKey(
		message,
		"alpha",
		"beta_gamma",
		true,
	)
	second := conversationSessionKey(
		message,
		"alpha_beta",
		"gamma",
		true,
	)
	if first == "" || second == "" || first == second {
		t.Fatalf("collision pair returned %q and %q", first, second)
	}
}

func TestConversationSessionKeyPreservesLegacyChannelBoundary(t *testing.T) {
	message := &bus.InboundMessage{
		Channel:   "feishu",
		AccountID: "main",
		ChatID:    "oc_team",
		Timestamp: time.Unix(1_700_000_000, 0),
	}
	if got := conversationSessionKey(message, "default", "inbox", false); got != "feishu:main:oc_team" {
		t.Fatalf("legacy session key = %q", got)
	}
	message.ChatID = ""
	if got := conversationSessionKey(message, "default", "inbox", false); got != "feishu:main:1700000000" {
		t.Fatalf("ephemeral session key = %q", got)
	}
}

func TestCloneMessageMetadata(t *testing.T) {
	input := map[string]interface{}{"project_id": "alpha"}
	output := cloneMessageMetadata(input)
	output["project_id"] = "beta"
	if input["project_id"] != "alpha" {
		t.Fatalf("metadata alias leaked into input: %#v", input)
	}
}
