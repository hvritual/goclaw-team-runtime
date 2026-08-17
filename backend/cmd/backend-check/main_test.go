package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestFindUnformattedGoFilesIsPortableAndExcludesRPCGeneratedOutput(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "formatted.go"), "package sample\n")
	writeTestFile(t, filepath.Join(root, "windows.go"), "package sample\r\n\r\nfunc value() int {\r\n\treturn 1\r\n}\r\n")
	writeTestFile(t, filepath.Join(root, "nested", "unformatted.go"), "package sample\n\nfunc value( )int{return 1}\n")
	writeTestFile(t, filepath.Join(root, "rpc", "pb", "generated.go"), "package pb\n\nfunc generated( )int{return 1}\n")
	writeTestFile(t, filepath.Join(root, "ignored.txt"), "not Go")

	files, err := findUnformattedGoFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join("nested", "unformatted.go")}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("unformatted files = %#v, want %#v", files, want)
	}
}

func TestPolicyHelpersRejectServerChangesAndLegacyImports(t *testing.T) {
	server := blockedServerPaths([]string{"backend/allowed.go", "server/legacy.go", "server"})
	wantServer := []string{"server", "server/legacy.go"}
	if !reflect.DeepEqual(server, wantServer) {
		t.Fatalf("blocked server paths = %#v, want %#v", server, wantServer)
	}

	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "allowed.go"), "package sample\n\nimport _ \"example.com/allowed\"\n")
	writeTestFile(t, filepath.Join(root, "legacy.go"), "package sample\n\nimport _ \"github.com/hvritual/workspace/server/internal/example\"\n")
	writeTestFile(t, filepath.Join(root, "rule.go"), "package sample\n\nconst forbidden = \"github.com/hvritual/workspace/server/internal/example\"\n")
	legacy, err := findLegacyBackendImports(root)
	if err != nil {
		t.Fatal(err)
	}
	wantLegacy := []string{"legacy.go"}
	if !reflect.DeepEqual(legacy, wantLegacy) {
		t.Fatalf("legacy imports = %#v, want %#v", legacy, wantLegacy)
	}
}

func TestWindowsRaceRunnerResolvesHighestInstalledCompiler(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows compiler selection")
	}
	script := filepath.Join("..", "..", "ci", "test-race.ps1")
	command := exec.Command("pwsh", "-NoProfile", "-File", script, "-ResolveOnly")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("resolve Windows race compiler: %v\n%s", err, output)
	}
	selected := strings.TrimSpace(string(output))
	if !strings.EqualFold(filepath.Base(selected), "gcc.exe") {
		t.Fatalf("selected compiler = %q, want gcc.exe", selected)
	}
	selectedVersion, err := exec.Command(selected, "-dumpfullversion").Output()
	if err != nil {
		t.Fatalf("read selected compiler version: %v", err)
	}
	for _, directory := range filepath.SplitList(os.Getenv("PATH")) {
		candidate := filepath.Join(directory, "gcc.exe")
		if _, statErr := os.Stat(candidate); statErr != nil {
			continue
		}
		version, versionErr := exec.Command(candidate, "-dumpfullversion").Output()
		if versionErr != nil {
			continue
		}
		if compilerMajor(t, string(version)) > compilerMajor(t, string(selectedVersion)) {
			t.Fatalf("selected GCC %s but newer PATH candidate %s is available", strings.TrimSpace(string(selectedVersion)), strings.TrimSpace(string(version)))
		}
	}
}

func compilerMajor(t *testing.T, version string) int {
	t.Helper()
	major, err := strconv.Atoi(strings.Split(strings.TrimSpace(version), ".")[0])
	if err != nil {
		t.Fatalf("parse compiler version %q: %v", version, err)
	}
	return major
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
