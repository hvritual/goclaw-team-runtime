package application

import (
	"context"
	"errors"
	"testing"

	"github.com/hvritual/workspace/internal/modules/system/contract"
)

func TestRecognizedSkillURLAllowsOnlyFrozenHTTPSProviders(t *testing.T) {
	for _, value := range []string{
		"https://github.com/acme/demo/archive/main.zip",
		"https://raw.githubusercontent.com/acme/demo/main/skill.zip",
		"https://codeload.github.com/acme/demo/zip/refs/heads/main",
		"https://clawhub.ai/skills/demo.skill",
		"https://skills.sh/demo.zip",
	} {
		if _, _, err := recognizedSkillURL(value); err != nil {
			t.Errorf("recognizedSkillURL(%q) = %v", value, err)
		}
	}
	for _, value := range []string{
		"http://github.com/acme/demo.zip",
		"https://user:pass@github.com/acme/demo.zip",
		"https://github.com:8443/acme/demo.zip",
		"https://github.com.evil.example/demo.zip",
		"https://127.0.0.1/demo.zip",
		"https://github.com/acme/demo.zip#fragment",
		"file:///tmp/demo.zip",
	} {
		if _, _, err := recognizedSkillURL(value); err == nil {
			t.Errorf("recognizedSkillURL(%q) error = nil", value)
		}
	}
}

func TestSkillImporterAuthorizesBeforeFetchingURL(t *testing.T) {
	fetched := false
	importer := NewSkillImporter(nil, nil, func(context.Context, contract.SkillIdentity, string) error {
		return errors.New("denied")
	}, nil, nil, func(context.Context, string) ([]byte, error) {
		fetched = true
		return nil, nil
	})
	identity := contract.SkillIdentity{WorkspaceID: "workspace-1", ActorID: "actor-1"}
	if _, err := importer.PreviewURL(t.Context(), identity, "https://github.com/acme/demo.zip"); !errors.Is(err, contract.ErrSkillAccessDenied) {
		t.Fatalf("PreviewURL error = %v", err)
	}
	if fetched {
		t.Fatal("PreviewURL fetched before authorization")
	}
	if _, err := importer.ImportURL(t.Context(), identity, "https://github.com/acme/demo.zip", "token", "new_version", 0, "key"); !errors.Is(err, contract.ErrSkillAccessDenied) {
		t.Fatalf("ImportURL error = %v", err)
	}
	if fetched {
		t.Fatal("ImportURL fetched before authorization")
	}
}
