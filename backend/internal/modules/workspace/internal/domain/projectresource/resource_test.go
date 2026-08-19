package projectresource

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeGitHubRepository(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rawURL  string
		ref     string
		wantURL string
		wantRef string
	}{
		{
			name:    "https clone URL",
			rawURL:  "https://github.com/Multica-AI/runtime.git/",
			ref:     " refs/heads/main ",
			wantURL: "https://github.com/multica-ai/runtime",
			wantRef: "refs/heads/main",
		},
		{
			name:    "scp clone URL",
			rawURL:  "git@github.com:Multica-AI/runtime.git",
			wantURL: "https://github.com/multica-ai/runtime",
		},
		{
			name:    "ssh clone URL",
			rawURL:  "ssh://git@github.com/Multica-AI/runtime.git",
			wantURL: "https://github.com/multica-ai/runtime",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := Normalize(TypeGitHubRepository, test.rawURL, test.ref)
			if err != nil {
				t.Fatalf("Normalize() error = %v", err)
			}
			if got.URL != test.wantURL || got.Ref != test.wantRef {
				t.Fatalf("Normalize() = %#v, want URL=%q Ref=%q", got, test.wantURL, test.wantRef)
			}
		})
	}
}

func TestNormalizeGenericURL(t *testing.T) {
	t.Parallel()

	got, err := Normalize(TypeURL, " HTTPS://Example.COM:443/docs/guide/ ", "")
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if got.URL != "https://example.com/docs/guide" || got.Ref != "" {
		t.Fatalf("Normalize() = %#v", got)
	}
}

func TestNormalizeGenericURLPreservesSingleEscaping(t *testing.T) {
	t.Parallel()

	got, err := Normalize(TypeURL, "https://example.com/a%20guide/%E8%B5%84%E6%BA%90", "")
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if got.URL != "https://example.com/a%20guide/%E8%B5%84%E6%BA%90" {
		t.Fatalf("Normalize() URL = %q", got.URL)
	}
}

func TestNormalizeRejectsCredentialAndPrivateTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		kind   Type
		rawURL string
		ref    string
	}{
		{name: "generic http", kind: TypeURL, rawURL: "http://example.com/docs"},
		{name: "generic userinfo", kind: TypeURL, rawURL: "https://token@example.com/docs"},
		{name: "generic query", kind: TypeURL, rawURL: "https://example.com/docs?token=secret"},
		{name: "generic fragment", kind: TypeURL, rawURL: "https://example.com/docs#token"},
		{name: "generic encoded control", kind: TypeURL, rawURL: "https://example.com/docs%0Asecret"},
		{name: "generic localhost", kind: TypeURL, rawURL: "https://localhost/docs"},
		{name: "generic loopback", kind: TypeURL, rawURL: "https://127.0.0.1/docs"},
		{name: "generic private address", kind: TypeURL, rawURL: "https://10.0.0.2/docs"},
		{name: "generic shared address", kind: TypeURL, rawURL: "https://100.64.0.1/docs"},
		{name: "generic documentation address", kind: TypeURL, rawURL: "https://192.0.2.1/docs"},
		{name: "generic benchmark address", kind: TypeURL, rawURL: "https://198.18.0.1/docs"},
		{name: "generic IPv6 documentation address", kind: TypeURL, rawURL: "https://[2001:db8::1]/docs"},
		{name: "generic malformed domain", kind: TypeURL, rawURL: "https://bad..example.com/docs"},
		{name: "generic invalid label", kind: TypeURL, rawURL: "https://-bad.example.com/docs"},
		{name: "generic reserved TLD", kind: TypeURL, rawURL: "https://service.invalid/docs"},
		{name: "github token", kind: TypeGitHubRepository, rawURL: "https://token@github.com/acme/repo"},
		{name: "github wrong host", kind: TypeGitHubRepository, rawURL: "https://gitlab.com/acme/repo"},
		{name: "github missing repo", kind: TypeGitHubRepository, rawURL: "https://github.com/acme"},
		{name: "github extra path", kind: TypeGitHubRepository, rawURL: "https://github.com/acme/repo/issues"},
		{name: "github query", kind: TypeGitHubRepository, rawURL: "https://github.com/acme/repo?token=secret"},
		{name: "github invalid ref", kind: TypeGitHubRepository, rawURL: "https://github.com/acme/repo", ref: "main\nsecret"},
		{name: "generic URL over storage limit", kind: TypeURL, rawURL: "https://example.com/" + strings.Repeat("a", 2049)},
		{name: "unknown type", kind: Type("drive"), rawURL: "https://example.com/docs"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Normalize(test.kind, test.rawURL, test.ref); !errors.Is(err, ErrInvalidReference) {
				t.Fatalf("Normalize() error = %v, want ErrInvalidReference", err)
			}
		})
	}
}

func TestFingerprintUsesCanonicalTypeURLAndRef(t *testing.T) {
	t.Parallel()

	first, err := Normalize(TypeGitHubRepository, "git@github.com:Acme/Repo.git", "main")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Normalize(TypeGitHubRepository, "https://github.com/acme/repo", " main ")
	if err != nil {
		t.Fatal(err)
	}
	if Fingerprint(TypeGitHubRepository, first) != Fingerprint(TypeGitHubRepository, second) {
		t.Fatalf("equivalent references produced distinct fingerprints")
	}
	if Fingerprint(TypeGitHubRepository, first) == Fingerprint(TypeGitHubRepository, Reference{URL: first.URL, Ref: "release"}) {
		t.Fatalf("distinct refs produced the same fingerprint")
	}
}
