package main

import (
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: backend-check <fmt|policy|generated> [base-ref]")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "fmt":
		if len(os.Args) != 2 {
			err = fmt.Errorf("fmt accepts no arguments")
			break
		}
		err = runFormatCheck(".")
	case "policy":
		baseRef := "codex/multica-six-domain-baseline"
		if len(os.Args) == 3 {
			baseRef = os.Args[2]
		} else if len(os.Args) != 2 {
			err = fmt.Errorf("policy accepts at most one base ref")
			break
		}
		err = runPolicyCheck(baseRef)
	case "generated":
		if len(os.Args) != 2 {
			err = fmt.Errorf("generated accepts no arguments")
			break
		}
		err = runGeneratedCheck()
	default:
		err = fmt.Errorf("unknown check %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runFormatCheck(root string) error {
	files, err := findUnformattedGoFiles(root)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	return fmt.Errorf("unformatted Go files:\n%s", strings.Join(files, "\n"))
}

func findUnformattedGoFiles(root string) ([]string, error) {
	files := []string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		generatedRoot := filepath.Join("rpc", "pb")
		if relative == generatedRoot || strings.HasPrefix(relative, generatedRoot+string(filepath.Separator)) {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		normalized := bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
		formatted, err := format.Source(normalized)
		if err != nil {
			return fmt.Errorf("format %s: %w", relative, err)
		}
		if !bytes.Equal(normalized, formatted) {
			files = append(files, relative)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func blockedServerPaths(paths []string) []string {
	blocked := []string{}
	for _, path := range paths {
		normalized := strings.ReplaceAll(strings.TrimSpace(path), `\`, "/")
		if normalized == "server" || strings.HasPrefix(normalized, "server/") {
			blocked = append(blocked, normalized)
		}
	}
	sort.Strings(blocked)
	return blocked
}

func findLegacyBackendImports(root string) ([]string, error) {
	legacy := []string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("parse imports in %s: %w", path, err)
		}
		blocked := false
		for _, imported := range parsed.Imports {
			importPath, unquoteErr := strconv.Unquote(imported.Path.Value)
			if unquoteErr != nil {
				return fmt.Errorf("parse import in %s: %w", path, unquoteErr)
			}
			if strings.HasPrefix(importPath, "github.com/hvritual/workspace/"+"server") || strings.Contains(importPath, "/server/"+"internal/") {
				blocked = true
				break
			}
		}
		if !blocked {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		legacy = append(legacy, relative)
		return nil
	})
	sort.Strings(legacy)
	return legacy, err
}

func runPolicyCheck(baseRef string) error {
	repositoryRootBytes, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	repositoryRoot := strings.TrimSpace(string(repositoryRootBytes))
	if baseRef == strings.Repeat("0", 40) {
		command := exec.Command("git", "rev-parse", "HEAD^")
		command.Dir = repositoryRoot
		parent, parentErr := command.Output()
		if parentErr != nil {
			return fmt.Errorf("resolve policy parent: %w", parentErr)
		}
		baseRef = strings.TrimSpace(string(parent))
	}
	verify := exec.Command("git", "rev-parse", "--verify", baseRef+"^{commit}")
	verify.Dir = repositoryRoot
	if output, verifyErr := verify.CombinedOutput(); verifyErr != nil {
		return fmt.Errorf("policy-check: base ref %s is unavailable: %s", baseRef, strings.TrimSpace(string(output)))
	}
	diff := exec.Command("git", "diff", "--name-only", "--diff-filter=ACDMRTUXB", baseRef+"...HEAD")
	diff.Dir = repositoryRoot
	output, err := diff.Output()
	if err != nil {
		return fmt.Errorf("list policy changes: %w", err)
	}
	if blocked := blockedServerPaths(strings.Split(strings.TrimSpace(string(output)), "\n")); len(blocked) > 0 {
		return fmt.Errorf("policy-check: server/** is permanently read-only:\n%s", strings.Join(blocked, "\n"))
	}
	legacy, err := findLegacyBackendImports(filepath.Join(repositoryRoot, "backend"))
	if err != nil {
		return err
	}
	if len(legacy) > 0 {
		return fmt.Errorf("policy-check: canonical backend imports legacy server code:\n%s", strings.Join(legacy, "\n"))
	}
	fmt.Println("policy-check: server boundary and canonical backend dependency boundary passed")
	return nil
}

func runGeneratedCheck() error {
	command := exec.Command("git", "diff", "--exit-code", "--", "rpc/pb", "rpc/openapi")
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("generated output differs: %w", err)
	}
	return nil
}
