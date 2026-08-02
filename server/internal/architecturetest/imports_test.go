// Package architecturetest verifies generated DDD module dependency direction.
package architecturetest

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

const moduleImportRoot = "github.com/multica-ai/multica/server/internal/modules/"

var allowedCrossModuleContracts = map[string]map[string]bool{
	"auth": {"workspace": true},
}

func TestGeneratedModuleImportsRespectBoundaries(t *testing.T) {
	root := repositoryRoot(t)
	modulesRoot := filepath.Join(root, "internal", "modules")
	err := filepath.WalkDir(modulesRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		checkFileImports(t, modulesRoot, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk generated modules: %v", err)
	}
}

func checkFileImports(t *testing.T, modulesRoot, path string) {
	t.Helper()
	relative, err := filepath.Rel(modulesRoot, path)
	if err != nil {
		t.Fatalf("relative module path: %v", err)
	}
	parts := strings.Split(filepath.ToSlash(relative), "/")
	if len(parts) < 2 {
		return
	}
	owner := parts[0]
	layer := moduleLayer(parts)
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v", relative, err)
	}
	for _, imported := range file.Imports {
		pathValue, unquoteErr := strconv.Unquote(imported.Path.Value)
		if unquoteErr != nil {
			t.Fatalf("unquote import in %s: %v", relative, unquoteErr)
		}
		if violation := importViolation(owner, layer, pathValue); violation != "" {
			t.Errorf("%s imports %q: %s", filepath.ToSlash(relative), pathValue, violation)
		}
	}
}

func moduleLayer(parts []string) string {
	if len(parts) >= 4 && parts[1] == "internal" {
		return parts[2]
	}
	if len(parts) >= 3 && parts[1] == "contract" {
		return "contract"
	}
	return "root"
}

func importViolation(owner, layer, imported string) string {
	if violation := crossModuleViolation(owner, imported); violation != "" {
		return violation
	}
	return layerViolation(owner, layer, imported)
}

func crossModuleViolation(owner, imported string) string {
	if strings.HasPrefix(imported, moduleImportRoot) {
		remainder := strings.TrimPrefix(imported, moduleImportRoot)
		parts := strings.Split(remainder, "/")
		if len(parts) > 0 && parts[0] != owner {
			if len(parts) < 2 || parts[1] != "contract" {
				return "cross-module imports must target the provider contract"
			}
			if !allowedCrossModuleContracts[owner][parts[0]] {
				return "cross-module contract import is not an intentional registered edge"
			}
		}
	}
	return ""
}

func layerViolation(owner, layer, imported string) string {
	ownRoot := moduleImportRoot + owner + "/internal/"
	switch layer {
	case "domain":
		if strings.HasPrefix(imported, ownRoot) && !strings.HasPrefix(imported, ownRoot+"domain") {
			return "domain may import only its own domain packages"
		}
		if strings.Contains(imported, "/gen/") || strings.Contains(imported, "kratos") || strings.Contains(imported, "grpc") {
			return "domain must not depend on generated or transport packages"
		}
	case "application":
		if strings.HasPrefix(imported, ownRoot+"infrastructure") || strings.HasPrefix(imported, ownRoot+"interfaces") {
			return "application must depend inward, not on adapters"
		}
	case "infrastructure":
		if strings.HasPrefix(imported, ownRoot+"interfaces") {
			return "infrastructure must not depend on transport adapters"
		}
	case "interfaces":
		if strings.HasPrefix(imported, ownRoot+"infrastructure") {
			return "interfaces must not depend on infrastructure adapters"
		}
	}
	return ""
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve architecture test path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
