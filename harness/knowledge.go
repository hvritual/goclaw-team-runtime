package harness

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/smallnest/goclaw/governance"
)

var governedKnowledgeRoots = []string{
	"01-goals",
	"02-decisions",
	"03-constraints",
	"04-requirements",
	"05-knowledge",
}

func (s *Service) CreateKnowledgeProposal(targetPath, proposedContent, reason, evidenceTraceID, actor string) (KnowledgeProposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	targetPath, target, err := s.resolveKnowledgeTarget(targetPath)
	if err != nil {
		return KnowledgeProposal{}, err
	}
	baseHash := ""
	if _, err := os.Stat(target); err == nil {
		baseHash, err = fileSHA256(target)
		if err != nil {
			return KnowledgeProposal{}, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return KnowledgeProposal{}, err
	}
	baseRevision, err := s.knowledgeRevision()
	if err != nil {
		return KnowledgeProposal{}, err
	}
	if strings.TrimSpace(proposedContent) == "" {
		return KnowledgeProposal{}, errors.New("proposed content must not be empty")
	}
	if strings.TrimSpace(reason) == "" {
		return KnowledgeProposal{}, errors.New("proposal reason must not be empty")
	}
	if strings.TrimSpace(actor) == "" {
		actor = "goclaw-agent"
	}
	proposal := KnowledgeProposal{
		SchemaVersion:   SchemaVersion,
		ID:              "kp-" + uuid.NewString(),
		ProjectID:       s.cfg.ProjectID,
		TargetPath:      targetPath,
		BaseSHA256:      baseHash,
		BaseRevision:    baseRevision,
		SourceURI:       s.knowledgeSourceURI(targetPath),
		StoreKind:       s.cfg.KnowledgeBackend,
		ProposedContent: proposedContent,
		Reason:          reason,
		EvidenceTraceID: evidenceTraceID,
		Status:          KnowledgeProposalPending,
		CreatedBy:       actor,
		CreatedAt:       time.Now().UTC(),
	}
	if err := writeJSONAtomic(s.knowledgeProposalPath(proposal.ID), proposal, 0o644); err != nil {
		return KnowledgeProposal{}, err
	}
	if err := s.writeKnowledgeProjection(proposal); err != nil {
		return KnowledgeProposal{}, err
	}
	return proposal, nil
}

func (s *Service) GetKnowledgeProposal(id string) (KnowledgeProposal, error) {
	var proposal KnowledgeProposal
	if err := readJSON(s.knowledgeProposalPath(id), &proposal); err != nil {
		return KnowledgeProposal{}, err
	}
	return proposal, nil
}

func (s *Service) ListKnowledgeProposals(status KnowledgeProposalStatus) ([]KnowledgeProposal, error) {
	entries, err := os.ReadDir(s.knowledgeProposalsDir())
	if err != nil {
		return nil, err
	}
	result := make([]KnowledgeProposal, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var proposal KnowledgeProposal
		if err := readJSON(filepath.Join(s.knowledgeProposalsDir(), entry.Name()), &proposal); err != nil {
			continue
		}
		if status == "" || proposal.Status == status {
			result = append(result, proposal)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return result, nil
}

func (s *Service) ApproveKnowledgeProposal(id, reviewer, comment string) (KnowledgeProposal, error) {
	if s.governance.Enabled && s.governance.RequireAuthenticatedReviewers {
		return KnowledgeProposal{}, errors.New("authenticated governance requires ApproveKnowledgeProposalWithReview")
	}
	return s.ApproveKnowledgeProposalWithReview(id, governance.Review{
		ReviewerID:    reviewer,
		Rationale:     comment,
		Role:          governance.RoleKnowledgeApprove,
		Source:        "local-cli",
		Authenticated: true,
		CreatedAt:     time.Now().UTC(),
	})
}

func (s *Service) ApproveKnowledgeProposalWithReview(
	id string,
	review governance.Review,
) (KnowledgeProposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	proposal, err := s.GetKnowledgeProposal(id)
	if err != nil {
		return KnowledgeProposal{}, err
	}
	if proposal.Status != KnowledgeProposalPending {
		return KnowledgeProposal{}, fmt.Errorf("knowledge proposal %s is not pending", id)
	}
	if err := governance.ValidateRole(review, governance.RoleKnowledgeApprove); err != nil {
		return KnowledgeProposal{}, err
	}
	if err := governance.ValidateApproval(s.governance, review, proposal.CreatedBy); err != nil {
		return KnowledgeProposal{}, err
	}
	decision := governance.ToDecision(review, "approved")
	_, target, err := s.resolveKnowledgeTarget(proposal.TargetPath)
	if err != nil {
		return KnowledgeProposal{}, err
	}
	currentHash := ""
	if _, err := os.Stat(target); err == nil {
		currentHash, err = fileSHA256(target)
		if err != nil {
			return KnowledgeProposal{}, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return KnowledgeProposal{}, err
	}
	if currentHash != proposal.BaseSHA256 {
		return KnowledgeProposal{}, fmt.Errorf("knowledge conflict for %s: governed Markdown changed since proposal creation", proposal.TargetPath)
	}
	currentRevision, err := s.knowledgeRevision()
	if err != nil {
		return KnowledgeProposal{}, err
	}
	if proposal.BaseRevision != "" && currentRevision != proposal.BaseRevision {
		return KnowledgeProposal{}, fmt.Errorf(
			"knowledge conflict for %s: base revision changed from %s to %s",
			proposal.TargetPath,
			proposal.BaseRevision,
			currentRevision,
		)
	}
	if err := writeBytesAtomic(target, []byte(proposal.ProposedContent), 0o644); err != nil {
		return KnowledgeProposal{}, err
	}
	proposal.Status = KnowledgeProposalApproved
	proposal.ReviewedBy = decision.ReviewerID
	proposal.ReviewComment = decision.Rationale
	proposal.ReviewedAt = &decision.CreatedAt
	proposal.Review = &decision
	if err := writeJSONAtomic(s.knowledgeProposalPath(id), proposal, 0o644); err != nil {
		return KnowledgeProposal{}, err
	}
	if err := s.writeKnowledgeProjection(proposal); err != nil {
		return KnowledgeProposal{}, err
	}
	return proposal, nil
}

func (s *Service) RejectKnowledgeProposal(id, reviewer, comment string) (KnowledgeProposal, error) {
	if s.governance.Enabled && s.governance.RequireAuthenticatedReviewers {
		return KnowledgeProposal{}, errors.New("authenticated governance requires RejectKnowledgeProposalWithReview")
	}
	return s.RejectKnowledgeProposalWithReview(id, governance.Review{
		ReviewerID:    reviewer,
		Rationale:     comment,
		Role:          governance.RoleKnowledgeApprove,
		Source:        "local-cli",
		Authenticated: true,
		CreatedAt:     time.Now().UTC(),
	})
}

func (s *Service) RejectKnowledgeProposalWithReview(
	id string,
	review governance.Review,
) (KnowledgeProposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	proposal, err := s.GetKnowledgeProposal(id)
	if err != nil {
		return KnowledgeProposal{}, err
	}
	if proposal.Status != KnowledgeProposalPending {
		return KnowledgeProposal{}, fmt.Errorf("knowledge proposal %s is not pending", id)
	}
	if err := governance.ValidateRole(review, governance.RoleKnowledgeApprove); err != nil {
		return KnowledgeProposal{}, err
	}
	if err := governance.ValidateDecision(s.governance, review, "rejected", proposal.CreatedBy); err != nil {
		return KnowledgeProposal{}, err
	}
	decision := governance.ToDecision(review, "rejected")
	proposal.Status = KnowledgeProposalRejected
	proposal.ReviewedBy = decision.ReviewerID
	proposal.ReviewComment = decision.Rationale
	proposal.ReviewedAt = &decision.CreatedAt
	proposal.Review = &decision
	if err := writeJSONAtomic(s.knowledgeProposalPath(id), proposal, 0o644); err != nil {
		return KnowledgeProposal{}, err
	}
	if err := s.writeKnowledgeProjection(proposal); err != nil {
		return KnowledgeProposal{}, err
	}
	return proposal, nil
}

func (s *Service) ReadKnowledge(targetPath string) (string, error) {
	_, target, err := s.resolveKnowledgeTarget(targetPath)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return "", err
	}
	if len(data) > 2*1024*1024 {
		return "", fmt.Errorf("knowledge file exceeds 2 MiB limit")
	}
	return string(data), nil
}

func (s *Service) SearchKnowledge(query string, limit int) ([]KnowledgeSearchResult, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil, errors.New("knowledge search query must not be empty")
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	rootPath, err := s.resolvedKnowledgeRoot()
	if err != nil {
		return nil, err
	}
	results := make([]KnowledgeSearchResult, 0, limit)
	for _, root := range governedKnowledgeRoots {
		dir, err := safeJoin(rootPath, root)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
			continue
		}
		walkErr := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if len(results) >= limit {
				return filepath.SkipAll
			}
			if entry.Type()&os.ModeSymlink != 0 {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".md" {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if info.Size() > 2*1024*1024 {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			lower := strings.ToLower(string(data))
			index := strings.Index(lower, query)
			if index < 0 {
				return nil
			}
			start := index - 100
			if start < 0 {
				start = 0
			}
			end := index + len(query) + 180
			if end > len(data) {
				end = len(data)
			}
			rel, err := filepath.Rel(rootPath, path)
			if err != nil {
				return err
			}
			results = append(results, KnowledgeSearchResult{
				Path:    filepath.ToSlash(rel),
				Excerpt: strings.TrimSpace(string(data[start:end])),
			})
			return nil
		})
		if walkErr != nil {
			return nil, walkErr
		}
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}

func (s *Service) resolveKnowledgeTarget(rel string) (string, string, error) {
	if strings.TrimSpace(s.cfg.KnowledgeRoot) == "" {
		return "", "", errors.New("harness.knowledge_root is required for knowledge proposals")
	}
	rel = filepath.ToSlash(filepath.Clean(strings.TrimSpace(rel)))
	if rel == "." || filepath.Ext(rel) != ".md" {
		return "", "", errors.New("knowledge target must be a relative Markdown path")
	}
	allowed := false
	for _, root := range governedKnowledgeRoots {
		if rel == root || strings.HasPrefix(rel, root+"/") {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", "", fmt.Errorf("knowledge target %s is outside governed folders", rel)
	}
	rootPath, err := s.resolvedKnowledgeRoot()
	if err != nil {
		return "", "", err
	}
	target, err := safeJoin(rootPath, filepath.FromSlash(rel))
	if err != nil {
		return "", "", err
	}
	if err := ensureNoSymlinkComponents(rootPath, target); err != nil {
		return "", "", err
	}
	return rel, target, nil
}

func (s *Service) resolvedKnowledgeRoot() (string, error) {
	rootPath, err := filepath.Abs(s.cfg.KnowledgeRoot)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(rootPath)
	if err != nil {
		return "", fmt.Errorf("resolve knowledge root: %w", err)
	}
	return resolved, nil
}

func (s *Service) knowledgeRevision() (string, error) {
	if s.cfg.KnowledgeBackend != "git" {
		return "", nil
	}
	rootPath, err := s.resolvedKnowledgeRoot()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(
		ctx,
		"git",
		"-C",
		rootPath,
		"rev-parse",
		"--verify",
		"HEAD",
	).Output()
	if err != nil {
		return "", fmt.Errorf("resolve Git knowledge revision: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (s *Service) knowledgeSourceURI(targetPath string) string {
	scheme := "markdown"
	if s.cfg.KnowledgeBackend == "git" {
		scheme = "git+markdown"
	}
	return fmt.Sprintf("%s://%s/%s", scheme, s.cfg.ProjectID, filepath.ToSlash(targetPath))
}

func ensureNoSymlinkComponents(root, target string) error {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return err
	}
	current := root
	for _, component := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink is not allowed in governed knowledge path: %s", current)
		}
	}
	return nil
}

func (s *Service) knowledgeProposalPath(id string) string {
	path, err := safeJoin(s.knowledgeProposalsDir(), id+".json")
	if err != nil {
		return filepath.Join(s.knowledgeProposalsDir(), "__invalid__.json")
	}
	return path
}

func (s *Service) writeKnowledgeProjection(proposal KnowledgeProposal) error {
	if strings.TrimSpace(s.cfg.KnowledgeRoot) == "" {
		return nil
	}
	stateDir := "inbox"
	if proposal.Status == KnowledgeProposalApproved {
		stateDir = "approved"
	}
	if proposal.Status == KnowledgeProposalRejected {
		stateDir = "rejected"
	}
	rootPath, err := s.resolvedKnowledgeRoot()
	if err != nil {
		return err
	}
	projectionPath, err := safeJoin(rootPath, filepath.Join("08-reviews", stateDir, proposal.ID+".md"))
	if err != nil {
		return err
	}
	body := fmt.Sprintf(`---
type: knowledge-proposal
id: %s
project_id: %s
status: %s
target_path: %s
source_uri: %s
store_kind: %s
base_revision: %s
created_by: %s
created_at: %s
reviewed_by: %s
---

# Knowledge proposal %s

## Reason

%s

## Proposed content

~~~markdown
%s
~~~
`, proposal.ID, proposal.ProjectID, proposal.Status, proposal.TargetPath,
		proposal.SourceURI, proposal.StoreKind, proposal.BaseRevision,
		proposal.CreatedBy, proposal.CreatedAt.Format(time.RFC3339), proposal.ReviewedBy,
		proposal.ID, proposal.Reason, proposal.ProposedContent)
	if err := writeBytesAtomic(projectionPath, []byte(body), 0o644); err != nil {
		return err
	}
	for _, dir := range []string{"inbox", "approved", "rejected"} {
		if dir == stateDir {
			continue
		}
		oldPath, joinErr := safeJoin(rootPath, filepath.Join("08-reviews", dir, proposal.ID+".md"))
		if joinErr == nil {
			if err := os.Remove(oldPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	return nil
}
