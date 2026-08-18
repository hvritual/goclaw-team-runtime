package application

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	spacecontract "github.com/hvritual/workspace/internal/modules/space/contract"
	"github.com/hvritual/workspace/internal/modules/system/contract"
)

const skillImportValidatorVersion = "skill-import-v1"

type SkillSourceFetcher func(context.Context, string) ([]byte, error)

type SkillImporter struct {
	repository contract.SkillImportRepository
	objects    spacecontract.SkillObjectService
	authorize  contract.SkillAccessAuthorizer
	preflight  contract.SkillVisibilityPreflight
	bind       contract.SkillVisibilityBinder
	fetch      SkillSourceFetcher
	now        func() time.Time
	newID      func() string
}

func NewSkillImporter(repository contract.SkillImportRepository, objects spacecontract.SkillObjectService, authorize contract.SkillAccessAuthorizer, preflight contract.SkillVisibilityPreflight, bind contract.SkillVisibilityBinder, fetcher SkillSourceFetcher) *SkillImporter {
	if fetcher == nil {
		fetcher = fetchRecognizedSkillURL
	}
	return &SkillImporter{repository: repository, objects: objects, authorize: authorize, preflight: preflight, bind: bind, fetch: fetcher, now: time.Now, newID: uuid.NewString}
}

func (s *SkillImporter) PreviewArchive(ctx context.Context, identity contract.SkillIdentity, data []byte) (contract.SkillImportPreview, error) {
	if err := s.authorize(ctx, identity, contract.PermissionSkillImport); err != nil {
		return contract.SkillImportPreview{}, contract.ErrSkillAccessDenied
	}
	preview, err := ValidateSkillArchive(data)
	if err != nil {
		return contract.SkillImportPreview{}, contract.ErrInvalidSkill
	}
	token := s.newID() + s.newID()
	now := s.now().UTC()
	if err := s.repository.SavePreview(ctx, contract.SkillImportPreviewRecord{
		TokenHash: tokenHash(token), WorkspaceID: identity.WorkspaceID, ActorID: identity.ActorID,
		ValidatorVersion: skillImportValidatorVersion, SourceChecksum: preview.Checksum,
		ExpiresAt: now.Add(15 * time.Minute), CreatedAt: now,
	}); err != nil {
		return contract.SkillImportPreview{}, err
	}
	preview.Token = token
	return preview, nil
}

func (s *SkillImporter) PreviewURL(ctx context.Context, identity contract.SkillIdentity, sourceURL string) (contract.SkillImportPreview, error) {
	if err := s.authorize(ctx, identity, contract.PermissionSkillImport); err != nil {
		return contract.SkillImportPreview{}, contract.ErrSkillAccessDenied
	}
	data, err := s.fetch(ctx, sourceURL)
	if err != nil {
		return contract.SkillImportPreview{}, contract.ErrInvalidSkill
	}
	return s.PreviewArchive(ctx, identity, data)
}

func (s *SkillImporter) ImportArchive(ctx context.Context, identity contract.SkillIdentity, data []byte, previewToken, conflictMode string, expectedRevision int64, idempotencyKey string) (contract.SkillCatalogEntry, error) {
	if err := s.authorize(ctx, identity, contract.PermissionSkillImport); err != nil {
		return contract.SkillCatalogEntry{}, contract.ErrSkillAccessDenied
	}
	if conflictMode == "" {
		conflictMode = "new_version"
	}
	if previewToken == "" || idempotencyKey == "" || (conflictMode != "new_version" && conflictMode != "replace") || conflictMode == "replace" && expectedRevision <= 0 {
		return contract.SkillCatalogEntry{}, contract.ErrInvalidSkill
	}
	preview, err := ValidateSkillArchive(data)
	if err != nil {
		return contract.SkillCatalogEntry{}, contract.ErrInvalidSkill
	}
	previewHash := tokenHash(previewToken)
	requestHash := tokenHash(strings.Join([]string{identity.WorkspaceID, identity.ActorID, previewHash, preview.Checksum, conflictMode, fmt.Sprint(expectedRevision)}, "\x00"))
	if replay, found, err := s.repository.FindImportResult(ctx, identity.WorkspaceID, idempotencyKey, requestHash); err != nil || found {
		return replay, err
	}
	record, err := s.repository.GetPreview(ctx, previewHash)
	if err != nil || record.WorkspaceID != identity.WorkspaceID || record.ActorID != identity.ActorID || record.ValidatorVersion != skillImportValidatorVersion || record.SourceChecksum != preview.Checksum || !s.now().UTC().Before(record.ExpiresAt) {
		return contract.SkillCatalogEntry{}, contract.ErrInvalidSkill
	}
	committed := false
	defer func() {
		if !committed {
			_ = s.repository.DiscardPreview(context.WithoutCancel(ctx), previewHash)
		}
	}()
	skillID, versionID := s.newID(), s.newID()
	if s.preflight == nil || s.bind == nil || s.preflight(ctx, identity, skillID, versionID) != nil {
		return contract.SkillCatalogEntry{}, contract.ErrSkillAccessDenied
	}
	staged := make([]spacecontract.SkillObject, 0, len(preview.Files))
	manifests := make([]contract.SkillFileManifest, 0, len(preview.Files))
	cleanup := true
	defer func() {
		if cleanup {
			for _, object := range staged {
				_ = s.objects.Discard(context.WithoutCancel(ctx), identity.WorkspaceID, object.ID)
			}
		}
	}()
	for _, file := range preview.Files {
		object, err := s.objects.Stage(ctx, spacecontract.StageSkillObjectRequest{WorkspaceID: identity.WorkspaceID, MediaType: file.MediaType, Content: file.Content})
		if err != nil {
			return contract.SkillCatalogEntry{}, err
		}
		staged = append(staged, object)
		manifests = append(manifests, contract.SkillFileManifest{ID: s.newID(), Path: file.Path, SpaceObjectID: object.ID, MediaType: object.MediaType, SizeBytes: object.SizeBytes, Checksum: object.Checksum})
	}
	entry, err := s.repository.Import(ctx, contract.ImportSkillRequest{
		Identity: identity, Name: preview.Name, Description: preview.Description, SourceChecksum: preview.Checksum,
		PreviewTokenHash: previewHash, ConflictMode: conflictMode, ExpectedRevision: expectedRevision,
		IdempotencyKey: idempotencyKey, RequestHash: requestHash, SkillID: skillID, VersionID: versionID, Files: manifests,
	}, s.now().UTC(), func(bindingContext context.Context, executor contract.SkillCreateExecutor) error {
		return s.bind(bindingContext, executor, identity, skillID, versionID)
	}, func(promoteContext context.Context, executor contract.SkillCreateExecutor, objectID string) error {
		return s.objects.Promote(promoteContext, executor, identity.WorkspaceID, objectID)
	})
	if err != nil {
		return contract.SkillCatalogEntry{}, err
	}
	cleanup = false
	committed = true
	return entry, nil
}

