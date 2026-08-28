package domain

import (
	"errors"
	"testing"
	"time"
)

func TestChangeSeparatesWorkRunAndAcceptedMutation(t *testing.T) {
	workItem := mustNode(t, NodeKindTodo, "todo-321")
	change, err := NewChange(
		"change-1", "ws-1", "project-1", "req-1", &workItem, "run-1001",
		"Use bounded exponential backoff",
		[]string{"service:session", "service:device-gateway", "service:device-gateway"},
		nil, mustProvenance(t), testNow,
	)
	if err != nil {
		t.Fatalf("NewChange() error = %v", err)
	}
	if change.Status() != ChangeStatusProposed {
		t.Fatalf("status = %q", change.Status())
	}
	if got := change.AffectedEntityIDs(); len(got) != 2 || got[0] != "service:device-gateway" || got[1] != "service:session" {
		t.Fatalf("affected entities = %#v", got)
	}
	accepted, err := change.Accept(testNow.Add(time.Minute))
	if err != nil {
		t.Fatalf("Accept() error = %v", err)
	}
	if accepted.Status() != ChangeStatusAccepted || accepted.AcceptedAt() == nil {
		t.Fatalf("accepted change = status %q, acceptedAt %v", accepted.Status(), accepted.AcceptedAt())
	}
	if _, err := accepted.Reject(testNow.Add(2 * time.Minute)); !errors.Is(err, ErrChangeTransitionInvalid) {
		t.Fatalf("reject accepted change error = %v", err)
	}
}

func TestChangeRejectsExecutionNodeAsWorkItem(t *testing.T) {
	run := mustNode(t, NodeKindRun, "run-1")
	if _, err := NewChange("change-1", "ws-1", "", "", &run, "run-1", "bad link", []string{"service:a"}, nil, mustProvenance(t), testNow); !errors.Is(err, ErrNodeKindInvalid) {
		t.Fatalf("error = %v, want %v", err, ErrNodeKindInvalid)
	}
}

func TestContextPackIsFrozenAndDeterministic(t *testing.T) {
	workItem := mustNode(t, NodeKindTodo, "todo-321")
	architecture, err := NewContextReference(ContextKindArchitecture, "ARCH-021", "r4", "sha256:arch")
	if err != nil {
		t.Fatalf("NewContextReference() error = %v", err)
	}
	standard, err := NewContextReference(ContextKindStandard, "STD-GO-001", "r6", "sha256:std")
	if err != nil {
		t.Fatalf("NewContextReference() error = %v", err)
	}
	first, err := NewContextPack(
		"cp-1", "ws-1", workItem, "r7",
		[]string{"service:session", "service:device-gateway"},
		[]ContextReference{standard, architecture}, "policy-v1", testNow,
	)
	if err != nil {
		t.Fatalf("NewContextPack() error = %v", err)
	}
	second, err := NewContextPack(
		"cp-2", "ws-1", workItem, "r7",
		[]string{"service:device-gateway", "service:session"},
		[]ContextReference{architecture, standard}, "policy-v1", testNow.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("NewContextPack() error = %v", err)
	}
	if first.Checksum() != second.Checksum() {
		t.Fatalf("checksums differ: %q != %q", first.Checksum(), second.Checksum())
	}
	if _, err := RehydrateContextPack("cp-1", "ws-1", workItem, "r7", first.TargetEntityIDs(), first.References(), "policy-v1", "wrong", testNow); !errors.Is(err, ErrContextPackChecksumMismatch) {
		t.Fatalf("rehydrate error = %v", err)
	}
}

func TestContextReferenceRequiresRevisionAndChecksum(t *testing.T) {
	if _, err := NewContextReference(ContextKindADR, "ADR-027", "", "sha256:x"); !errors.Is(err, ErrContextReferenceRevisionRequired) {
		t.Fatalf("revision error = %v", err)
	}
	if _, err := NewContextReference(ContextKindADR, "ADR-027", "r2", ""); !errors.Is(err, ErrContextReferenceChecksumRequired) {
		t.Fatalf("checksum error = %v", err)
	}
}
