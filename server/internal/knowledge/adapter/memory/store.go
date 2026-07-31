package memory

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/multica-ai/multica/server/internal/knowledge"
)

type Store struct {
	mu              sync.RWMutex
	candidates      map[string]knowledge.Candidate
	entries         map[string]knowledge.Entry
	evidence        map[string]knowledge.Evidence
	idempotencyKeys map[string]string
}

func New() *Store {
	return &Store{
		candidates:      make(map[string]knowledge.Candidate),
		entries:         make(map[string]knowledge.Entry),
		evidence:        make(map[string]knowledge.Evidence),
		idempotencyKeys: make(map[string]string),
	}
}

func (s *Store) CreateCandidate(_ context.Context, candidate knowledge.Candidate) (knowledge.Candidate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	currentTime := time.Now().UTC()
	candidate.ID = uuid.NewString()
	candidate.CreatedAt = currentTime
	candidate.UpdatedAt = currentTime
	candidate.SourceRefs = append([]knowledge.SourceRef(nil), candidate.SourceRefs...)
	s.candidates[candidate.ID] = candidate
	return candidate, nil
}

func (s *Store) GetCandidate(_ context.Context, id string) (knowledge.Candidate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	candidate, ok := s.candidates[id]
	if !ok {
		return knowledge.Candidate{}, knowledge.ErrNotFound
	}
	candidate.SourceRefs = append([]knowledge.SourceRef(nil), candidate.SourceRefs...)
	return candidate, nil
}

func (s *Store) GetEntry(_ context.Context, workspaceID, id string) (knowledge.Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.entries[id]
	if !ok || entry.WorkspaceID != workspaceID {
		return knowledge.Entry{}, knowledge.ErrNotFound
	}
	return cloneEntry(entry), nil
}

func (s *Store) ListCandidates(
	_ context.Context,
	query knowledge.CandidateQuery,
) (knowledge.CandidatePage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset, _ := strconv.Atoi(query.Cursor)
	candidates := make([]knowledge.Candidate, 0)
	for _, candidate := range s.candidates {
		if candidate.WorkspaceID != query.WorkspaceID ||
			(query.ProjectID != "" && candidate.ProjectID != query.ProjectID) ||
			!statusAllowed(candidate.Status, query.Statuses) ||
			!kindAllowed(candidate.Kind, query.Kinds) {
			continue
		}
		candidate.SourceRefs = append([]knowledge.SourceRef(nil), candidate.SourceRefs...)
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].UpdatedAt.Equal(candidates[j].UpdatedAt) {
			return candidates[i].ID < candidates[j].ID
		}
		return candidates[i].UpdatedAt.After(candidates[j].UpdatedAt)
	})
	if offset > len(candidates) {
		offset = len(candidates)
	}
	end := offset + limit
	nextCursor := ""
	if end < len(candidates) {
		nextCursor = strconv.Itoa(end)
	} else {
		end = len(candidates)
	}
	return knowledge.CandidatePage{
		Candidates: append([]knowledge.Candidate(nil), candidates[offset:end]...),
		NextCursor: nextCursor,
	}, nil
}

func (s *Store) ReviewCandidate(
	_ context.Context,
	command knowledge.ReviewCommand,
) (knowledge.Candidate, *knowledge.Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	candidate, ok := s.candidates[command.CandidateID]
	if !ok {
		return knowledge.Candidate{}, nil, knowledge.ErrNotFound
	}
	if candidate.WorkspaceID != command.WorkspaceID {
		return knowledge.Candidate{}, nil, knowledge.ErrWorkspaceMismatch
	}
	if candidate.Revision != command.ExpectedRevision {
		return knowledge.Candidate{}, nil, knowledge.ErrRevisionConflict
	}
	if candidate.Status != knowledge.StatusCandidate && candidate.Status != knowledge.StatusInReview {
		return knowledge.Candidate{}, nil, errors.New("knowledge candidate is already reviewed")
	}

	candidate.Status = command.NewStatus
	candidate.Revision++
	candidate.UpdatedAt = command.Review.ReviewedAt
	s.candidates[candidate.ID] = candidate

	if command.Entry == nil {
		return candidate, nil, nil
	}
	entry := cloneEntry(*command.Entry)
	s.entries[entry.ID] = entry
	result := cloneEntry(entry)
	return candidate, &result, nil
}

