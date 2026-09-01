package github

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hvritual/workspace/internal/modules/engineering/internal/manifest"
)

const (
	testCommitSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testTreeSHA   = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	testHeadSHA   = "cccccccccccccccccccccccccccccccccccccccc"
	testBaseSHA   = "dddddddddddddddddddddddddddddddddddddddd"
)

func TestClientReadsRepositoryCommitManifestAndPullRequest(t *testing.T) {
	secret := "test-secret-must-not-leak"
	manifestBytes := []byte("schema_version: v1\nentity:\n  id: service-a\n  type: service\n  name: A\n  status: active\nsource:\n  type: github\n  locator: github://acme/device-gateway\n")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+secret {
			t.Errorf("authorization = %q", got)
		}
		if r.Header.Get("Accept") != "application/vnd.github+json" || r.Header.Get("X-GitHub-Api-Version") != "2022-11-28" || r.Header.Get("User-Agent") != userAgent {
			t.Errorf("missing canonical github headers: %#v", r.Header)
		}
		switch {
		case r.URL.Path == "/repos/acme/device-gateway":
			fmt.Fprint(w, `{"node_id":"R_1","full_name":"Acme/Device-Gateway","default_branch":"main","private":true,"archived":false,"updated_at":"2026-08-28T12:00:00Z"}`)
		case strings.HasPrefix(r.RequestURI, "/repos/acme/device-gateway/commits/"):
			if !strings.Contains(r.RequestURI, "feature%2Fweak-network") {
				t.Errorf("commit ref was not escaped as one path value: %s", r.RequestURI)
			}
			fmt.Fprintf(w, `{"sha":%q,"html_url":"https://github.com/acme/device-gateway/commit/%s","commit":{"author":{"date":"2026-08-28T11:00:00Z"},"committer":{"date":"2026-08-28T11:01:00Z"},"tree":{"sha":%q}}}`, testCommitSHA, testCommitSHA, testTreeSHA)
		case r.URL.Path == "/repos/acme/device-gateway/contents/engineering.yaml":
			if r.URL.Query().Get("ref") != testCommitSHA {
				t.Errorf("manifest ref = %q", r.URL.Query().Get("ref"))
			}
			fmt.Fprintf(w, `{"type":"file","encoding":"base64","content":%q,"sha":"blob-sha","path":"engineering.yaml","size":%d}`, base64.StdEncoding.EncodeToString(manifestBytes), len(manifestBytes))
		case r.URL.Path == "/repos/acme/device-gateway/pulls/7":
			fmt.Fprintf(w, `{"number":7,"node_id":"PR_7","state":"closed","merged":true,"merge_commit_sha":%q,"html_url":"https://github.com/acme/device-gateway/pull/7","updated_at":"2026-08-28T12:10:00Z","head":{"sha":%q},"base":{"sha":%q}}`, testCommitSHA, testHeadSHA, testBaseSHA)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := New(server.Client(), server.URL+"/", secret)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	repository, err := client.GetRepository(ctx, "github://Acme/Device-Gateway.git")
	if err != nil {
		t.Fatal(err)
	}
	if repository.Locator != "github://acme/device-gateway" || repository.NodeID != "R_1" || repository.DefaultBranch != "main" || !repository.Private {
		t.Fatalf("repository = %+v", repository)
	}
	commit, err := client.ResolveCommit(ctx, repository.Locator, "feature/weak-network")
	if err != nil {
		t.Fatal(err)
	}
	if commit.SHA != testCommitSHA || commit.TreeSHA != testTreeSHA || commit.RepositoryLocator != repository.Locator {
		t.Fatalf("commit = %+v", commit)
	}
	blob, err := client.ReadEngineeringManifestAtCommit(ctx, repository.Locator, commit.SHA)
	if err != nil {
		t.Fatal(err)
	}
	if blob.CommitSHA != testCommitSHA || blob.BlobSHA != "blob-sha" || string(blob.Content) != string(manifestBytes) {
		t.Fatalf("manifest blob = %+v", blob)
	}
	parsed, err := manifest.Parse(blob.Content)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Entity.ID != "service-a" {
		t.Fatalf("parsed manifest = %+v", parsed)
	}
	pull, err := client.GetPullRequest(ctx, repository.Locator, 7)
	if err != nil {
		t.Fatal(err)
	}
	if pull.Number != 7 || !pull.Merged || pull.HeadSHA != testHeadSHA || pull.BaseSHA != testBaseSHA {
		t.Fatalf("pull request = %+v", pull)
	}
}

func TestClientMapsGitHubStatusWithoutLeakingTokenOrBody(t *testing.T) {
	for _, test := range []struct {
		name      string
		status    int
		remaining string
		want      error
	}{
		{name: "unauthorized", status: http.StatusUnauthorized, want: ErrUnauthorized},
		{name: "not found", status: http.StatusNotFound, want: ErrNotFound},
		{name: "forbidden", status: http.StatusForbidden, remaining: "42", want: ErrForbidden},
		{name: "rate limited forbidden", status: http.StatusForbidden, remaining: "0", want: ErrRateLimited},
		{name: "rate limited", status: http.StatusTooManyRequests, want: ErrRateLimited},
		{name: "server", status: http.StatusInternalServerError, want: ErrUnexpectedStatus},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-GitHub-Request-Id", "REQ-123")
				if test.remaining != "" {
					w.Header().Set("X-RateLimit-Remaining", test.remaining)
				}
				w.WriteHeader(test.status)
				fmt.Fprint(w, `{"message":"body-secret"}`)
			}))
			defer server.Close()
			client, err := New(server.Client(), server.URL+"/", "token-secret")
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.GetRepository(context.Background(), "github://acme/repo")
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
			if strings.Contains(err.Error(), "token-secret") || strings.Contains(err.Error(), "body-secret") {
				t.Fatalf("error leaked sensitive content: %v", err)
			}
		})
	}
}

func TestClientRequiresPinnedManifestRevisionAndBoundsContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		oversized := make([]byte, manifest.MaxManifestBytes+1)
		fmt.Fprintf(w, `{"type":"file","encoding":"base64","content":%q,"sha":"blob","path":"engineering.yaml","size":%d}`, base64.StdEncoding.EncodeToString(oversized), len(oversized))
	}))
	defer server.Close()
	client, err := New(server.Client(), server.URL+"/", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.ReadEngineeringManifestAtCommit(context.Background(), "github://acme/repo", "main"); !errors.Is(err, ErrInvalidRevision) {
		t.Fatalf("mutable ref error = %v", err)
	}
	if _, err := client.ReadEngineeringManifestAtCommit(context.Background(), "github://acme/repo", testCommitSHA); !errors.Is(err, ErrManifestTooLarge) {
		t.Fatalf("oversized manifest error = %v", err)
	}
}

func TestClientRejectsInvalidRepositoryLocatorsAndPayloads(t *testing.T) {
	client, err := New(http.DefaultClient, "https://api.github.com/", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, locator := range []string{"", "https://github.com/acme/repo", "github://acme", "github://acme/repo/sub", "github://a b/repo"} {
		if _, err := client.GetRepository(context.Background(), locator); !errors.Is(err, ErrInvalidLocator) {
			t.Fatalf("locator %q error = %v", locator, err)
		}
	}
	if _, err := client.GetPullRequest(context.Background(), "github://acme/repo", 0); !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("pull request number error = %v", err)
	}
	if _, err := New(nil, "", ""); !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("nil client error = %v", err)
	}
}
