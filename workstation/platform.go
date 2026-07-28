package workstation

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const (
	RunnerRuntimeContract = "goclaw.runner/linux-v1"
	RunnerLinuxCapability = "goclaw-runtime-linux-v1"
)

type RunnerRuntime struct {
	OS                  string `json:"os"`
	Arch                string `json:"arch"`
	Substrate           string `json:"substrate"`
	Contract            string `json:"contract"`
	ExecutionSupported  bool   `json:"execution_supported"`
	WSLInteropDetected  bool   `json:"wsl_interop_detected,omitempty"`
	WindowsPathDetected bool   `json:"windows_path_detected,omitempty"`
	ContainerDetected   bool   `json:"container_detected,omitempty"`
	UnsupportedReason   string `json:"unsupported_reason,omitempty"`
}

func CurrentRunnerRuntime() RunnerRuntime {
	return detectRunnerRuntime(
		runtime.GOOS,
		runtime.GOARCH,
		os.Getenv,
		os.ReadFile,
	)
}

func RunnerRegistrationMetadata() map[string]string {
	info := CurrentRunnerRuntime()
	return map[string]string{
		"runtime_contract":  info.Contract,
		"runtime_os":        info.OS,
		"runtime_arch":      info.Arch,
		"runtime_substrate": info.Substrate,
		"runner_goos":       info.OS,
		"runner_goarch":     info.Arch,
		"host_profile":      info.Substrate,
		"isolation_backend": "unconfigured",
		"sandbox_sha256":    "",
	}
}

func RunnerRegistrationMetadataForSandbox(path string) (map[string]string, error) {
	metadata := RunnerRegistrationMetadata()
	if err := validateSandboxExecutable(path); err != nil {
		return nil, err
	}
	digest, err := sha256File(path)
	if err != nil {
		return nil, err
	}
	metadata["isolation_backend"] = "bwrap"
	metadata["sandbox_sha256"] = digest
	return metadata, nil
}

// RunnerRegistrationCapabilities adds facts resolved by the runner binary.
// The server requires RunnerLinuxCapability for pilot tasks, so a native
// Windows or macOS control CLI can register metadata but cannot claim execution.
func RunnerRegistrationCapabilities(configured []string) []string {
	info := CurrentRunnerRuntime()
	result := append([]string(nil), configured...)
	result = append(
		result,
		"arch:"+strings.ToLower(info.Arch),
		"substrate:"+strings.ToLower(info.Substrate),
	)
	if info.ExecutionSupported {
		result = append(result, RunnerLinuxCapability)
	}
	return normalizeCapabilities(result)
}

func ValidateRunnerExecutionRuntime(info RunnerRuntime) error {
	if info.OS != "linux" || !info.ExecutionSupported {
		reason := strings.TrimSpace(info.UnsupportedReason)
		if reason == "" {
			reason = "runner execution requires the supported Linux runtime"
		}
		return errors.New(reason)
	}
	if info.Substrate == "wsl2" {
		if info.WSLInteropDetected {
			return errors.New(
				"WSL2 interop is enabled; disable [interop] and restart WSL before running a pilot runner",
			)
		}
		if info.WindowsPathDetected {
			return errors.New(
				"WSL2 PATH contains a Windows mount; set appendWindowsPath=false before running a pilot runner",
			)
		}
	}
	return nil
}

