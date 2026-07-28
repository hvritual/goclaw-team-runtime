package ouroboros

import "testing"

func TestAmbiguityRequiresDimensionFloorsAndReadyStreak(t *testing.T) {
	input := clarityInput{
		Goal:       0.90,
		Constraint: 0.90,
		Success:    0.90,
		Context:    0.90,
	}
	first, err := calculateAmbiguity(
		input,
		false,
		0.20,
		1,
		0,
		2,
		"clear",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if first.Overall != 0.10 {
		t.Fatalf("expected ambiguity 0.10, got %.4f", first.Overall)
	}
	if first.Ready || first.ReadyStreak != 1 {
		t.Fatalf("first qualifying pass must not be ready: %#v", first)
	}

	second, err := calculateAmbiguity(
		input,
		false,
		0.20,
		2,
		first.ReadyStreak,
		2,
		"still clear",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Ready || second.ReadyStreak != 2 {
		t.Fatalf("second qualifying pass should be ready: %#v", second)
	}

	input.Success = 0.60
	failedFloor, err := calculateAmbiguity(
		input,
		false,
		0.30,
		3,
		second.ReadyStreak,
		2,
		"success remains vague",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if failedFloor.Ready || failedFloor.ReadyStreak != 0 {
		t.Fatalf("dimension floor must reset readiness: %#v", failedFloor)
	}
}

func TestBrownfieldRequiresContextClarity(t *testing.T) {
	assessment, err := calculateAmbiguity(
		clarityInput{
			Goal:       0.95,
			Constraint: 0.95,
			Success:    0.95,
			Context:    0.30,
		},
		true,
		0.20,
		1,
		0,
		1,
		"repository context is missing",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if assessment.Ready {
		t.Fatal("brownfield assessment must not pass below the context floor")
	}
}

func TestOntologySimilarityIsDeterministic(t *testing.T) {
	left := Ontology{
		Name: "Task",
		Fields: []OntologyField{
			{Name: "ID", Type: "string", Description: "stable identifier", Required: true},
		},
	}
	if got := ontologySimilarity(left, left); got != 1 {
		t.Fatalf("identical ontology should score 1, got %.4f", got)
	}
	right := Ontology{
		Name: "Task",
		Fields: []OntologyField{
			{Name: "ID", Type: "string", Description: "stable identifier", Required: true},
			{Name: "State", Type: "string", Description: "workflow state", Required: true},
		},
	}
	if got := ontologySimilarity(left, right); got != 0.75 {
		t.Fatalf("expected weighted similarity 0.75, got %.4f", got)
	}
}
