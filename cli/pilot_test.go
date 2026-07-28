package cli

import (
	"archive/tar"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestActivePilotWaveRequiresOneMatchingActiveEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wave-registry.json")
	writeTestFile(t, path, `{
	  "active_wave":"PILOT-W00",
	  "waves":[
	    {"id":"FE-W01","status":"blocked"},
	    {"id":"PILOT-W00","status":"active"}
	  ]
	}`, 0o600)
	id, err := activePilotWave(path)
	if err != nil {
		t.Fatal(err)
	}
	if id != "PILOT-W00" {
		t.Fatalf("active wave = %q", id)
	}

	writeTestFile(t, path, `{
	  "active_wave":"PILOT-W00",
	  "waves":[
	    {"id":"FE-W01","status":"active"},
	    {"id":"PILOT-W00","status":"active"}
	  ]
	}`, 0o600)
	if _, err := activePilotWave(path); err == nil {
		t.Fatal("expected two active Waves to fail")
	}
}

func TestCredentialAttestationFailsClosed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "attestation.json")
	writeTestFile(t, path, `{
	  "schema_version":"goclaw.credential-attestation/v1",
	  "issue_id":"FE-ISSUE-007",
	  "status":"rotated",
	  "attested_by":"credential-owner",
	  "attested_at":"2026-07-26T10:00:00Z",
	  "evidence_ref":"owner-ticket-42"
	}`, 0o600)
	if err := validatePilotAttestation(path); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, path, strings.Replace(
		string(mustReadTestFile(t, path)),
		`"rotated"`,
		`"pending"`,
		1,
	), 0o600)
	if err := validatePilotAttestation(path); err == nil {
		t.Fatal("expected a pending attestation to fail")
	}
}

func TestAgeIdentityMustBePrivateRegularFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pilot.agekey")
	writeTestFile(t, path, "AGE-SECRET-KEY-TEST\n", 0o600)
	if err := validateAgeIdentityPath(path); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := validateAgeIdentityPath(path); err == nil {
			t.Fatal("expected an overexposed identity to fail")
		}
	}
	if err := validateAgeIdentityPath(t.TempDir()); err == nil {
		t.Fatal("expected a directory identity to fail")
	}
}