func detectRunnerRuntime(
	goos, goarch string,
	getenv func(string) string,
	readFile func(string) ([]byte, error),
) RunnerRuntime {
	info := RunnerRuntime{
		OS:        strings.ToLower(strings.TrimSpace(goos)),
		Arch:      strings.ToLower(strings.TrimSpace(goarch)),
		Substrate: "unsupported-host",
		Contract:  RunnerRuntimeContract,
	}
	if info.OS != "linux" {
		info.UnsupportedReason = fmt.Sprintf(
			"runner execution is disabled on %s; use a Linux guest through WSL2 or Lima",
			valueOr(info.OS, "this host"),
		)
		return info
	}

	info.ExecutionSupported = true
	info.Substrate = "native-linux"
	version, _ := readFile("/proc/version")
	versionLower := strings.ToLower(string(version))
	if strings.TrimSpace(getenv("WSL_DISTRO_NAME")) != "" ||
		strings.Contains(versionLower, "microsoft") {
		info.Substrate = "wsl2"
	}
	if strings.TrimSpace(getenv("LIMA_INSTANCE")) != "" ||
		strings.TrimSpace(getenv("LIMA_CIDATA")) != "" {
		info.Substrate = "lima"
	}
	if _, err := readFile("/.dockerenv"); err == nil {
		info.ContainerDetected = true
	}
	if _, err := readFile("/run/.containerenv"); err == nil {
		info.ContainerDetected = true
	}
	if info.Substrate == "wsl2" {
		info.WSLInteropDetected = strings.TrimSpace(getenv("WSL_INTEROP")) != ""
		if data, err := readFile("/proc/sys/fs/binfmt_misc/WSLInterop"); err == nil {
			info.WSLInteropDetected = info.WSLInteropDetected ||
				strings.Contains(strings.ToLower(string(data)), "enabled")
		}
		info.WindowsPathDetected = containsWindowsMountPath(getenv("PATH"))
	}
	return info
}

func containsWindowsMountPath(value string) bool {
	for _, entry := range filepath.SplitList(value) {
		entry = filepath.ToSlash(filepath.Clean(strings.TrimSpace(entry)))
		if entry == "/mnt" || strings.HasPrefix(entry, "/mnt/") {
			return true
		}
	}
	return false
}

func validateRunnerLocalPath(info RunnerRuntime, path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	clean := filepath.ToSlash(filepath.Clean(absolute))
	if info.Substrate == "wsl2" &&
		(clean == "/mnt" || strings.HasPrefix(clean, "/mnt/")) {
		return fmt.Errorf(
			"%s is on a Windows-mounted filesystem; use the WSL2 virtual disk",
			absolute,
		)
	}
	if info.Substrate == "lima" &&
		(clean == "/mnt" || strings.HasPrefix(clean, "/mnt/")) {
		return fmt.Errorf(
			"%s is under a guest shared-mount root; use a guest-local disk",
			absolute,
		)
	}
	data, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return nil
	}
	filesystem := mountFilesystemForPath(absolute, string(data))
	if sharedRunnerFilesystem(filesystem) {
		return fmt.Errorf(
			"%s uses shared filesystem %s; runner repositories and work roots must be guest-local",
			absolute,
			filesystem,
		)
	}
	return nil
}

func mountFilesystemForPath(path, mountInfo string) string {
	path = filepath.Clean(path)
	type mount struct {
		point string
		fs    string
	}
	var mounts []mount
	for _, line := range strings.Split(mountInfo, "\n") {
		before, after, found := strings.Cut(line, " - ")
		if !found {
			continue
		}
		left := strings.Fields(before)
		right := strings.Fields(after)
		if len(left) < 5 || len(right) < 1 {
			continue
		}
		point := decodeMountInfoPath(left[4])
		if path != point &&
			!strings.HasPrefix(path, strings.TrimSuffix(point, string(filepath.Separator))+string(filepath.Separator)) {
			continue
		}
		mounts = append(mounts, mount{point: point, fs: strings.ToLower(right[0])})
	}
	if len(mounts) == 0 {
		return ""
	}
	sort.Slice(mounts, func(i, j int) bool {
		return len(mounts[i].point) > len(mounts[j].point)
	})
	return mounts[0].fs
}

func decodeMountInfoPath(value string) string {
	replacer := strings.NewReplacer(
		`\040`, " ",
		`\011`, "\t",
		`\012`, "\n",
		`\134`, `\`,
	)
	return filepath.Clean(replacer.Replace(value))
}

func sharedRunnerFilesystem(filesystem string) bool {
	filesystem = strings.ToLower(strings.TrimSpace(filesystem))
	switch filesystem {
	case "9p", "drvfs", "virtiofs", "nfs", "nfs4", "cifs", "smb3",
		"fuse.sshfs", "fuse.lima", "fuse.vmhgfs-fuse":
		return true
	default:
		return strings.HasPrefix(filesystem, "fuse.") &&
			filesystem != "fuse.overlayfs"
	}
}
