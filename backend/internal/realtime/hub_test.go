package realtime

import (
	"sync"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type recordingWriter struct {
	mu     sync.Mutex
	frames [][]byte
	notify chan struct{}
	closed bool
}

func newRecordingWriter() *recordingWriter { return &recordingWriter{notify: make(chan struct{}, 32)} }
func (w *recordingWriter) WriteMessage(_ int, data []byte) error {
	w.mu.Lock()
	w.frames = append(w.frames, append([]byte(nil), data...))
	w.mu.Unlock()
	w.notify <- struct{}{}
	return nil
}
func (w *recordingWriter) SetWriteDeadline(time.Time) error { return nil }
func (w *recordingWriter) Close() error                     { w.closed = true; return nil }

func TestHubEvictsSaturatedClientWithoutBlockingHealthyOrderedDelivery(t *testing.T) {
	hub := NewHub(nil)
	slowWriter := newRecordingWriter()
	slow := newClient(slowWriter, 2)
	// Deliberately do not start the slow writer pump and saturate its queue.
	slow.outbound <- []byte(`{"prefill":1}`)
	slow.outbound <- []byte(`{"prefill":2}`)
	healthyWriter := newRecordingWriter()
	healthy := newClient(healthyWriter, 32)
	go healthy.writePump()
	t.Cleanup(healthy.close)
	hub.add("workspace-one", slow)
	hub.add("workspace-one", healthy)

	events := []string{"issue:created", "issue:updated", "issue_metadata:changed", "issue:deleted", "comment:created", "comment:updated", "comment:deleted", "comment:resolved", "comment:unresolved", "reaction:added", "reaction:removed", "issue_reaction:added", "issue_reaction:removed", "subscriber:added", "subscriber:removed", "activity:created"}
	started := time.Now()
	for _, eventType := range events {
		hub.Publish("workspace-one", eventType, map[string]any{"sequence": eventType}, "member-one", "member")
	}
	if time.Since(started) > 100*time.Millisecond {
		t.Fatal("publish blocked on saturated client")
	}
	for range len(events) {
		select {
		case <-healthyWriter.notify:
		case <-time.After(time.Second):
			t.Fatal("healthy delivery timeout")
		}
	}
	healthyWriter.mu.Lock()
	defer healthyWriter.mu.Unlock()
	for index, eventType := range events {
		if !containsFrame(healthyWriter.frames[index], `"type":"`+eventType+`"`) {
			t.Fatalf("frame[%d]=%s", index, healthyWriter.frames[index])
		}
	}
	hub.mu.RLock()
	_, slowPresent := hub.clients["workspace-one"][slow]
	hub.mu.RUnlock()
	if slowPresent || !slowWriter.closed {
		t.Fatalf("saturated client present=%v closed=%v", slowPresent, slowWriter.closed)
	}
}

func TestHubRejectsEventsOutsideCanonicalM1Set(t *testing.T) {
	hub := NewHub(nil)
	writer := newRecordingWriter()
	client := newClient(writer, 2)
	go client.writePump()
	t.Cleanup(client.close)
	hub.add("workspace-one", client)
	hub.Publish("workspace-one", "task:created", map[string]any{"unexpected": true}, "", "")
	select {
	case <-writer.notify:
		t.Fatal("unsupported event was published")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestHubProjectsKnowledgeByReviewPermissionAndWorkspace(t *testing.T) {
	hub := NewHub(nil, func(workspaceID, actorType, actorID, eventType string) bool {
		return workspaceID == "workspace-one" && actorType == "member" && actorID == "reviewer" && eventType == "knowledge:candidate_updated"
	})
	reviewerWriter, memberWriter, foreignWriter := newRecordingWriter(), newRecordingWriter(), newRecordingWriter()
	reviewer, member, foreign := newClient(reviewerWriter, 4), newClient(memberWriter, 4), newClient(foreignWriter, 4)
	reviewer.identity = contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-one", ActorType: "member", ActorID: "reviewer"}
	member.identity = contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-one", ActorType: "member", ActorID: "member"}
	foreign.identity = contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-two", ActorType: "member", ActorID: "reviewer"}
	for _, value := range []*client{reviewer, member, foreign} {
		go value.writePump()
		t.Cleanup(value.close)
	}
	hub.add("workspace-one", reviewer)
	hub.add("workspace-one", member)
	hub.add("workspace-two", foreign)

	hub.Publish("workspace-one", "knowledge:candidate_updated", map[string]any{"candidate": map[string]any{"id": "candidate-one", "reason": "private"}}, "proposer", "member")
	waitForFrame(t, reviewerWriter)
	assertNoFrame(t, memberWriter)
	assertNoFrame(t, foreignWriter)
	if !containsFrame(reviewerWriter.frames[0], `"candidate"`) || !containsFrame(reviewerWriter.frames[0], `"private"`) {
		t.Fatalf("reviewer frame = %s", reviewerWriter.frames[0])
	}

	hub.Publish("workspace-one", "knowledge:candidate_updated", map[string]any{"candidate": map[string]any{"id": "candidate-one", "reason": "private"}, "entry": map[string]any{"id": "knowledge-one", "title": "public"}}, "reviewer", "member")
	waitForFrame(t, reviewerWriter)
	waitForFrame(t, memberWriter)
	assertNoFrame(t, foreignWriter)
	if containsFrame(memberWriter.frames[0], `"candidate"`) || containsFrame(memberWriter.frames[0], `"private"`) || !containsFrame(memberWriter.frames[0], `"entry"`) || !containsFrame(memberWriter.frames[0], `"public"`) {
		t.Fatalf("member frame = %s", memberWriter.frames[0])
	}
}

func TestHubKnowledgeResolverDenialFailsClosed(t *testing.T) {
	hub := NewHub(nil, func(string, string, string, string) bool { return false })
	writer := newRecordingWriter()
	value := newClient(writer, 2)
	value.identity = contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-one", ActorType: "member", ActorID: "owner"}
	go value.writePump()
	t.Cleanup(value.close)
	hub.add("workspace-one", value)
	hub.Publish("workspace-one", "knowledge:candidate_updated", map[string]any{"candidate": map[string]any{"id": "candidate-one"}}, "proposer", "member")
	assertNoFrame(t, writer)
}

func waitForFrame(t *testing.T, writer *recordingWriter) {
	t.Helper()
	select {
	case <-writer.notify:
	case <-time.After(time.Second):
		t.Fatal("realtime delivery timeout")
	}
}

func assertNoFrame(t *testing.T, writer *recordingWriter) {
	t.Helper()
	select {
	case <-writer.notify:
		t.Fatal("unexpected realtime delivery")
	case <-time.After(50 * time.Millisecond):
	}
}

func containsFrame(frame []byte, part string) bool {
	for index := 0; index+len(part) <= len(frame); index++ {
		if string(frame[index:index+len(part)]) == part {
			return true
		}
	}
	return false
}

var _ frameWriter = (*recordingWriter)(nil)
