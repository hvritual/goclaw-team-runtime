package projectresource

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/netip"
	"net/url"
	"path"
	"regexp"
	"strings"
)

type Type string

const (
	TypeGitHubRepository Type = "github_repo"
	TypeURL              Type = "url"
)

var ErrInvalidReference = errors.New("invalid Project Resource reference")

var (
	githubSCPPattern         = regexp.MustCompile(`(?i)^git@github\.com:([^/]+)/([^/]+?)(?:\.git)?/?$`)
	githubNamePattern        = regexp.MustCompile(`^[a-z0-9_.-]+$`)
	domainLabelPattern       = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
	nonPublicAddressPrefixes = []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/8"),
		netip.MustParsePrefix("100.64.0.0/10"),
		netip.MustParsePrefix("192.0.0.0/24"),
		netip.MustParsePrefix("192.0.2.0/24"),
		netip.MustParsePrefix("192.88.99.0/24"),
		netip.MustParsePrefix("198.18.0.0/15"),
		netip.MustParsePrefix("198.51.100.0/24"),
		netip.MustParsePrefix("203.0.113.0/24"),
		netip.MustParsePrefix("240.0.0.0/4"),
		netip.MustParsePrefix("100::/64"),
		netip.MustParsePrefix("2001:2::/48"),
		netip.MustParsePrefix("2001:db8::/32"),
	}
)

type Reference struct {
	URL string
	Ref string
}

func Normalize(kind Type, rawURL, rawRef string) (Reference, error) {
	if containsControl(rawURL) || containsControl(rawRef) {
		return Reference{}, ErrInvalidReference
	}
	switch kind {
	case TypeGitHubRepository:
		return normalizeGitHub(rawURL, rawRef)
	case TypeURL:
		return normalizeURL(rawURL, rawRef)
	default:
		return Reference{}, ErrInvalidReference
	}
}

func Fingerprint(kind Type, reference Reference) string {
	sum := sha256.Sum256([]byte(string(kind) + "\x00" + reference.URL + "\x00" + reference.Ref))
	return hex.EncodeToString(sum[:])
}

func normalizeGitHub(rawURL, rawRef string) (Reference, error) {
	value := strings.TrimSpace(rawURL)
	if match := githubSCPPattern.FindStringSubmatch(value); len(match) == 3 {
		return canonicalGitHub(match[1], match[2], rawRef)
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Hostname() == "" {
		return Reference{}, ErrInvalidReference
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" && scheme != "http" && scheme != "ssh" {
		return Reference{}, ErrInvalidReference
	}
	if !strings.EqualFold(parsed.Hostname(), "github.com") || parsed.Port() != "" {
		return Reference{}, ErrInvalidReference
	}
	if parsed.User != nil {
		if scheme != "ssh" || parsed.User.Username() != "git" {
			return Reference{}, ErrInvalidReference
		}
		if _, hasPassword := parsed.User.Password(); hasPassword {
			return Reference{}, ErrInvalidReference
		}
	}
	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(segments) != 2 {
		return Reference{}, ErrInvalidReference
	}
	return canonicalGitHub(segments[0], segments[1], rawRef)
}

func canonicalGitHub(owner, repository, rawRef string) (Reference, error) {
	owner = strings.ToLower(strings.TrimSpace(owner))
	repository = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(repository), ".git"))
	ref := strings.TrimSpace(rawRef)
	if owner == "" || repository == "" || len(owner) > 100 || len(repository) > 100 ||
		!githubNamePattern.MatchString(owner) || !githubNamePattern.MatchString(repository) ||
		len(ref) > 255 || containsControl(ref) {
		return Reference{}, ErrInvalidReference
	}
	return Reference{URL: "https://github.com/" + owner + "/" + repository, Ref: ref}, nil
}

func normalizeURL(rawURL, rawRef string) (Reference, error) {
	if strings.TrimSpace(rawRef) != "" {
		return Reference{}, ErrInvalidReference
	}
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") || parsed.Hostname() == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" ||
		containsControl(parsed.Path) {
		return Reference{}, ErrInvalidReference
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if !publicHostForm(host) {
		return Reference{}, ErrInvalidReference
	}
	port := parsed.Port()
	if port == "443" {
		port = ""
	}
	parsed.Scheme = "https"
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.RawFragment = ""
	parsed.ForceQuery = false
	parsed.Host = host
	if port != "" {
		parsed.Host = net.JoinHostPort(host, port)
	}
	cleanPath := path.Clean(parsed.Path)
	if cleanPath == "." || cleanPath == "/" {
		cleanPath = ""
	}
	parsed.Path = cleanPath
	parsed.RawPath = ""
	canonical := parsed.String()
	if len(canonical) > 2048 {
		return Reference{}, ErrInvalidReference
	}
	return Reference{URL: canonical}, nil
}

func publicHostForm(host string) bool {
	if host == "" {
		return false
	}
	if address, err := netip.ParseAddr(host); err == nil {
		address = address.Unmap()
		if !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() ||
			address.IsLinkLocalUnicast() || address.IsMulticast() || address.IsUnspecified() {
			return false
		}
		for _, prefix := range nonPublicAddressPrefixes {
			if prefix.Contains(address) {
				return false
			}
		}
		return true
	}
	if len(host) > 253 || !strings.Contains(host, ".") || reservedDomainName(host) {
		return false
	}
	labels := strings.Split(host, ".")
	for _, label := range labels {
		if !domainLabelPattern.MatchString(label) {
			return false
		}
	}
	for _, character := range labels[len(labels)-1] {
		if character < '0' || character > '9' {
			return true
		}
	}
	return false
}

func reservedDomainName(host string) bool {
	for _, suffix := range []string{"localhost", "local", "invalid", "test", "internal", "example", "home.arpa", "onion"} {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}
	return false
}

func containsControl(value string) bool {
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return true
		}
	}
	return false
}
