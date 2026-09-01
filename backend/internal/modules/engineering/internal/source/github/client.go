package github

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/engineering/internal/manifest"
)

const (
	defaultBaseURL = "https://api.github.com/"
	userAgent      = "goclaw-engineering-control-plane"
	maxJSONBytes   = 2 << 20
)

var (
	ErrInvalidLocator    = errors.New("invalid github repository locator")
	ErrInvalidRevision   = errors.New("invalid immutable github revision")
	ErrNotFound          = errors.New("github source not found")
	ErrUnauthorized      = errors.New("github source unauthorized")
	ErrForbidden         = errors.New("github source forbidden")
	ErrRateLimited       = errors.New("github source rate limited")
	ErrUnexpectedStatus  = errors.New("unexpected github response status")
	ErrInvalidPayload    = errors.New("invalid github response payload")
	ErrManifestTooLarge  = errors.New("github engineering manifest too large")
	ErrUnsupportedObject = errors.New("unsupported github contents object")
)

type Client struct {
	httpClient *http.Client
	baseURL    *url.URL
	token      string
}

type Repository struct {
	Locator       string
	NodeID        string
	FullName      string
	DefaultBranch string
	Private       bool
	Archived      bool
	UpdatedAt     time.Time
}

type Commit struct {
	RepositoryLocator string
	SHA               string
	TreeSHA           string
	HTMLURL           string
	AuthoredAt        time.Time
	CommittedAt       time.Time
}

type ManifestBlob struct {
	RepositoryLocator string
	CommitSHA         string
	Path              string
	BlobSHA           string
	Content           []byte
}

type PullRequest struct {
	RepositoryLocator string
	Number            int
	NodeID            string
	State             string
	Merged            bool
	MergeCommitSHA    string
	HeadSHA           string
	BaseSHA           string
	HTMLURL           string
	UpdatedAt         time.Time
}

func New(httpClient *http.Client, baseURL, token string) (*Client, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("%w: http client is required", ErrInvalidPayload)
	}
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("%w: invalid API base URL", ErrInvalidPayload)
	}
	if !strings.HasSuffix(parsed.Path, "/") {
		parsed.Path += "/"
	}
	copied := *httpClient
	copied.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{httpClient: &copied, baseURL: parsed, token: strings.TrimSpace(token)}, nil
}

func (c *Client) GetRepository(ctx context.Context, locator string) (Repository, error) {
	owner, repository, canonical, err := parseRepositoryLocator(locator)
	if err != nil {
		return Repository{}, err
	}
	var payload struct {
		NodeID        string    `json:"node_id"`
		FullName      string    `json:"full_name"`
		DefaultBranch string    `json:"default_branch"`
		Private       bool      `json:"private"`
		Archived      bool      `json:"archived"`
		UpdatedAt     time.Time `json:"updated_at"`
	}
	if err := c.getJSON(ctx, repositoryEndpoint(owner, repository), nil, &payload); err != nil {
		return Repository{}, err
	}
	if strings.TrimSpace(payload.NodeID) == "" || strings.TrimSpace(payload.FullName) == "" || strings.TrimSpace(payload.DefaultBranch) == "" {
		return Repository{}, fmt.Errorf("%w: incomplete repository response", ErrInvalidPayload)
	}
	return Repository{
		Locator: canonical, NodeID: payload.NodeID, FullName: payload.FullName, DefaultBranch: payload.DefaultBranch,
		Private: payload.Private, Archived: payload.Archived, UpdatedAt: payload.UpdatedAt.UTC(),
	}, nil
}