func (s *SkillImporter) ImportURL(ctx context.Context, identity contract.SkillIdentity, sourceURL, previewToken, conflictMode string, expectedRevision int64, idempotencyKey string) (contract.SkillCatalogEntry, error) {
	if err := s.authorize(ctx, identity, contract.PermissionSkillImport); err != nil {
		return contract.SkillCatalogEntry{}, contract.ErrSkillAccessDenied
	}
	data, err := s.fetch(ctx, sourceURL)
	if err != nil {
		return contract.SkillCatalogEntry{}, contract.ErrInvalidSkill
	}
	return s.ImportArchive(ctx, identity, data, previewToken, conflictMode, expectedRevision, idempotencyKey)
}

func tokenHash(value string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(value))) }

func fetchRecognizedSkillURL(ctx context.Context, raw string) ([]byte, error) {
	parsed, provider, err := recognizedSkillURL(raw)
	if err != nil {
		return nil, err
	}
	if err := rejectPrivateResolution(ctx, parsed.Hostname()); err != nil {
		return nil, err
	}
	transport := &http.Transport{Proxy: nil, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, DialContext: dialPublicSkillSource}
	client := &http.Client{Transport: transport, Timeout: 45 * time.Second, CheckRedirect: func(request *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many Skill source redirects")
		}
		_, redirectedProvider, err := recognizedSkillURL(request.URL.String())
		if err != nil || redirectedProvider != provider {
			return errors.New("cross-provider Skill source redirect")
		}
		return rejectPrivateResolution(request.Context(), request.URL.Hostname())
	}}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("recognized Skill source unavailable")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, SkillImportMaxCompressedBytes+1))
	if err != nil || len(body) > SkillImportMaxCompressedBytes {
		return nil, errors.New("Skill source exceeds compressed size limit")
	}
	return body, nil
}

func dialPublicSkillSource(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, errors.New("invalid Skill source address")
	}
	addresses, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil || len(addresses) == 0 {
		return nil, errors.New("Skill source host did not resolve")
	}
	for _, candidate := range addresses {
		if candidate.IsLoopback() || candidate.IsPrivate() || candidate.IsLinkLocalUnicast() || candidate.IsLinkLocalMulticast() || candidate.IsUnspecified() {
			continue
		}
		return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, net.JoinHostPort(candidate.String(), port))
	}
	return nil, errors.New("Skill source resolved only to non-public addresses")
}

func recognizedSkillURL(raw string) (*url.URL, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Port() != "" || parsed.Fragment != "" || parsed.RawQuery != "" && len(parsed.RawQuery) > 2048 {
		return nil, "", errors.New("invalid Skill source URL")
	}
	host := strings.ToLower(parsed.Hostname())
	providers := map[string][]string{
		"github":    {"github.com", "raw.githubusercontent.com", "codeload.github.com"},
		"clawhub":   {"clawhub.ai", "www.clawhub.ai"},
		"skills-sh": {"skills.sh", "www.skills.sh"},
	}
	for provider, hosts := range providers {
		for _, allowed := range hosts {
			if host == allowed {
				return parsed, provider, nil
			}
		}
	}
	return nil, "", errors.New("unrecognized Skill source host")
}

func rejectPrivateResolution(ctx context.Context, host string) error {
	addresses, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil || len(addresses) == 0 {
		return errors.New("Skill source host did not resolve")
	}
	for _, address := range addresses {
		if address.IsLoopback() || address.IsPrivate() || address.IsLinkLocalUnicast() || address.IsLinkLocalMulticast() || address.IsUnspecified() {
			return errors.New("Skill source resolved to a non-public address")
		}
	}
	return nil
}

var _ contract.SkillImportService = (*SkillImporter)(nil)
