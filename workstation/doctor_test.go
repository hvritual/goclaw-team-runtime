package workstation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

type ownerFileInfo struct {
	os.FileInfo
	uid uint32
}

func (i ownerFileInfo) Sys() any {
	return &syscall.Stat_t{Uid: i.uid}
}

func TestRunRunnerDoctorProducesMachineReadableReadyReport(t *testing.T) {
	if CurrentRunnerRuntime().OS != "linux" {
		t.Skip("pilot execution doctor is Linux-only")
	}
	requireGit(t)
	repository, _ := createGitFixture(t)
	keyPath := filepath.Join(t.TempDir(), "runner.key")
	if err := os.WriteFile(keyPath, testDeviceKey('d'), 0o600); err != nil {
		t.Fatal(err)
	}
	fakeCodex := writeExecutable(
		t,
		"codex",
		"#!/bin/sh\nprintf '%s\\n' 'codex-cli pilot-test'\n",
	)
	codexHome := t.TempDir()
	t.Setenv("CODEX_HOME", codexHome)
	report := RunRunnerDoctor(
		context.Background(),
		RunnerDoctorConfig{
			DeviceKeyPath:          keyPath,
			WorkRoot:               t.TempDir(),
			RepositoryPaths:        map[string]string{"fixture": repository},
			CodexCommand:           fakeCodex,
			UnsafeHostVerification: true,
		},
	)
	if !report.Ready {
		t.Fatalf("doctor unexpectedly blocked: %#v", report.Checks)
	}
	if report.SchemaVersion != 1 ||
		report.Metadata["host_profile"] == "" ||
		!containsString(report.Capabilities, RunnerLinuxCapability, true) {
		t.Fatalf("doctor contract missing: %#v", report)
	}
	foundWarning := false
	for _, check := range report.Checks {
		if check.ID == "verification-isolation" &&
			check.Status == DoctorWarn {
			foundWarning = true
		}
	}
	if !foundWarning {
		t.Fatalf("explicit host verification warning missing: %#v", report.Checks)
	}
	data, err := report.MarshalJSONIndent()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"ready": true`) ||
		!strings.Contains(string(data), RunnerLinuxCapability) {
		t.Fatalf("machine report lost readiness contract: %s", data)
	}
}

func TestRunnerRegistrationMetadataForSandboxHashesWrapper(t *testing.T) {
	wrapper := writeExecutable(t, "verify-sandbox-bwrap.sh", "#!/bin/sh\nexit 0\n")
	if os.Geteuid() != 0 {
		if _, err := RunnerRegistrationMetadataForSandbox(wrapper); err == nil ||
			!strings.Contains(err.Error(), "root-owned") {
			t.Fatalf("non-root wrapper error = %v", err)
		}
		return
	}
	metadata, err := RunnerRegistrationMetadataForSandbox(wrapper)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := sha256File(wrapper)
	if err != nil {
		t.Fatal(err)
	}
	if metadata["isolation_backend"] != "bwrap" ||
		metadata["sandbox_sha256"] != expected ||
		metadata["runner_goos"] == "" ||
		metadata["runner_goarch"] == "" ||
		metadata["host_profile"] == "" {
		t.Fatalf("registration metadata = %#v", metadata)
	}
}

func TestRunRunnerDoctorRejectsMissingCodexSubscriptionLogin(t *testing.T) {
	if CurrentRunnerRuntime().OS != "linux" {
		t.Skip("pilot execution doctor is Linux-only")
	}
	requireGit(t)
	repository, _ := createGitFixture(t)
	keyPath := filepath.Join(t.TempDir(), "runner.key")
	if err := os.WriteFile(keyPath, testDeviceKey('e'), 0o600); err != nil {
		t.Fatal(err)
	}
	fakeCodex := writeExecutable(t, "codex-no-login", `#!/bin/sh
if [ "$1" = "login" ]; then
  echo "not logged in" >&2
  exit 1
fi
echo "codex-cli pilot-test"
`)
	t.Setenv("CODEX_HOME", t.TempDir())
	report := RunRunnerDoctor(
		context.Background(),
		RunnerDoctorConfig{
			DeviceKeyPath:          keyPath,
			WorkRoot:               t.TempDir(),
			RepositoryPaths:        map[string]string{"fixture": repository},
			CodexCommand:           fakeCodex,
			UnsafeHostVerification: true,
		},
	)
	if report.Ready {
		t.Fatalf("doctor accepted missing subscription login: %#v", report.Checks)
	}
	for _, check := range report.Checks {
		if check.ID == "codex-login" && check.Status == DoctorFail {
			return
		}
	}
	t.Fatalf("codex-login failure missing: %#v", report.Checks)
}

func TestOwnershipGatesRejectWritableParentAndWrongOwner(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "unsafe-parent")
	if err := os.Mkdir(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	workRoot := filepath.Join(parent, "work")
	if err := os.Mkdir(workRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := validateTrustedPathChain(workRoot, true); err == nil ||
		!strings.Contains(err.Error(), "writable by group or others") {
		t.Fatalf("writable parent was not rejected: %v", err)
	}

	wrapper := writeExecutable(t, "untrusted-wrapper", "#!/bin/sh\nexit 0\n")
	err := validateSandboxExecutable(wrapper)
	if os.Geteuid() == 0 {
		if err != nil {
			t.Fatalf("root-owned wrapper rejected: %v", err)
		}
	} else if err == nil || !strings.Contains(err.Error(), "root-owned") {
		t.Fatalf("non-root wrapper was not rejected: %v", err)
	}

	foreignUID := uint32(os.Geteuid() + 1)
	foreignInfo := ownerFileInfo{uid: foreignUID}
	if err := validateRootOwner("/wrapper", foreignInfo); err == nil ||
		!strings.Contains(err.Error(), "root-owned") {
		t.Fatalf("foreign wrapper owner was not rejected: %v", err)
	}
	if err := validateCurrentUserOwner("/device.key", foreignInfo); err == nil ||
		!strings.Contains(err.Error(), "runner uid") {
		t.Fatalf("foreign device-key owner was not rejected: %v", err)
	}
}
