package sqlite

import "testing"

func TestKnowledgeTransitionMatrix(t *testing.T) {
	tests := []struct {
		name, status, action, next string
		target, terminal, valid    bool
	}{
		{name: "approve", status: "candidate", action: "approve", next: "in_review", valid: true},
		{name: "reject", status: "in_review", action: "reject", next: "rejected", valid: true},
		{name: "quarantine", status: "in_review", action: "quarantine", next: "quarantined", valid: true},
		{name: "return", status: "quarantined", action: "return", next: "in_review", valid: true},
		{name: "publish new", status: "in_review", action: "publish", next: "published", terminal: true, valid: true},
		{name: "supersede target", status: "in_review", action: "supersede", next: "published", target: true, terminal: true, valid: true},
		{name: "invalidate target", status: "in_review", action: "invalidate", next: "published", target: true, terminal: true, valid: true},
		{name: "new cannot supersede", status: "in_review", action: "supersede"},
		{name: "target cannot publish", status: "in_review", action: "publish", target: true},
		{name: "terminal published immutable", status: "published", action: "reject"},
		{name: "terminal rejected immutable", status: "rejected", action: "approve"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			next, terminal, valid := knowledgeTransition(test.status, test.action, test.target)
			if next != test.next || terminal != test.terminal || valid != test.valid {
				t.Fatalf("transition = %q/%v/%v", next, terminal, valid)
			}
		})
	}
}
