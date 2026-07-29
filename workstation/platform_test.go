package workstation

import (
	"errors"
	"strings"
	"testing"
)

func TestDetectRunnerRuntimeSupportMatrix(t *testing.T) {
	tests := []struct {
		name        string
		goos        string
		env         map[string]string
		files       map[string]string
		substrate   string
		supported   bool
		interop     bool
		windowsPath bool
	}{
		{
			name:      "native linux",
			goos:      "linux",
			substrate: "native-linux",
			supported: true,
		},
		{
			name: "wsl2 interop and windows path",
			goos: "linux",
			env: map[string]string{
				"WSL_DISTRO_NAME": "Ubuntu",
				"WSL_INTEROP":     "/run/WSL/1_interop",
				"PATH":            "/usr/bin:/mnt/c/Windows/System32",
			},
			files: map[string]string{
				"/proc/version": "Linux version microsoft-standard-WSL2",
			},
			substrate:   "wsl2",
			supported:   true,
			interop:     true,
			windowsPath: true,
		},
		{
			name:      "lima guest",
			goos:      "linux",
			env:       map[string]string{"LIMA_INSTANCE": "goclaw-pilot"},
			substrate: "lima",
			supported: true,
		},
		{
			name:      "native windows is control only",
			goos:      "windows",
			substrate: "native-windows",
			supported: false,
		},
		{
			name:      "native macos is control only",
			goos:      "darwin",
			substrate: "native-darwin",
			supported: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			getenv := func(name string) string {
				return test.env[name]
			}
			readFile := func(path string) ([]byte, error) {
				value, found := test.files[path]
				if !found {
					return nil, errors.New("not found")
				}
				return []byte(value), nil
			}
			got := detectRunnerRuntime(test.goos, "amd64", getenv, readFile)
			if got.Substrate != test.substrate ||
				got.ExecutionSupported != test.supported ||
				got.WSLInteropDetected != test.interop ||
				got.WindowsPathDetected != test.windowsPath {
				t.Fatalf("runtime = %#v", got)
			}
		})
	}
}

func TestValidateRunnerExecutionRuntimeFailsClosed(t *testing.T) {
	for _, info := range []RunnerRuntime{
		{OS: "windows", UnsupportedReason: "use WSL2"},
		{OS: "darwin", UnsupportedReason: "use Lima"},
		{
			OS:                 "linux",
			ExecutionSupported: true,
			Substrate:          "wsl2",
			WSLInteropDetected: true,
		},
		{
			OS:                  "linux",
			ExecutionSupported:  true,
			Substrate:           "wsl2",
			WindowsPathDetected: true,
		},
	} {
		if err := ValidateRunnerExecutionRuntime(info); err == nil {
			t.Fatalf("unsafe runtime unexpectedly passed: %#v", info)
		}
	}
	if err := ValidateRunnerExecutionRuntime(RunnerRuntime{
		OS: "linux", Arch: "arm64", Substrate: "lima",
		ExecutionSupported: true,
	}); err != nil {
		t.Fatalf("Linux guest rejected: %v", err)
	}
}

func TestMountFilesystemForPathUsesMostSpecificMount(t *testing.T) {
	mountInfo := strings.Join([]string{
		"26 22 0:22 / / rw,relatime - ext4 /dev/root rw",
		"31 26 0:27 / /mnt/c rw,relatime - 9p C:\\134 rw",
		"32 26 0:28 / /workspace rw,relatime - fuse.overlayfs overlay rw",
	}, "\n")
	if got := mountFilesystemForPath("/mnt/c/src/app", mountInfo); got != "9p" {
		t.Fatalf("filesystem = %q, want 9p", got)
	}
	if got := mountFilesystemForPath("/workspace/project", mountInfo); got != "fuse.overlayfs" {
		t.Fatalf("filesystem = %q, want fuse.overlayfs", got)
	}
	if !sharedRunnerFilesystem("virtiofs") ||
		!sharedRunnerFilesystem("fuse.lima") ||
		sharedRunnerFilesystem("ext4") ||
		sharedRunnerFilesystem("fuse.overlayfs") {
		t.Fatal("shared filesystem classification is incorrect")
	}
}

func TestRunnerRegistrationCapabilitiesAdvertiseLinuxContract(t *testing.T) {
	got := RunnerRegistrationCapabilities([]string{"codex", "Go", "codex"})
	if CurrentRunnerRuntime().OS == "linux" &&
		!containsString(got, RunnerLinuxCapability, true) {
		t.Fatalf("Linux capability missing: %v", got)
	}
	if !containsString(got, "arch:"+CurrentRunnerRuntime().Arch, true) {
		t.Fatalf("architecture capability missing: %v", got)
	}
}