func cloneEntry(entry knowledge.Entry) knowledge.Entry {
	entry.Revisions = append([]knowledge.Revision(nil), entry.Revisions...)
	for index := range entry.Revisions {
		entry.Revisions[index].SourceRefs = append(
			[]knowledge.SourceRef(nil),
			entry.Revisions[index].SourceRefs...,
		)
	}
	return entry
}

func (s *Store) IngestEvidence(
	_ context.Context,
	command knowledge.IngestionCommand,
) (knowledge.IngestionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.evidence[command.Evidence.ID]; exists {
		return knowledge.IngestionResult{Duplicate: true}, nil
	}
	if _, exists := s.idempotencyKeys[command.Evidence.IdempotencyKey]; exists {
		return knowledge.IngestionResult{Duplicate: true}, nil
	}

	evidence := command.Evidence
	evidence.SourceRefs = append([]knowledge.SourceRef(nil), evidence.SourceRefs...)
	evidence.Metadata = cloneMap(evidence.Metadata)
	s.evidence[evidence.ID] = evidence
	s.idempotencyKeys[evidence.IdempotencyKey] = evidence.ID

	result := knowledge.IngestionResult{}
	if command.Candidate != nil {
		candidate := *command.Candidate
		candidate.SourceRefs = append([]knowledge.SourceRef(nil), candidate.SourceRefs...)
		s.candidates[candidate.ID] = candidate
		result.Candidate = &candidate
	}
	if command.Entry != nil {
		entry := cloneEntry(*command.Entry)
		s.entries[entry.ID] = entry
		resultEntry := cloneEntry(entry)
		result.Entry = &resultEntry
	}
	return result, nil
}

func cloneMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func (s *Store) Search(_ context.Context, query knowledge.SearchQuery) (knowledge.SearchPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset, _ := strconv.Atoi(query.Cursor)
	textQuery := strings.ToLower(strings.TrimSpace(query.Text))
	results := make([]knowledge.SearchResult, 0)
	for _, entry := range s.entries {
		if entry.WorkspaceID != query.WorkspaceID ||
			entry.Status != knowledge.StatusPublished ||
			(query.ProjectID != "" && entry.ProjectID != query.ProjectID) ||
			!kindAllowed(entry.Kind, query.Kinds) {
			continue
		}
		revision, ok := currentRevision(entry)
		if !ok {
			continue
		}
		haystack := strings.ToLower(revision.Title + "\n" + revision.Content)
		if textQuery != "" && !strings.Contains(haystack, textQuery) {
			continue
		}
		matchedBy := []string{"title", "content"}
		results = append(results, knowledge.SearchResult{
			Entry:     cloneEntry(entry),
			Score:     1,
			MatchedBy: matchedBy,
			Citation:  "knowledge://" + entry.WorkspaceID + "/entries/" + entry.ID,
		})
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Entry.UpdatedAt.Equal(results[j].Entry.UpdatedAt) {
			return results[i].Entry.ID < results[j].Entry.ID
		}
		return results[i].Entry.UpdatedAt.After(results[j].Entry.UpdatedAt)
	})
	if offset > len(results) {
		offset = len(results)
	}
	end := offset + limit
	nextCursor := ""
	if end < len(results) {
		nextCursor = strconv.Itoa(end)
	} else {
		end = len(results)
	}
	return knowledge.SearchPage{
		Results:    append([]knowledge.SearchResult(nil), results[offset:end]...),
		NextCursor: nextCursor,
	}, nil
}

func (s *Store) Rebuild(context.Context) error {
	// The memory adapter derives search results directly from authoritative
	// entries, so there is no secondary index to rebuild.
	return nil
}

func currentRevision(entry knowledge.Entry) (knowledge.Revision, bool) {
	for _, revision := range entry.Revisions {
		if revision.Number == entry.CurrentRevision {
			return revision, true
		}
	}
	return knowledge.Revision{}, false
}

func kindAllowed(kind knowledge.Kind, allowed []knowledge.Kind) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, candidate := range allowed {
		if kind == candidate {
			return true
		}
	}
	return false
}

func statusAllowed(status knowledge.Status, allowed []knowledge.Status) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, candidate := range allowed {
		if status == candidate {
			return true
		}
	}
	return false
}