func (c *Client) ResolveCommit(ctx context.Context, locator, ref string) (Commit, error) {
	owner, repository, canonical, err := parseRepositoryLocator(locator)
	if err != nil {
		return Commit{}, err
	}
	ref = strings.TrimSpace(ref)
	if ref == "" || strings.ContainsAny(ref, "\r\n") {
		return Commit{}, fmt.Errorf("%w: empty or unsafe ref", ErrInvalidRevision)
	}
	var payload struct {
		SHA     string `json:"sha"`
		HTMLURL string `json:"html_url"`
		Commit  struct {
			Author struct {
				Date time.Time `json:"date"`
			} `json:"author"`
			Committer struct {
				Date time.Time `json:"date"`
			} `json:"committer"`
			Tree struct {
				SHA string `json:"sha"`
			} `json:"tree"`
		} `json:"commit"`
	}
	endpoint := repositoryEndpoint(owner, repository) + "/commits/" + url.PathEscape(ref)
	if err := c.getJSON(ctx, endpoint, nil, &payload); err != nil {
		return Commit{}, err
	}
	if !isImmutableGitSHA(payload.SHA) || strings.TrimSpace(payload.Commit.Tree.SHA) == "" {
		return Commit{}, fmt.Errorf("%w: incomplete commit response", ErrInvalidPayload)
	}
	return Commit{
		RepositoryLocator: canonical, SHA: strings.ToLower(payload.SHA), TreeSHA: strings.ToLower(payload.Commit.Tree.SHA), HTMLURL: payload.HTMLURL,
		AuthoredAt: payload.Commit.Author.Date.UTC(), CommittedAt: payload.Commit.Committer.Date.UTC(),
	}, nil
}

func (c *Client) ReadEngineeringManifestAtCommit(ctx context.Context, locator, commitSHA string) (ManifestBlob, error) {
	owner, repository, canonical, err := parseRepositoryLocator(locator)
	if err != nil {
		return ManifestBlob{}, err
	}
	commitSHA = strings.ToLower(strings.TrimSpace(commitSHA))
	if !isImmutableGitSHA(commitSHA) {
		return ManifestBlob{}, ErrInvalidRevision
	}
	query := url.Values{"ref": []string{commitSHA}}
	var payload struct {
		Type     string `json:"type"`
		Encoding string `json:"encoding"`
		Content  string `json:"content"`
		SHA      string `json:"sha"`
		Path     string `json:"path"`
		Size     int64  `json:"size"`
	}
	endpoint := repositoryEndpoint(owner, repository) + "/contents/engineering.yaml"
	if err := c.getJSON(ctx, endpoint, query, &payload); err != nil {
		return ManifestBlob{}, err
	}
	if payload.Type != "file" || payload.Encoding != "base64" {
		return ManifestBlob{}, fmt.Errorf("%w: type=%q encoding=%q", ErrUnsupportedObject, payload.Type, payload.Encoding)
	}
	if payload.Size < 0 || payload.Size > manifest.MaxManifestBytes {
		return ManifestBlob{}, fmt.Errorf("%w: size=%d", ErrManifestTooLarge, payload.Size)
	}
	encoded := strings.NewReplacer("\r", "", "\n", "").Replace(payload.Content)
	content, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return ManifestBlob{}, fmt.Errorf("%w: invalid base64 manifest", ErrInvalidPayload)
	}
	if len(content) > manifest.MaxManifestBytes || (payload.Size > 0 && int64(len(content)) != payload.Size) {
		return ManifestBlob{}, fmt.Errorf("%w: decoded size mismatch", ErrInvalidPayload)
	}
	if strings.TrimSpace(payload.SHA) == "" || strings.TrimSpace(payload.Path) != "engineering.yaml" {
		return ManifestBlob{}, fmt.Errorf("%w: incomplete contents response", ErrInvalidPayload)
	}
	return ManifestBlob{
		RepositoryLocator: canonical, CommitSHA: commitSHA, Path: "engineering.yaml", BlobSHA: strings.ToLower(payload.SHA), Content: content,
	}, nil
}

