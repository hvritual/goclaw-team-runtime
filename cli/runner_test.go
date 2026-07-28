package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smallnest/goclaw/workstation"
)

func TestParseRepositoryMappings(t *testing.T) {
	got, err := parseRepositoryMappings([]string{
		"iot=/src/iot",
		"console=/src/console",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got["iot"] != "/src/iot" || got["console"] != "/src/console" {
		t.Fatalf("unexpected mappings: %#v", got)
	}
	for _, values := range [][]string{
		nil,
		{"missing-separator"},
		{"iot=/one", "iot=/two"},
	} {
		if _, err := parseRepositoryMappings(values); err == nil {
			t.Fatalf("expected invalid mappings to fail: %#v", values)
		}
	}
}

func TestStageBinarySecretFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runner", "device.key")
	expected := []byte{0, 1, 2, 3, 255}
	cleanup, err := stageBinarySecretFile(path, expected)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != string(expected) {
		t.Fatalf("device key changed: %v", actual)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("device key permissions = %o, want 600", info.Mode().Perm())
	}
	cleanup()
}

func TestIsNoWorkError(t *testing.T) {
	if !isNoWorkError(errors.New("RPC error: no compatible queued task")) {
		t.Fatal("expected no-work error to be recognized")
	}
	if isNoWorkError(errors.New("forbidden")) {
		t.Fatal("unexpected no-work match")
	}
}

func TestPrintRunnerDoctorHumanAndJSON(t *testing.T) {
	report := workstation.RunnerDoctorReport{
		SchemaVersion: 1,
		Ready:         false,
		Runtime: workstation.RunnerRuntime{
			OS: "linux", Arch: "arm64", Substrate: "lima",
		},
		Checks: []workstation.RunnerDoctorCheck{
			{
				ID: "runtime", Status: workstation.DoctorPass,
				Summary: "Linux guest accepted",
			},
			{
				ID:      "verification-isolation",
				Status:  workstation.DoctorFail,
				Summary: "bwrap probe failed",
				Detail:  "user namespaces unavailable",
			},
		},
	}
	var human bytes.Buffer
	if err := printRunnerDoctor(&human, report, false); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"Runner doctor: BLOCKED (linux/arm64, lima)",
		"[PASS] runtime",
		"[FAIL] verification-isolation",
		"user namespaces unavailable",
	} {
		if !strings.Contains(human.String(), expected) {
			t.Fatalf("human report missing %q:\n%s", expected, human.String())
		}
	}

	var machine bytes.Buffer
	if err := printRunnerDoctor(&machine, report, true); err != nil {
		t.Fatal(err)
	}
	var decoded workstation.RunnerDoctorReport
	if err := json.Unmarshal(machine.Bytes(), &decoded); err != nil {
		t.Fatalf("JSON report: %v\n%s", err, machine.String())
	}
	if decoded.Ready || decoded.Runtime.Substrate != "lima" {
		t.Fatalf("decoded report = %#v", decoded)
	}
}