func TestPilotManifestRoundTripAndTamperRejection(t *testing.T) {
	stage := t.TempDir()
	sources := prepareCompletePilotStage(t, stage)
	writeTestFile(t, filepath.Join(stage, "data", "workstation", "tasks", "task.txt"), "completed", 0o600)
	manifest, err := buildPilotManifest(stage, sources, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := writePilotManifest(stage, &manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := readAndVerifyPilotManifest(stage); err != nil {
		t.Fatal(err)
	}
	incomplete := manifest
	incomplete.Sources = append([]pilotBackupSource(nil), manifest.Sources...)
	incomplete.Sources = incomplete.Sources[:len(incomplete.Sources)-1]
	if err := validatePilotBackupSources(stage, incomplete); err == nil {
		t.Fatal("expected an incomplete recovery point to fail semantic validation")
	}

	writeTestFile(t, filepath.Join(stage, "data", "workstation", "tasks", "task.txt"), "tampered", 0o600)
	if _, err := readAndVerifyPilotManifest(stage); err == nil {
		t.Fatal("expected tampered backup member to fail")
	}
}

func TestPilotTarRejectsPathTraversalAndRestoresSafeArchive(t *testing.T) {
	stage := t.TempDir()
	sources := prepareCompletePilotStage(t, stage)
	manifest, err := buildPilotManifest(stage, sources, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := writePilotManifest(stage, &manifest); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "safe.tar")
	if err := writePilotTar(stage, archive); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "restore")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := extractPilotTar(archive, target); err != nil {
		t.Fatal(err)
	}
	if _, err := readAndVerifyPilotManifest(target); err != nil {
		t.Fatal(err)
	}

	unsafeArchive := filepath.Join(t.TempDir(), "unsafe.tar")
	file, err := os.OpenFile(unsafeArchive, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	writer := tar.NewWriter(file)
	if err := writer.WriteHeader(&tar.Header{
		Name: "../escape", Mode: 0o600, Size: 1, Typeflag: tar.TypeReg,
		ModTime: time.Unix(0, 0),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := extractPilotTar(unsafeArchive, t.TempDir()); err == nil {
		t.Fatal("expected traversal member to fail")
	}
}

func prepareCompletePilotStage(t *testing.T, stage string) []pilotSource {
	t.Helper()
	writeTestFile(t, filepath.Join(stage, "data", "config"), `{
	  "harness":{"enabled":false},
	  "ouroboros":{"enabled":false},
	  "memory":{"catalog":{"enabled":false}}
	}`, 0o600)
	writeTestFile(t, filepath.Join(stage, "data", "credential-attestation"), `{
	  "schema_version":"goclaw.credential-attestation/v1",
	  "issue_id":"FE-ISSUE-007",
	  "status":"rotated",
	  "attested_by":"credential-owner",
	  "attested_at":"2026-07-26T10:00:00Z"
	}`, 0o600)
	writeTestFile(
		t,
		filepath.Join(stage, "data", "teamcontrol", "teamcontrol.json"),
		`{"schema_version":1,"revision":7}`,
		0o600,
	)
	for _, name := range []string{"tasks", "runners", "credentials", "evidence"} {
		if err := os.MkdirAll(
			filepath.Join(stage, "data", "workstation", name),
			0o700,
		); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"development", "sessions", "workspace"} {
		if err := os.MkdirAll(filepath.Join(stage, "data", name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	return []pilotSource{
		{Name: "config"},
		{Name: "credential-attestation"},
		{Name: "teamcontrol"},
		{Name: "workstation"},
		{Name: "development"},
		{Name: "sessions"},
		{Name: "workspace"},
	}
}

func TestRequireNewRestoreTarget(t *testing.T) {
	newPath := filepath.Join(t.TempDir(), "new")
	if err := requireNewRestoreTarget(newPath); err != nil {
		t.Fatal(err)
	}
	empty := t.TempDir()
	if err := requireNewRestoreTarget(empty); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(empty, "existing"), "x", 0o600)
	if err := requireNewRestoreTarget(empty); err == nil {
		t.Fatal("expected non-empty restore target to fail")
	}
}

func TestValidatePilotRunnerPlatformRequiresCompleteLinuxContract(t *testing.T) {
	metadata := map[string]string{
		"runner_goos":       "linux",
		"runner_goarch":     "amd64",
		"host_profile":      "wsl2",
		"isolation_backend": "bwrap",
		"sandbox_sha256":    strings.Repeat("a", 64),
	}
	hash, ok := validatePilotRunnerPlatform(
		metadata,
		[]interface{}{"codex", "goclaw-runtime-linux-v1"},
	)
	if !ok || hash != metadata["sandbox_sha256"] {
		t.Fatalf("valid runner contract rejected: hash=%q ok=%v", hash, ok)
	}
	for name, mutation := range map[string]func(map[string]string, *[]interface{}){
		"native macOS": func(value map[string]string, _ *[]interface{}) {
			value["runner_goos"] = "darwin"
		},
		"unreleased architecture": func(value map[string]string, _ *[]interface{}) {
			value["runner_goarch"] = "386"
		},
		"unreviewed profile": func(value map[string]string, _ *[]interface{}) {
			value["host_profile"] = "docker-desktop"
		},
		"host verification": func(value map[string]string, _ *[]interface{}) {
			value["isolation_backend"] = "host"
		},
		"invalid wrapper digest": func(value map[string]string, _ *[]interface{}) {
			value["sandbox_sha256"] = "not-a-sha256"
		},
		"missing runtime capability": func(_ map[string]string, capabilities *[]interface{}) {
			*capabilities = []interface{}{"codex"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			changed := make(map[string]string, len(metadata))
			for key, value := range metadata {
				changed[key] = value
			}
			capabilities := []interface{}{"codex", "goclaw-runtime-linux-v1"}
			mutation(changed, &capabilities)
			if _, ok := validatePilotRunnerPlatform(changed, capabilities); ok {
				t.Fatalf("invalid runner contract accepted: %+v %+v", changed, capabilities)
			}
		})
	}
}

func writeTestFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func mustReadTestFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