func (c *Client) GetPullRequest(ctx context.Context, locator string, number int) (PullRequest, error) {
	owner, repository, canonical, err := parseRepositoryLocator(locator)
	if err != nil {
		return PullRequest{}, err
	}
	if number <= 0 {
		return PullRequest{}, fmt.Errorf("%w: pull request number", ErrInvalidPayload)
	}
	var payload struct {
		Number         int       `json:"number"`
		NodeID         string    `json:"node_id"`
		State          string    `json:"state"`
		Merged         bool      `json:"merged"`
		MergeCommitSHA string    `json:"merge_commit_sha"`
		HTMLURL        string    `json:"html_url"`
		UpdatedAt      time.Time `json:"updated_at"`
		Head           struct {
			SHA string `json:"sha"`
		} `json:"head"`
		Base struct {
			SHA string `json:"sha"`
		} `json:"base"`
	}
	endpoint := repositoryEndpoint(owner, repository) + "/pulls/" + strconv.Itoa(number)
	if err := c.getJSON(ctx, endpoint, nil, &payload); err != nil {
		return PullRequest{}, err
	}
	if payload.Number != number || strings.TrimSpace(payload.NodeID) == "" || !isImmutableGitSHA(payload.Head.SHA) || !isImmutableGitSHA(payload.Base.SHA) {
		return PullRequest{}, fmt.Errorf("%w: incomplete pull request response", ErrInvalidPayload)
	}
	return PullRequest{
		RepositoryLocator: canonical, Number: payload.Number, NodeID: payload.NodeID, State: payload.State, Merged: payload.Merged,
		MergeCommitSHA: strings.ToLower(strings.TrimSpace(payload.MergeCommitSHA)), HeadSHA: strings.ToLower(payload.Head.SHA), BaseSHA: strings.ToLower(payload.Base.SHA),
		HTMLURL: payload.HTMLURL, UpdatedAt: payload.UpdatedAt.UTC(),
	}, nil
}

func (c *Client) getJSON(ctx context.Context, endpoint string, query url.Values, out any) error {
	if c == nil || c.httpClient == nil || c.baseURL == nil {
		return fmt.Errorf("%w: uninitialized client", ErrInvalidPayload)
	}
	target, err := url.Parse(c.baseURL.String() + strings.TrimPrefix(endpoint, "/"))
	if err != nil || target.Host != c.baseURL.Host || target.Scheme != c.baseURL.Scheme {
		return fmt.Errorf("%w: invalid request target", ErrInvalidPayload)
	}
	if query != nil {
		target.RawQuery = query.Encode()
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return fmt.Errorf("%w: create request", ErrInvalidPayload)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", userAgent)
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("github source request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return mapStatus(response)
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxJSONBytes+1))
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("%w: decode response", ErrInvalidPayload)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: trailing response data", ErrInvalidPayload)
	}
	return nil
}

func mapStatus(response *http.Response) error {
	requestID := strings.TrimSpace(response.Header.Get("X-GitHub-Request-Id"))
	detail := fmt.Sprintf("status=%d", response.StatusCode)
	if requestID != "" {
		detail += " request_id=" + requestID
	}
	switch response.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: %s", ErrUnauthorized, detail)
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", ErrNotFound, detail)
	case http.StatusTooManyRequests:
		return fmt.Errorf("%w: %s", ErrRateLimited, detail)
	case http.StatusForbidden:
		if strings.TrimSpace(response.Header.Get("X-RateLimit-Remaining")) == "0" {
			return fmt.Errorf("%w: %s", ErrRateLimited, detail)
		}
		return fmt.Errorf("%w: %s", ErrForbidden, detail)
	default:
		return fmt.Errorf("%w: %s", ErrUnexpectedStatus, detail)
	}
}

func repositoryEndpoint(owner, repository string) string {
	return "repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repository)
}

func parseRepositoryLocator(raw string) (owner, repository, canonical string, err error) {
	parsed, parseErr := url.Parse(strings.TrimSpace(raw))
	if parseErr != nil || !strings.EqualFold(parsed.Scheme, "github") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", "", "", ErrInvalidLocator
	}
	owner = strings.TrimSpace(parsed.Host)
	repository = strings.Trim(strings.TrimSpace(parsed.Path), "/")
	if owner == "" || repository == "" || strings.Contains(repository, "/") || strings.ContainsAny(owner+repository, " \t\r\n") {
		return "", "", "", ErrInvalidLocator
	}
	repository = strings.TrimSuffix(repository, ".git")
	if repository == "" {
		return "", "", "", ErrInvalidLocator
	}
	owner = strings.ToLower(owner)
	repository = strings.ToLower(repository)
	return owner, repository, "github://" + owner + "/" + repository, nil
}

func isImmutableGitSHA(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
