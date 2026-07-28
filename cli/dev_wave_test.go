package cli

import (
	"strings"
	"testing"

	dev "github.com/smallnest/goclaw/orchestratorlite"
)

func TestNormalizeTeamCreateRequestRequiresWaveStep(t *testing.T) {
	request := dev.CreateRequest{}
	err := normalizeTeamCreateRequest(&request, "")
	if err == nil || !strings.Contains(err.Error(), "--wave-step is required") {
		t.Fatalf("expected missing Wave step error, got %v", err)
	}
}

func TestNormalizeTeamCreateRequestSendsOnlyStepIntent(t *testing.T) {
	request := dev.CreateRequest{Wave: &dev.WaveBinding{
		WaveID:         "forged-wave",
		PlanRevision:   999,
		StepID:         "PILOT-W00-S01",
		PlanPath:       "/tmp/forged",
		RegistrySHA256: strings.Repeat("a", 64),
		PlanSHA256:     strings.Repeat("b", 64),
	}}
	if err := normalizeTeamCreateRequest(&request, " PILOT-W00-S02 "); err != nil {
		t.Fatal(err)
	}
	if request.Wave == nil || request.Wave.StepID != "PILOT-W00-S02" {
		t.Fatalf("unexpected Wave intent: %+v", request.Wave)
	}
	if request.Wave.WaveID != "" ||
		request.Wave.PlanRevision != 0 ||
		request.Wave.PlanPath != "" ||
		request.Wave.RegistrySHA256 != "" ||
		request.Wave.PlanSHA256 != "" {
		t.Fatalf("untrusted Wave authority survived normalization: %+v", request.Wave)
	}
}
