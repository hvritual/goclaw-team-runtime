package realtime

import (
	"sync"
	"testing"
	"time"
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

func containsFrame(frame []byte, part string) bool {
	for index := 0; index+len(part) <= len(frame); index++ {
		if string(frame[index:index+len(part)]) == part {
			return true
		}
	}
	return false
}

var _ frameWriter = (*recordingWriter)(nil)
