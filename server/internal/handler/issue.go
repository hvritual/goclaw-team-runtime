package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/issueguard"
	"github.com/multica-ai/multica/server/internal/logger"
	"github.com/multica-ai/multica/server/internal/service"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

// IssueResponse is the JSON response for an issue.
type IssueResponse struct {
	ID            string  `json:"id"`
	WorkspaceID   string  `json:"workspace_id"`
	Number        int32   `json:"number"`
	Identifier    string  `json:"identifier"`
	Title         string  `json:"title"`
	Description   *string `json:"description"`
	Status        string  `json:"status"`
	Priority      string  `json:"priority"`
	AssigneeType  *string `json:"assignee_type"`
	AssigneeID    *string `json:"assignee_id"`
	CreatorType   string  `json:"creator_type"`
	CreatorID     string  `json:"creator_id"`
	ParentIssueID *string `json:"parent_issue_id"`
	ProjectID     *string `json:"project_id"`
	Position      float64 `json:"position"`
	// Stage groups sub-issues under the same parent into ordered barrier
	// groups (null = unstaged). See issue_child_done.go for how a closed
	// stage gates the child-done -> parent wake.
	Stage     *int32  `json:"stage"`
	StartDate *string `json:"start_date"`
	DueDate   *string `json:"due_date"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
	// Metadata is the per-issue KV map (see issue_metadata.go). Always emitted
	// (empty object when unset) so frontend code can `issue.metadata[key]`
	// without nil-guarding the parent field.
	Metadata map[string]any `json:"metadata"`
	// Properties is the custom-property value bag keyed by property definition
	// UUID (see property.go). Always emitted, mirroring Metadata.
	Properties  map[string]any          `json:"properties"`
	Reactions   []IssueReactionResponse `json:"reactions,omitempty"`
	Attachments []AttachmentResponse    `json:"attachments,omitempty"`
	// Labels are bulk-attached by list/detail endpoints so the client can render
	// chips without an N+1 round-trip per row. Pointer + omitempty so paths that
	// don't load labels (e.g. UpdateIssue, batch UpdateIssues, the issue:updated
	// WS broadcast) emit no `labels` field at all — the client merge then
	// preserves whatever labels are already in cache. nil pointer = "field
	// absent, do not touch"; non-nil (incl. empty slice) = authoritative list.
	Labels *[]LabelResponse `json:"labels,omitempty"`
}

// validIssueStatuses / validIssuePriorities mirror the CHECK constraints on
// the issue table. Write handlers pre-validate these so callers get a clean
// 400 with the allowed values instead of a database CHECK violation bubbling
// up as a 500.
var validIssueStatuses = []string{"backlog", "todo", "in_progress", "in_review", "done", "blocked", "cancelled"}
var validIssuePriorities = []string{"urgent", "high", "medium", "low", "none"}

func validateIssueEnum(w http.ResponseWriter, field, value string, allowed []string) bool {
	for _, a := range allowed {
		if value == a {
			return true
		}
	}
	writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid %s %q; valid values: %s", field, value, strings.Join(allowed, ", ")))
	return false
}

func issueToResponse(i db.Issue, issuePrefix string) IssueResponse {
	identifier := issuePrefix + "-" + strconv.Itoa(int(i.Number))
	return IssueResponse{
		ID:            uuidToString(i.ID),
		WorkspaceID:   uuidToString(i.WorkspaceID),
		Number:        i.Number,
		Identifier:    identifier,
		Title:         i.Title,
		Description:   textToPtr(i.Description),
		Status:        i.Status,
		Priority:      i.Priority,
		AssigneeType:  textToPtr(i.AssigneeType),
		AssigneeID:    uuidToPtr(i.AssigneeID),
		CreatorType:   i.CreatorType,
		CreatorID:     uuidToString(i.CreatorID),
		ParentIssueID: uuidToPtr(i.ParentIssueID),
		ProjectID:     uuidToPtr(i.ProjectID),
		Position:      i.Position,
		Stage:         int4ToPtr(i.Stage),
		StartDate:     dateToPtr(i.StartDate),
		DueDate:       dateToPtr(i.DueDate),
		CreatedAt:     timestampToString(i.CreatedAt),
		UpdatedAt:     timestampToString(i.UpdatedAt),
		Metadata:      parseIssueMetadata(i.Metadata),
		Properties:    parseIssueProperties(i.Properties),
	}
}

// issueListRowToResponse converts a list-query row (no description) to an IssueResponse.
func issueListRowToResponse(i db.ListIssuesRow, issuePrefix string) IssueResponse {
	identifier := issuePrefix + "-" + strconv.Itoa(int(i.Number))
	return IssueResponse{
		ID:            uuidToString(i.ID),
		WorkspaceID:   uuidToString(i.WorkspaceID),
		Number:        i.Number,
		Identifier:    identifier,
		Title:         i.Title,
		Description:   textToPtr(i.Description),
		Status:        i.Status,
		Priority:      i.Priority,
		AssigneeType:  textToPtr(i.AssigneeType),
		AssigneeID:    uuidToPtr(i.AssigneeID),
		CreatorType:   i.CreatorType,
		CreatorID:     uuidToString(i.CreatorID),
		ParentIssueID: uuidToPtr(i.ParentIssueID),
		ProjectID:     uuidToPtr(i.ProjectID),
		Position:      i.Position,
		Stage:         int4ToPtr(i.Stage),
		StartDate:     dateToPtr(i.StartDate),
		DueDate:       dateToPtr(i.DueDate),
		CreatedAt:     timestampToString(i.CreatedAt),
		UpdatedAt:     timestampToString(i.UpdatedAt),
		Metadata:      parseIssueMetadata(i.Metadata),
		Properties:    parseIssueProperties(i.Properties),
	}
}

// labelsByIssue bulk-loads labels for the given issue IDs and returns a map
// keyed by issue UUID string. On error or empty input, returns an empty map —
// label rendering is non-critical and we'd rather serve issues without labels
// than fail the whole list call.
func (h *Handler) labelsByIssue(ctx context.Context, wsUUID pgtype.UUID, issueIDs []pgtype.UUID) map[string][]LabelResponse {
	out := map[string][]LabelResponse{}
	if len(issueIDs) == 0 {
		return out
	}
	rows, err := h.Queries.ListLabelsForIssues(ctx, db.ListLabelsForIssuesParams{
		IssueIds:    issueIDs,
		WorkspaceID: wsUUID,
	})
	if err != nil {
		slog.Warn("ListLabelsForIssues failed", "error", err)
		return out
	}
	for _, r := range rows {
		issueID := uuidToString(r.IssueID)
		out[issueID] = append(out[issueID], LabelResponse{
			ID:           uuidToString(r.ID),
			WorkspaceID:  uuidToString(r.WorkspaceID),
			ResourceType: r.ResourceType,
			Name:         r.Name,
			Description:  r.Description,
			Color:        r.Color,
			CreatedAt:    timestampToString(r.CreatedAt),
			UpdatedAt:    timestampToString(r.UpdatedAt),
		})
	}
	return out
}

func openIssueRowToResponse(i db.ListOpenIssuesRow, issuePrefix string) IssueResponse {
	identifier := issuePrefix + "-" + strconv.Itoa(int(i.Number))
	return IssueResponse{
		ID:            uuidToString(i.ID),
		WorkspaceID:   uuidToString(i.WorkspaceID),
		Number:        i.Number,
		Identifier:    identifier,
		Title:         i.Title,
		Description:   textToPtr(i.Description),
		Status:        i.Status,
		Priority:      i.Priority,
		AssigneeType:  textToPtr(i.AssigneeType),
		AssigneeID:    uuidToPtr(i.AssigneeID),
		CreatorType:   i.CreatorType,
		CreatorID:     uuidToString(i.CreatorID),
		ParentIssueID: uuidToPtr(i.ParentIssueID),
		ProjectID:     uuidToPtr(i.ProjectID),
		Position:      i.Position,
		Stage:         int4ToPtr(i.Stage),
		StartDate:     dateToPtr(i.StartDate),
		DueDate:       dateToPtr(i.DueDate),
		CreatedAt:     timestampToString(i.CreatedAt),
		UpdatedAt:     timestampToString(i.UpdatedAt),
		Metadata:      parseIssueMetadata(i.Metadata),
		Properties:    parseIssueProperties(i.Properties),
	}
}

type IssueAssigneeGroupResponse struct {
	ID           string          `json:"id"`
	AssigneeType *string         `json:"assignee_type"`
	AssigneeID   *string         `json:"assignee_id"`
	Issues       []IssueResponse `json:"issues"`
	Total        int64           `json:"total"`
}

type GroupedIssuesResponse struct {
	Groups []IssueAssigneeGroupResponse `json:"groups"`
}

type groupedIssueRow struct {
	db.ListIssuesRow
	GroupTotal int64
}

func assigneeGroupID(assigneeType pgtype.Text, assigneeID pgtype.UUID) string {
	if assigneeType.Valid && assigneeID.Valid {
		return "assignee:" + assigneeType.String + ":" + uuidToString(assigneeID)
	}
	return "assignee:unassigned"
}

// SearchIssueResponse extends IssueResponse with search metadata.
type SearchIssueResponse struct {
	IssueResponse
	MatchSource               string  `json:"match_source"`
	MatchedSnippet            *string `json:"matched_snippet,omitempty"`
	MatchedDescriptionSnippet *string `json:"matched_description_snippet,omitempty"`
	MatchedCommentSnippet     *string `json:"matched_comment_snippet,omitempty"`
}

// extractSnippet extracts a snippet of text around the first occurrence of query.
// Returns up to ~120 runes centered on the match. Uses rune-based slicing to
// avoid splitting multi-byte UTF-8 characters (important for CJK content).
// For multi-word queries, tries phrase match first; if not found, locates the
// earliest occurring individual term and centers the snippet around it.
func extractSnippet(content, query string) string {
	runes := []rune(content)
	lowerRunes := []rune(strings.ToLower(content))
	queryRunes := []rune(strings.ToLower(query))

	idx := findRuneSubstring(lowerRunes, queryRunes)

	// If phrase not found, try individual terms for multi-word queries.
	matchLen := len(queryRunes)
	if idx < 0 {
		terms := strings.Fields(strings.ToLower(query))
		if len(terms) > 1 {
			earliest := -1
			earliestLen := 0
			for _, term := range terms {
				termRunes := []rune(term)
				pos := findRuneSubstring(lowerRunes, termRunes)
				if pos >= 0 && (earliest < 0 || pos < earliest) {
					earliest = pos
					earliestLen = len(termRunes)
				}
			}
			if earliest >= 0 {
				idx = earliest
				matchLen = earliestLen
			}
		}
	}

	if idx < 0 {
		if len(runes) > 120 {
			return string(runes[:120]) + "..."
		}
		return content
	}
	start := idx - 40
	if start < 0 {
		start = 0
	}
	end := idx + matchLen + 80
	if end > len(runes) {
		end = len(runes)
	}
	snippet := string(runes[start:end])
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(runes) {
		snippet = snippet + "..."
	}
	return snippet
}

// findRuneSubstring returns the index of needle in haystack, or -1 if not found.
func findRuneSubstring(haystack, needle []rune) int {
	if len(needle) == 0 || len(haystack) < len(needle) {
		return -1
	}
	for i := 0; i <= len(haystack)-len(needle); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// descriptionContains checks if the description text contains the search phrase or all terms.
func descriptionContains(desc pgtype.Text, phrase string, terms []string) bool {
	if !desc.Valid || desc.String == "" {
		return false
	}
	lower := strings.ToLower(desc.String)
	if strings.Contains(lower, strings.ToLower(phrase)) {
		return true
	}
	if len(terms) > 1 {
		for _, t := range terms {
			if !strings.Contains(lower, strings.ToLower(t)) {
				return false
			}
		}
		return true
	}
	return false
}

// escapeLike escapes LIKE special characters (%, _, \) in user input.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// splitSearchTerms splits a query into individual search terms, filtering empty strings.
func splitSearchTerms(q string) []string {
	fields := strings.FieldsFunc(q, func(r rune) bool {
		return unicode.IsSpace(r)
	})
	terms := make([]string, 0, len(fields))
	for _, f := range fields {
		if f != "" {
			terms = append(terms, f)
		}
	}
	return terms
}

// identifierNumberRe matches patterns like "MUL-123" or "ABC-45".
var identifierNumberRe = regexp.MustCompile(`(?i)^[a-z]+-(\d+)$`)

// parseQueryNumber extracts an issue number from the query if it looks like
// an identifier (e.g. "MUL-123") or a bare number (e.g. "123").
func parseQueryNumber(q string) (int, bool) {
	q = strings.TrimSpace(q)
	// Check for identifier pattern like "MUL-123"
	if m := identifierNumberRe.FindStringSubmatch(q); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil && n > 0 {
			return n, true
		}
	}
	// Check for bare number
	if n, err := strconv.Atoi(q); err == nil && n > 0 {
		return n, true
	}
	return 0, false
}

// searchResult holds a raw row from the dynamic search query.
type searchResult struct {
	issue                 db.Issue
	totalCount            int64
	matchSource           string
	matchedCommentContent string
}

// buildSearchQuery builds a dynamic SQL query for issue search.
// It uses LOWER(column) LIKE for case-insensitive matching compatible with pg_bigm 1.2 GIN indexes.
// Search patterns are lowercased in Go to avoid redundant LOWER() on the pattern side in SQL.
// LIKE patterns are pre-built in Go (e.g. "%html%") so pg_bigm can extract bigrams from a single parameter value.
func buildSearchQuery(phrase string, terms []string, queryNum int, hasNum bool, includeClosed bool) (string, []any) {
	// Lowercase in Go so SQL only needs LOWER() on the column side.
	phrase = strings.ToLower(phrase)
	for i, t := range terms {
		terms[i] = strings.ToLower(t)
	}

	// Parameter index tracker
	argIdx := 1
	args := []any{}
	nextArg := func(val any) string {
		args = append(args, val)
		s := fmt.Sprintf("$%d", argIdx)
		argIdx++
		return s
	}

	escapedPhrase := escapeLike(phrase)
	// $1: exact phrase (for exact title match)
	phraseParam := nextArg(escapedPhrase)
	// $2: "%phrase%" (contains pattern — pre-built for pg_bigm index usage)
	phraseContainsParam := nextArg("%" + escapedPhrase + "%")
	// $3: "phrase%" (starts-with pattern)
	phraseStartsWithParam := nextArg(escapedPhrase + "%")

	wsParam := nextArg(nil) // $4 — workspace_id, will be filled by caller position

	// Build per-term LIKE conditions only for multi-word search.
	var termContainsParams []string
	if len(terms) > 1 {
		for _, t := range terms {
			et := escapeLike(t)
			termContainsParams = append(termContainsParams, nextArg("%"+et+"%"))
		}
	}

	// --- WHERE clause ---
	var whereParts []string

	// Full phrase match: title, description, or comment.
	//
	// The comment EXISTS subquery is deliberately correlated on BOTH
	// c.issue_id = i.id AND c.workspace_id = wsParam. The workspace_id
	// filter is not strictly necessary for correctness (comment.workspace_id
	// is FK-consistent with its issue's workspace), but it is critical for
	// the planner. Without it, Postgres rewrites the correlated EXISTS
	// into a hashed subplan that materializes every comment in the entire
	// `comment` table matching the LIKE — for common tokens like "search"
	// this can be hundreds of thousands of rows, blowing out work_mem into
	// a lossy bitmap and taking 30+ seconds. With the workspace_id
	// constant duplicated into the subquery, the hashed set collapses to
	// this workspace's comments and the plan uses the supporting
	// idx_comment_workspace (migration 135). See MUL-4059 EXPLAIN reports.
	phraseMatch := fmt.Sprintf(
		"(LOWER(i.title) LIKE %s OR LOWER(COALESCE(i.description, '')) LIKE %s OR EXISTS (SELECT 1 FROM comment c WHERE c.issue_id = i.id AND c.workspace_id = %s AND LOWER(c.content) LIKE %s))",
		phraseContainsParam, phraseContainsParam, wsParam, phraseContainsParam,
	)
	whereParts = append(whereParts, phraseMatch)

	// Multi-word AND match (each term must appear somewhere). Same
	// workspace_id-in-subquery contract as above.
	if len(termContainsParams) > 1 {
		var termConditions []string
		for _, tp := range termContainsParams {
			termConditions = append(termConditions, fmt.Sprintf(
				"(LOWER(i.title) LIKE %s OR LOWER(COALESCE(i.description, '')) LIKE %s OR EXISTS (SELECT 1 FROM comment c WHERE c.issue_id = i.id AND c.workspace_id = %s AND LOWER(c.content) LIKE %s))",
				tp, tp, wsParam, tp,
			))
		}
		whereParts = append(whereParts, "("+strings.Join(termConditions, " AND ")+")")
	}

	// Number match
	numParam := ""
	if hasNum {
		numParam = nextArg(queryNum)
		whereParts = append(whereParts, fmt.Sprintf("i.number = %s", numParam))
	}

	whereClause := "(" + strings.Join(whereParts, " OR ") + ")"

	if !includeClosed {
		whereClause += " AND i.status NOT IN ('done', 'cancelled')"
	}

	// --- ORDER BY clause ---
	// Build ranking CASE with fine-grained tiers.
	var rankCases []string

	// Tier 0: Identifier exact match
	if hasNum {
		rankCases = append(rankCases, fmt.Sprintf("WHEN i.number = %s THEN 0", numParam))
	}

	// Tier 1: Exact title match
	rankCases = append(rankCases, fmt.Sprintf("WHEN LOWER(i.title) = %s THEN 1", phraseParam))

	// Tier 2: Title starts with phrase
	rankCases = append(rankCases, fmt.Sprintf("WHEN LOWER(i.title) LIKE %s THEN 2", phraseStartsWithParam))

	// Tier 3: Title contains phrase
	rankCases = append(rankCases, fmt.Sprintf("WHEN LOWER(i.title) LIKE %s THEN 3", phraseContainsParam))

	// Tier 4: Title matches all words (multi-word only)
	if len(termContainsParams) > 1 {
		var titleTerms []string
		for _, tp := range termContainsParams {
			titleTerms = append(titleTerms, fmt.Sprintf("LOWER(i.title) LIKE %s", tp))
		}
		rankCases = append(rankCases, fmt.Sprintf("WHEN (%s) THEN 4", strings.Join(titleTerms, " AND ")))
	}

	// Tier 5: Description contains phrase
	rankCases = append(rankCases, fmt.Sprintf("WHEN LOWER(COALESCE(i.description, '')) LIKE %s THEN 5", phraseContainsParam))

	// Tier 6: Description matches all words (multi-word only)
	if len(termContainsParams) > 1 {
		var descTerms []string
		for _, tp := range termContainsParams {
			descTerms = append(descTerms, fmt.Sprintf("LOWER(COALESCE(i.description, '')) LIKE %s", tp))
		}
		rankCases = append(rankCases, fmt.Sprintf("WHEN (%s) THEN 6", strings.Join(descTerms, " AND ")))
	}

	// Tier 7: Comment contains phrase. Same workspace_id-in-subquery
	// contract as the WHERE clause; see the phraseMatch comment above.
	rankCases = append(rankCases, fmt.Sprintf("WHEN EXISTS (SELECT 1 FROM comment c WHERE c.issue_id = i.id AND c.workspace_id = %s AND LOWER(c.content) LIKE %s) THEN 7", wsParam, phraseContainsParam))

	// Tier 8: Comment matches all words (multi-word only)
	if len(termContainsParams) > 1 {
		var commentTerms []string
		for _, tp := range termContainsParams {
			commentTerms = append(commentTerms, fmt.Sprintf("LOWER(c.content) LIKE %s", tp))
		}
		rankCases = append(rankCases, fmt.Sprintf("WHEN EXISTS (SELECT 1 FROM comment c WHERE c.issue_id = i.id AND c.workspace_id = %s AND (%s)) THEN 8", wsParam, strings.Join(commentTerms, " AND ")))
	}

	rankExpr := "CASE " + strings.Join(rankCases, " ") + " ELSE 9 END"

	// Status priority: active issues first
	statusRank := `CASE i.status
		WHEN 'in_progress' THEN 0
		WHEN 'in_review' THEN 1
		WHEN 'todo' THEN 2
		WHEN 'blocked' THEN 3
		WHEN 'backlog' THEN 4
		WHEN 'done' THEN 5
		WHEN 'cancelled' THEN 6
		ELSE 7
	END`

	// --- match_source expression ---
	matchSourceExpr := fmt.Sprintf(`CASE
		WHEN LOWER(i.title) LIKE %s THEN 'title'
		WHEN LOWER(COALESCE(i.description, '')) LIKE %s THEN 'description'
		ELSE 'comment'
	END`, phraseContainsParam, phraseContainsParam)

	// For multi-word: also check if all terms match in title/description
	if len(termContainsParams) > 1 {
		var titleTerms []string
		var descTerms []string
		for _, tp := range termContainsParams {
			titleTerms = append(titleTerms, fmt.Sprintf("LOWER(i.title) LIKE %s", tp))
			descTerms = append(descTerms, fmt.Sprintf("LOWER(COALESCE(i.description, '')) LIKE %s", tp))
		}
		matchSourceExpr = fmt.Sprintf(`CASE
			WHEN LOWER(i.title) LIKE %s THEN 'title'
			WHEN (%s) THEN 'title'
			WHEN LOWER(COALESCE(i.description, '')) LIKE %s THEN 'description'
			WHEN (%s) THEN 'description'
			ELSE 'comment'
		END`,
			phraseContainsParam, strings.Join(titleTerms, " AND "),
			phraseContainsParam, strings.Join(descTerms, " AND "),
		)
	}

	// --- matched_comment_content subquery ---
	// Always return matching comment content regardless of match_source,
	// so frontend can display comment snippet alongside title/description matches.
	// The c.workspace_id filter mirrors the WHERE clause: without it,
	// the planner can pick a global comment scan that ignores workspace
	// scoping.
	commentSubquery := fmt.Sprintf(`COALESCE(
		(SELECT c.content FROM comment c
		 WHERE c.issue_id = i.id AND c.workspace_id = %s AND LOWER(c.content) LIKE %s
		 ORDER BY c.created_at DESC LIMIT 1),
		''
	)`, wsParam, phraseContainsParam)

	if len(termContainsParams) > 1 {
		var commentTerms []string
		for _, tp := range termContainsParams {
			commentTerms = append(commentTerms, fmt.Sprintf("LOWER(c.content) LIKE %s", tp))
		}
		commentSubquery = fmt.Sprintf(`COALESCE(
			(SELECT c.content FROM comment c
			 WHERE c.issue_id = i.id AND c.workspace_id = %s AND (LOWER(c.content) LIKE %s OR (%s))
			 ORDER BY c.created_at DESC LIMIT 1),
			''
		)`, wsParam, phraseContainsParam, strings.Join(commentTerms, " AND "))
	}

	limitParam := nextArg(nil)  // placeholder
	offsetParam := nextArg(nil) // placeholder

	query := fmt.Sprintf(`SELECT i.id, i.workspace_id, i.title, i.description, i.status, i.priority,
		i.assignee_type, i.assignee_id, i.creator_type, i.creator_id,
		i.parent_issue_id, i.acceptance_criteria, i.context_refs, i.position,
		i.start_date, i.due_date, i.created_at, i.updated_at, i.number, i.project_id,
		COUNT(*) OVER() AS total_count,
		%s AS match_source,
		%s AS matched_comment_content
	FROM issue i
	WHERE i.workspace_id = %s AND %s
	ORDER BY %s, %s, i.updated_at DESC
	LIMIT %s OFFSET %s`,
		matchSourceExpr,
		commentSubquery,
		wsParam,
		whereClause,
		rankExpr,
		statusRank,
		limitParam,
		offsetParam,
	)

	return query, args
}

func (h *Handler) SearchIssues(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workspaceID := h.resolveWorkspaceID(r)

	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, "q parameter is required")
		return
	}

	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if limit > 50 {
		limit = 50
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	includeClosed := r.URL.Query().Get("include_closed") == "true"

	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace_id")
	if !ok {
		return
	}
	terms := splitSearchTerms(q)
	queryNum, hasNum := parseQueryNumber(q)

	sqlQuery, args := buildSearchQuery(q, terms, queryNum, hasNum, includeClosed)
	// Fill placeholder args: $4 = workspace_id, last two = limit, offset
	args[3] = wsUUID
	args[len(args)-2] = limit
	args[len(args)-1] = offset

	var results []searchResult
	err := runSearchQuery(ctx, h.TxStarter, sqlQuery, args, func(rows pgx.Rows) error {
		for rows.Next() {
			var sr searchResult
			if err := rows.Scan(
				&sr.issue.ID,
				&sr.issue.WorkspaceID,
				&sr.issue.Title,
				&sr.issue.Description,
				&sr.issue.Status,
				&sr.issue.Priority,
				&sr.issue.AssigneeType,
				&sr.issue.AssigneeID,
				&sr.issue.CreatorType,
				&sr.issue.CreatorID,
				&sr.issue.ParentIssueID,
				&sr.issue.AcceptanceCriteria,
				&sr.issue.ContextRefs,
				&sr.issue.Position,
				&sr.issue.StartDate,
				&sr.issue.DueDate,
				&sr.issue.CreatedAt,
				&sr.issue.UpdatedAt,
				&sr.issue.Number,
				&sr.issue.ProjectID,
				&sr.totalCount,
				&sr.matchSource,
				&sr.matchedCommentContent,
			); err != nil {
				return fmt.Errorf("scan: %w", err)
			}
			results = append(results, sr)
		}
		return rows.Err()
	})
	if err != nil {
		// Statement-timeout surfaces as SQLSTATE 57014. Return a 503
		// so the frontend can distinguish a timeout ("try a more
		// specific query") from a generic 500. This is the fail-fast
		// path when GIN search indexes are absent or the database is
		// overloaded; see runSearchQuery header for context.
		if isSearchStatementTimeout(err) {
			slog.Warn("search issues timed out",
				"workspace_id", workspaceID,
				"query", q,
				"timeout", searchStatementTimeout)
			writeError(w, http.StatusServiceUnavailable, "search timed out; please refine your query or try again")
			return
		}
		slog.Warn("search issues failed", "error", err, "workspace_id", workspaceID, "query", q)
		writeError(w, http.StatusInternalServerError, "failed to search issues")
		return
	}

	var total int64
	if len(results) > 0 {
		total = results[0].totalCount
	}

	prefix := h.getIssuePrefix(ctx, wsUUID)
	resp := make([]SearchIssueResponse, len(results))
	for i, sr := range results {
		sir := SearchIssueResponse{
			IssueResponse: issueToResponse(sr.issue, prefix),
			MatchSource:   sr.matchSource,
		}
		// Always populate comment snippet when a matching comment exists
		if sr.matchedCommentContent != "" {
			snippet := extractSnippet(sr.matchedCommentContent, q)
			sir.MatchedCommentSnippet = &snippet
			// Keep backward compat: also set MatchedSnippet for comment-source matches
			if sr.matchSource == "comment" {
				sir.MatchedSnippet = &snippet
			}
		}
		// Populate description snippet when description matches
		if sr.matchSource == "description" || descriptionContains(sr.issue.Description, q, terms) {
			if sr.issue.Description.Valid && sr.issue.Description.String != "" {
				snippet := extractSnippet(sr.issue.Description.String, q)
				sir.MatchedDescriptionSnippet = &snippet
			}
		}
		resp[i] = sir
	}

	w.Header().Set("X-Total-Count", strconv.FormatInt(total, 10))
	writeJSON(w, http.StatusOK, map[string]any{
		"issues": resp,
		"total":  total,
	})
}

// QueryIssues is the POST twin of ListIssues for filter sets too large for a
// GET request line — an explicit id facet can carry hundreds of
// issue ids, and common reverse proxies cap request lines around 8 KB. The
// body is a flat JSON object with EXACTLY the same keys and string encodings
// as ListIssues' query parameters; the handler rebuilds the query string and
// delegates, so the two transports cannot drift.
func (h *Handler) QueryIssues(w http.ResponseWriter, r *http.Request) {
	var params map[string]string
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&params); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	values := make(url.Values, len(params))
	for key, value := range params {
		values.Set(key, value)
	}
	r.URL.RawQuery = values.Encode()
	h.ListIssues(w, r)
}

func (h *Handler) ListIssues(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	workspaceID := h.resolveWorkspaceID(r)
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace_id")
	if !ok {
		return
	}

	// Parse optional filter params. Malformed UUIDs in filters return 400 —
	// silently coercing them to a zero UUID would mask a client bug and let
	// the query return an empty result set (or worse, match a NULL row).
	var priorityFilter pgtype.Text
	if p := r.URL.Query().Get("priority"); p != "" {
		priorityFilter = pgtype.Text{String: p, Valid: true}
	}
	var assigneeFilter pgtype.UUID
	if a := r.URL.Query().Get("assignee_id"); a != "" {
		id, ok := parseUUIDOrBadRequest(w, a, "assignee_id")
		if !ok {
			return
		}
		assigneeFilter = id
	}
	var assigneeIdsFilter []pgtype.UUID
	if ids := r.URL.Query().Get("assignee_ids"); ids != "" {
		for _, raw := range strings.Split(ids, ",") {
			if s := strings.TrimSpace(raw); s != "" {
				id, ok := parseUUIDOrBadRequest(w, s, "assignee_ids")
				if !ok {
					return
				}
				assigneeIdsFilter = append(assigneeIdsFilter, id)
			}
		}
	}
	var creatorFilter pgtype.UUID
	if c := r.URL.Query().Get("creator_id"); c != "" {
		id, ok := parseUUIDOrBadRequest(w, c, "creator_id")
		if !ok {
			return
		}
		creatorFilter = id
	}
	var projectFilter pgtype.UUID
	if p := r.URL.Query().Get("project_id"); p != "" {
		id, ok := parseUUIDOrBadRequest(w, p, "project_id")
		if !ok {
			return
		}
		projectFilter = id
	}
	metadataFilter, ok := parseMetadataFilterParam(w, r.URL.Query().Get("metadata"))
	if !ok {
		return
	}
	propertiesFilter, ok := parsePropertiesFilterParam(w, r.URL.Query().Get("properties"))
	if !ok {
		return
	}
	dateFilter, ok := parseIssueDateFilter(w, r.URL.Query())
	if !ok {
		return
	}

	// open_only=true returns all non-done/cancelled issues (no limit).
	if r.URL.Query().Get("open_only") == "true" {
		// Serialize the parsed AND-of-ORs groups into the single jsonb param
		// the static query unrolls (see properties_filter in ListOpenIssues).
		var openPropertiesFilter []byte
		if len(propertiesFilter) > 0 {
			marshaled, marshalErr := json.Marshal(propertiesFilter)
			if marshalErr != nil {
				writeError(w, http.StatusInternalServerError, "failed to list issues")
				return
			}
			openPropertiesFilter = marshaled
		}
		issues, err := h.Queries.ListOpenIssues(ctx, db.ListOpenIssuesParams{
			WorkspaceID:      wsUUID,
			Priority:         priorityFilter,
			AssigneeID:       assigneeFilter,
			AssigneeIds:      assigneeIdsFilter,
			CreatorID:        creatorFilter,
			ProjectID:        projectFilter,
			MetadataFilter:   metadataFilter,
			PropertiesFilter: openPropertiesFilter,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list issues")
			return
		}

		prefix := h.getIssuePrefix(ctx, wsUUID)
		ids := make([]pgtype.UUID, len(issues))
		for i, issue := range issues {
			ids[i] = issue.ID
		}
		labelsMap := h.labelsByIssue(ctx, wsUUID, ids)
		resp := make([]IssueResponse, len(issues))
		for i, issue := range issues {
			resp[i] = openIssueRowToResponse(issue, prefix)
			labels := labelsMap[resp[i].ID]
			if labels == nil {
				labels = []LabelResponse{}
			}
			resp[i].Labels = &labels
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"issues": resp,
			"total":  len(resp),
		})
		return
	}

	limit := 100
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if limit > 100 {
		limit = 100
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	statusesFilter := splitCommaParam(r.URL.Query().Get("statuses"))
	if len(statusesFilter) == 0 {
		statusesFilter = splitCommaParam(r.URL.Query().Get("status"))
	}
	prioritiesFilter := splitCommaParam(r.URL.Query().Get("priorities"))
	if len(prioritiesFilter) == 0 {
		prioritiesFilter = splitCommaParam(r.URL.Query().Get("priority"))
	}

	// assignee_types narrows the list to member-assigned issues.
	assigneeTypesFilter := splitCommaParam(r.URL.Query().Get("assignee_types"))
	for _, assigneeType := range assigneeTypesFilter {
		if !isIssueActorType(assigneeType) {
			writeError(w, http.StatusBadRequest, "invalid assignee_types")
			return
		}
	}

	// scheduled=true restricts the result to issues that have at least one of
	// start_date / due_date set. Used by the Project Gantt view, which only
	// renders schedulable rows and shouldn't pay for the full project list.
	var scheduledFilter pgtype.Bool
	if r.URL.Query().Get("scheduled") == "true" {
		scheduledFilter = pgtype.Bool{Bool: true, Valid: true}
	}

	// Parse sort and direction params for dynamic ORDER BY.
	// Manual sort (position) is always ASC — direction is ignored because
	// the user defines order through drag-and-drop, reversing it has no
	// product meaning.
	sortCol := "position"
	sortIsExpr := false
	sortIsProperty := false
	if s := r.URL.Query().Get("sort"); s != "" {
		switch s {
		case "position", "title", "created_at", "updated_at", "start_date", "due_date":
			sortCol = s
		case "status":
			sortCol = "CASE i.status WHEN 'backlog' THEN 0 WHEN 'todo' THEN 1 WHEN 'in_progress' THEN 2 WHEN 'in_review' THEN 3 WHEN 'done' THEN 4 WHEN 'blocked' THEN 5 WHEN 'cancelled' THEN 6 ELSE 7 END"
			sortIsExpr = true
		case "priority":
			sortCol = "CASE i.priority WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 ELSE 4 END"
			sortIsExpr = true
		default:
			// property:<definitionId> sorts by the custom-property value
			// (typed expression); unknown/archived definitions degrade to
			// position order instead of erroring stale clients.
			expr, handled, sortErr := h.propertySortExpr(r, workspaceID, s)
			if !handled {
				writeError(w, http.StatusBadRequest, "invalid sort value")
				return
			}
			if sortErr != nil {
				if sortErr.Error() == "invalid sort value" || sortErr.Error() == "invalid workspace id" {
					writeError(w, http.StatusBadRequest, sortErr.Error())
					return
				}
				slog.Warn("propertySortExpr failed", append(logger.RequestAttrs(r), "error", sortErr)...)
				writeError(w, http.StatusInternalServerError, "failed to resolve sort")
				return
			}
			if expr != "" {
				sortCol = expr
				sortIsExpr = true
				sortIsProperty = true
			}
		}
	}
	sortDir := "ASC"
	if sortCol != "position" {
		if d := r.URL.Query().Get("direction"); d != "" {
			switch strings.ToLower(d) {
			case "asc":
				sortDir = "ASC"
			case "desc":
				sortDir = "DESC"
			default:
				writeError(w, http.StatusBadRequest, "invalid direction value")
				return
			}
		}
	}

	// Build dynamic SQL — same approach as ListGroupedIssues.
	where := []string{"i.workspace_id = $1"}
	args := []any{wsUUID}
	addArg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}

	if len(statusesFilter) > 0 {
		where = append(where, fmt.Sprintf("i.status = ANY(%s::text[])", addArg(statusesFilter)))
	}
	if len(prioritiesFilter) > 0 {
		where = append(where, fmt.Sprintf("i.priority = ANY(%s::text[])", addArg(prioritiesFilter)))
	}
	if assigneeFilter.Valid {
		where = append(where, fmt.Sprintf("i.assignee_id = %s::uuid", addArg(assigneeFilter)))
	}
	if len(assigneeIdsFilter) > 0 {
		where = append(where, fmt.Sprintf("i.assignee_id = ANY(%s::uuid[])", addArg(assigneeIdsFilter)))
	}
	if len(assigneeTypesFilter) > 0 {
		where = append(where, fmt.Sprintf("i.assignee_type = ANY(%s::text[])", addArg(assigneeTypesFilter)))
	}
	if creatorFilter.Valid {
		where = append(where, fmt.Sprintf("i.creator_id = %s::uuid", addArg(creatorFilter)))
	}
	if projectFilter.Valid {
		where = append(where, fmt.Sprintf("i.project_id = %s::uuid", addArg(projectFilter)))
	}

	// Table facets must be part of the server window. Applying them after
	// LIMIT/OFFSET hides matches that live on later pages and makes `total`
	// disagree with the rows the user sees/exports.
	assigneeFilters, ok := parseActorFilterList(w, r.URL.Query().Get("assignee_filters"), "assignee_filters")
	if !ok {
		return
	}
	includeNoAssignee := r.URL.Query().Get("include_no_assignee") == "true"
	if len(assigneeFilters) > 0 || includeNoAssignee {
		ors := make([]string, 0, len(assigneeFilters)+1)
		for _, filter := range assigneeFilters {
			ors = append(ors, fmt.Sprintf(
				"(i.assignee_type = %s::text AND i.assignee_id = %s::uuid)",
				addArg(filter.actorType),
				addArg(filter.actorID),
			))
		}
		if includeNoAssignee {
			ors = append(ors, "(i.assignee_type IS NULL AND i.assignee_id IS NULL)")
		}
		where = append(where, "("+strings.Join(ors, " OR ")+")")
	}

	creatorFilters, ok := parseActorFilterList(w, r.URL.Query().Get("creator_filters"), "creator_filters")
	if !ok {
		return
	}
	if len(creatorFilters) > 0 {
		ors := make([]string, 0, len(creatorFilters))
		for _, filter := range creatorFilters {
			ors = append(ors, fmt.Sprintf(
				"(i.creator_type = %s::text AND i.creator_id = %s::uuid)",
				addArg(filter.actorType),
				addArg(filter.actorID),
			))
		}
		where = append(where, "("+strings.Join(ors, " OR ")+")")
	}

	projectIDs, ok := parseUUIDParamList(w, r.URL.Query().Get("project_ids"), "project_ids")
	if !ok {
		return
	}
	includeNoProject := r.URL.Query().Get("include_no_project") == "true"
	if len(projectIDs) > 0 || includeNoProject {
		ors := make([]string, 0, 2)
		if len(projectIDs) > 0 {
			ors = append(ors, fmt.Sprintf("i.project_id = ANY(%s::uuid[])", addArg(projectIDs)))
		}
		if includeNoProject {
			ors = append(ors, "i.project_id IS NULL")
		}
		where = append(where, "("+strings.Join(ors, " OR ")+")")
	}

	labelIDs, ok := parseUUIDParamList(w, r.URL.Query().Get("label_ids"), "label_ids")
	if !ok {
		return
	}
	if len(labelIDs) > 0 {
		where = append(where, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM issue_to_label itl WHERE itl.issue_id = i.id AND itl.label_id = ANY(%s::uuid[]))",
			addArg(labelIDs),
		))
	}
	// ids restricts the window to an explicit id set. Presence with an empty
	// list is meaningful and must yield an empty window.
	if r.URL.Query().Has("ids") {
		idsFilter, ok := parseUUIDParamList(w, r.URL.Query().Get("ids"), "ids")
		if !ok {
			return
		}
		if idsFilter == nil {
			idsFilter = []pgtype.UUID{}
		}
		where = append(where, fmt.Sprintf("i.id = ANY(%s::uuid[])", addArg(idsFilter)))
	}
	if r.URL.Query().Get("top_level_only") == "true" {
		where = append(where, "i.parent_issue_id IS NULL")
	}
	where = appendIssueTableSearchFilter(where, addArg, r.URL.Query().Get("q"))
	if scheduledFilter.Valid {
		where = append(where, "(i.start_date IS NOT NULL OR i.due_date IS NOT NULL)")
	}
	if metadataFilter != nil {
		where = append(where, fmt.Sprintf("i.metadata @> %s::jsonb", addArg(string(metadataFilter))))
	}
	if propertiesFilter != nil {
		where = append(where, propertiesFilterPredicate(propertiesFilter, addArg))
	}
	where = appendIssueDateFilter(where, addArg, dateFilter)
	whereSql := strings.Join(where, " AND ")

	// Build ORDER BY clause.
	orderBy := sortCol
	if !sortIsExpr {
		orderBy = "i." + sortCol
	}
	orderBy += " " + sortDir
	if sortCol == "start_date" || sortCol == "due_date" || sortIsProperty {
		// Property values are sparse: issues without one sort last in both
		// directions (mirrors the client comparator).
		orderBy += " NULLS LAST"
	}
	// created_at alone is not unique (bulk imports share timestamps); without
	// a unique final key the database may reorder ties between two
	// LIMIT/OFFSET requests, duplicating or dropping rows at page boundaries.
	orderBy += ", i.created_at DESC, i.id DESC"

	offsetRef := addArg(int64(offset))
	limitRef := addArg(int64(limit))

	query := fmt.Sprintf(`SELECT i.id, i.workspace_id, i.title, i.description, i.status, i.priority,
       i.assignee_type, i.assignee_id, i.creator_type, i.creator_id,
       i.parent_issue_id, i.position, i.start_date, i.due_date, i.created_at, i.updated_at, i.number, i.project_id, i.metadata, i.stage, i.properties
FROM issue i
WHERE %s
ORDER BY %s
LIMIT %s OFFSET %s`, whereSql, orderBy, limitRef, offsetRef)

	rows, err := h.DB.Query(ctx, query, args...)
	if err != nil {
		slog.Warn("ListIssues query failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list issues")
		return
	}
	defer rows.Close()

	var issues []db.ListIssuesRow
	for rows.Next() {
		var row db.ListIssuesRow
		if err := rows.Scan(
			&row.ID,
			&row.WorkspaceID,
			&row.Title,
			&row.Description,
			&row.Status,
			&row.Priority,
			&row.AssigneeType,
			&row.AssigneeID,
			&row.CreatorType,
			&row.CreatorID,
			&row.ParentIssueID,
			&row.Position,
			&row.StartDate,
			&row.DueDate,
			&row.CreatedAt,
			&row.UpdatedAt,
			&row.Number,
			&row.ProjectID,
			&row.Metadata,
			&row.Stage,
			&row.Properties,
		); err != nil {
			slog.Warn("ListIssues scan failed", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to list issues")
			return
		}
		issues = append(issues, row)
	}
	if err := rows.Err(); err != nil {
		slog.Warn("ListIssues rows failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list issues")
		return
	}

	// Get the true total count for pagination awareness.
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM issue i WHERE %s`, whereSql)
	// Count query uses the same args minus the OFFSET and LIMIT params (last two added).
	countArgs := args[:len(args)-2]
	var total int64
	if err := h.DB.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		total = int64(len(issues))
	}

	prefix := h.getIssuePrefix(ctx, wsUUID)
	ids := make([]pgtype.UUID, len(issues))
	for i, issue := range issues {
		ids[i] = issue.ID
	}
	labelsMap := h.labelsByIssue(ctx, wsUUID, ids)
	resp := make([]IssueResponse, len(issues))
	for i, issue := range issues {
		resp[i] = issueListRowToResponse(issue, prefix)
		labels := labelsMap[resp[i].ID]
		if labels == nil {
			labels = []LabelResponse{}
		}
		resp[i].Labels = &labels
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"issues": resp,
		"total":  total,
	})
}

type issueActorFilter struct {
	actorType string
	actorID   pgtype.UUID
}

type issueDateFilter struct {
	column string
	start  time.Time
	end    time.Time
}

func parseIssueDateFilter(w http.ResponseWriter, values url.Values) (*issueDateFilter, bool) {
	field := strings.TrimSpace(values.Get("date_field"))
	startRaw := strings.TrimSpace(values.Get("date_start"))
	endRaw := strings.TrimSpace(values.Get("date_end"))
	if field == "" && startRaw == "" && endRaw == "" {
		return nil, true
	}
	if field == "" || startRaw == "" || endRaw == "" {
		writeError(w, http.StatusBadRequest, "date_field, date_start, and date_end are required together")
		return nil, false
	}

	column := ""
	switch field {
	case "created_at":
		column = "created_at"
	case "updated_at":
		column = "updated_at"
	default:
		writeError(w, http.StatusBadRequest, "invalid date_field")
		return nil, false
	}

	start, err := time.Parse(time.RFC3339Nano, startRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid date_start")
		return nil, false
	}
	end, err := time.Parse(time.RFC3339Nano, endRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid date_end")
		return nil, false
	}
	if !start.Before(end) {
		writeError(w, http.StatusBadRequest, "date_start must be before date_end")
		return nil, false
	}

	return &issueDateFilter{column: column, start: start, end: end}, true
}

func appendIssueDateFilter(where []string, addArg func(any) string, filter *issueDateFilter) []string {
	if filter == nil {
		return where
	}
	startRef := addArg(filter.start)
	endRef := addArg(filter.end)
	return append(where, fmt.Sprintf(
		"i.%s >= %s AND i.%s < %s",
		filter.column,
		startRef,
		filter.column,
		endRef,
	))
}

// appendIssueTableSearchFilter adds a quick identity search to the ordinary
// ListIssues window. Unlike the ranked global search endpoint, this predicate
// preserves the table's active filters, explicit sort, total, and pagination.
// Every word must appear in the title; a complete identifier (or bare issue
// number) also matches the immutable numeric issue number.
func appendIssueTableSearchFilter(where []string, addArg func(any) string, raw string) []string {
	query := strings.TrimSpace(raw)
	if query == "" {
		return where
	}

	words := splitSearchTerms(strings.ToLower(query))
	ors := make([]string, 0, 2)
	if len(words) > 0 {
		titleMatches := make([]string, 0, len(words))
		for _, word := range words {
			pattern := "%" + escapeLike(word) + "%"
			titleMatches = append(titleMatches, fmt.Sprintf("LOWER(i.title) LIKE %s", addArg(pattern)))
		}
		ors = append(ors, "("+strings.Join(titleMatches, " AND ")+")")
	}
	if number, ok := parseQueryNumber(query); ok {
		ors = append(ors, fmt.Sprintf("i.number = %s", addArg(number)))
	}
	if len(ors) == 0 {
		return where
	}
	return append(where, "("+strings.Join(ors, " OR ")+")")
}

func splitCommaParam(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func isIssueActorType(s string) bool {
	return s == "member"
}

func parseUUIDParamList(w http.ResponseWriter, raw, fieldName string) ([]pgtype.UUID, bool) {
	parts := splitCommaParam(raw)
	if len(parts) == 0 {
		return nil, true
	}
	ids := make([]pgtype.UUID, 0, len(parts))
	for _, part := range parts {
		id, ok := parseUUIDOrBadRequest(w, part, fieldName)
		if !ok {
			return nil, false
		}
		ids = append(ids, id)
	}
	return ids, true
}

func parseActorFilterList(w http.ResponseWriter, raw, fieldName string) ([]issueActorFilter, bool) {
	parts := splitCommaParam(raw)
	if len(parts) == 0 {
		return nil, true
	}
	filters := make([]issueActorFilter, 0, len(parts))
	for _, part := range parts {
		pieces := strings.SplitN(part, ":", 2)
		if len(pieces) != 2 || !isIssueActorType(pieces[0]) || strings.TrimSpace(pieces[1]) == "" {
			writeError(w, http.StatusBadRequest, "invalid "+fieldName)
			return nil, false
		}
		id, ok := parseUUIDOrBadRequest(w, strings.TrimSpace(pieces[1]), fieldName)
		if !ok {
			return nil, false
		}
		filters = append(filters, issueActorFilter{
			actorType: pieces[0],
			actorID:   id,
		})
	}
	return filters, true
}

func (h *Handler) ListGroupedIssues(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.DB == nil {
		writeError(w, http.StatusInternalServerError, "database is unavailable")
		return
	}

	groupBy := r.URL.Query().Get("group_by")
	if groupBy == "" {
		groupBy = "assignee"
	}
	if groupBy != "assignee" {
		writeError(w, http.StatusBadRequest, "unsupported group_by")
		return
	}

	workspaceID := h.resolveWorkspaceID(r)
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace_id")
	if !ok {
		return
	}

	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if limit > 100 {
		limit = 100
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v > 0 {
			offset = v
		}
	}

	where := []string{"i.workspace_id = $1"}
	args := []any{wsUUID}
	addArg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}

	statuses := splitCommaParam(r.URL.Query().Get("statuses"))
	if len(statuses) == 0 {
		statuses = splitCommaParam(r.URL.Query().Get("status"))
	}
	if len(statuses) > 0 {
		where = append(where, fmt.Sprintf("i.status = ANY(%s::text[])", addArg(statuses)))
	}

	priorities := splitCommaParam(r.URL.Query().Get("priorities"))
	if len(priorities) == 0 {
		priorities = splitCommaParam(r.URL.Query().Get("priority"))
	}
	if len(priorities) > 0 {
		where = append(where, fmt.Sprintf("i.priority = ANY(%s::text[])", addArg(priorities)))
	}

	assigneeTypes := splitCommaParam(r.URL.Query().Get("assignee_types"))
	if len(assigneeTypes) > 0 {
		for _, assigneeType := range assigneeTypes {
			if !isIssueActorType(assigneeType) {
				writeError(w, http.StatusBadRequest, "invalid assignee_types")
				return
			}
		}
		where = append(where, fmt.Sprintf("i.assignee_type = ANY(%s::text[])", addArg(assigneeTypes)))
	}

	if raw := r.URL.Query().Get("assignee_id"); raw != "" {
		id, ok := parseUUIDOrBadRequest(w, raw, "assignee_id")
		if !ok {
			return
		}
		where = append(where, fmt.Sprintf("i.assignee_id = %s::uuid", addArg(id)))
	}
	if raw := r.URL.Query().Get("assignee_ids"); raw != "" {
		ids, ok := parseUUIDParamList(w, raw, "assignee_ids")
		if !ok {
			return
		}
		if len(ids) > 0 {
			where = append(where, fmt.Sprintf("i.assignee_id = ANY(%s::uuid[])", addArg(ids)))
		}
	}
	if raw := r.URL.Query().Get("creator_id"); raw != "" {
		id, ok := parseUUIDOrBadRequest(w, raw, "creator_id")
		if !ok {
			return
		}
		where = append(where, fmt.Sprintf("i.creator_id = %s::uuid", addArg(id)))
	}
	if raw := r.URL.Query().Get("project_id"); raw != "" {
		id, ok := parseUUIDOrBadRequest(w, raw, "project_id")
		if !ok {
			return
		}
		where = append(where, fmt.Sprintf("i.project_id = %s::uuid", addArg(id)))
	}
	if filter, ok := parseMetadataFilterParam(w, r.URL.Query().Get("metadata")); !ok {
		return
	} else if filter != nil {
		where = append(where, fmt.Sprintf("i.metadata @> %s::jsonb", addArg(string(filter))))
	}
	if filter, ok := parsePropertiesFilterParam(w, r.URL.Query().Get("properties")); !ok {
		return
	} else if filter != nil {
		where = append(where, propertiesFilterPredicate(filter, addArg))
	}
	assigneeFilters, ok := parseActorFilterList(w, r.URL.Query().Get("assignee_filters"), "assignee_filters")
	if !ok {
		return
	}
	includeNoAssignee := r.URL.Query().Get("include_no_assignee") == "true"
	if len(assigneeFilters) > 0 || includeNoAssignee {
		ors := make([]string, 0, len(assigneeFilters)+1)
		for _, filter := range assigneeFilters {
			ors = append(ors, fmt.Sprintf(
				"(i.assignee_type = %s::text AND i.assignee_id = %s::uuid)",
				addArg(filter.actorType),
				addArg(filter.actorID),
			))
		}
		if includeNoAssignee {
			ors = append(ors, "(i.assignee_type IS NULL AND i.assignee_id IS NULL)")
		}
		where = append(where, "("+strings.Join(ors, " OR ")+")")
	}

	creatorFilters, ok := parseActorFilterList(w, r.URL.Query().Get("creator_filters"), "creator_filters")
	if !ok {
		return
	}
	if len(creatorFilters) > 0 {
		ors := make([]string, 0, len(creatorFilters))
		for _, filter := range creatorFilters {
			ors = append(ors, fmt.Sprintf(
				"(i.creator_type = %s::text AND i.creator_id = %s::uuid)",
				addArg(filter.actorType),
				addArg(filter.actorID),
			))
		}
		where = append(where, "("+strings.Join(ors, " OR ")+")")
	}

	projectIDs, ok := parseUUIDParamList(w, r.URL.Query().Get("project_ids"), "project_ids")
	if !ok {
		return
	}
	includeNoProject := r.URL.Query().Get("include_no_project") == "true"
	if len(projectIDs) > 0 || includeNoProject {
		ors := make([]string, 0, 2)
		if len(projectIDs) > 0 {
			ors = append(ors, fmt.Sprintf("i.project_id = ANY(%s::uuid[])", addArg(projectIDs)))
		}
		if includeNoProject {
			ors = append(ors, "i.project_id IS NULL")
		}
		where = append(where, "("+strings.Join(ors, " OR ")+")")
	}

	labelIDs, ok := parseUUIDParamList(w, r.URL.Query().Get("label_ids"), "label_ids")
	if !ok {
		return
	}
	if len(labelIDs) > 0 {
		where = append(where, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM issue_to_label itl WHERE itl.issue_id = i.id AND itl.label_id = ANY(%s::uuid[]))",
			addArg(labelIDs),
		))
	}

	dateFilter, ok := parseIssueDateFilter(w, r.URL.Query())
	if !ok {
		return
	}
	where = appendIssueDateFilter(where, addArg, dateFilter)

	if groupAssigneeType := r.URL.Query().Get("group_assignee_type"); groupAssigneeType != "" {
		if groupAssigneeType == "none" {
			where = append(where, "(i.assignee_type IS NULL AND i.assignee_id IS NULL)")
		} else {
			if !isIssueActorType(groupAssigneeType) {
				writeError(w, http.StatusBadRequest, "invalid group_assignee_type")
				return
			}
			rawID := r.URL.Query().Get("group_assignee_id")
			if rawID == "" {
				writeError(w, http.StatusBadRequest, "invalid group_assignee_id")
				return
			}
			assigneeID, ok := parseUUIDOrBadRequest(w, rawID, "group_assignee_id")
			if !ok {
				return
			}
			where = append(where, fmt.Sprintf(
				"(i.assignee_type = %s::text AND i.assignee_id = %s::uuid)",
				addArg(groupAssigneeType),
				addArg(assigneeID),
			))
		}
	}

	sortCol := "position"
	sortIsExpr := false
	sortIsProperty := false
	if s := r.URL.Query().Get("sort"); s != "" {
		switch s {
		case "position", "title", "created_at", "updated_at", "start_date", "due_date":
			sortCol = s
		case "status":
			sortCol = "CASE i.status WHEN 'backlog' THEN 0 WHEN 'todo' THEN 1 WHEN 'in_progress' THEN 2 WHEN 'in_review' THEN 3 WHEN 'done' THEN 4 WHEN 'blocked' THEN 5 WHEN 'cancelled' THEN 6 ELSE 7 END"
			sortIsExpr = true
		case "priority":
			sortCol = "CASE i.priority WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 ELSE 4 END"
			sortIsExpr = true
		default:
			// property:<definitionId> sorts by the custom-property value
			// (typed expression); unknown/archived definitions degrade to
			// position order instead of erroring stale clients.
			expr, handled, sortErr := h.propertySortExpr(r, workspaceID, s)
			if !handled {
				writeError(w, http.StatusBadRequest, "invalid sort value")
				return
			}
			if sortErr != nil {
				if sortErr.Error() == "invalid sort value" || sortErr.Error() == "invalid workspace id" {
					writeError(w, http.StatusBadRequest, sortErr.Error())
					return
				}
				slog.Warn("propertySortExpr failed", append(logger.RequestAttrs(r), "error", sortErr)...)
				writeError(w, http.StatusInternalServerError, "failed to resolve sort")
				return
			}
			if expr != "" {
				sortCol = expr
				sortIsExpr = true
				sortIsProperty = true
			}
		}
	}
	sortDir := "ASC"
	if sortCol != "position" {
		if d := r.URL.Query().Get("direction"); d != "" {
			switch strings.ToLower(d) {
			case "asc":
				sortDir = "ASC"
			case "desc":
				sortDir = "DESC"
			default:
				writeError(w, http.StatusBadRequest, "invalid direction value")
				return
			}
		}
	}

	intraGroupOrder := sortCol
	if !sortIsExpr {
		intraGroupOrder = "i." + sortCol
	}
	intraGroupOrder += " " + sortDir
	if sortCol == "start_date" || sortCol == "due_date" || sortIsProperty {
		intraGroupOrder += " NULLS LAST"
	}
	// Unique final key — see ListIssues: created_at ties would otherwise make
	// ROW_NUMBER() unstable across per-group offset pages.
	intraGroupOrder += ", i.created_at DESC, i.id DESC"

	offsetRef := addArg(int64(offset))
	limitRef := addArg(int64(limit))
	query := fmt.Sprintf(`
WITH ranked AS (
	SELECT
		i.id, i.workspace_id, i.title, i.description, i.status, i.priority,
		i.assignee_type, i.assignee_id, i.creator_type, i.creator_id,
		i.parent_issue_id, i.position, i.start_date, i.due_date, i.created_at, i.updated_at,
		i.number, i.project_id, i.metadata, i.stage, i.properties,
		COUNT(*) OVER (PARTITION BY i.assignee_type, i.assignee_id) AS group_total,
		ROW_NUMBER() OVER (
			PARTITION BY i.assignee_type, i.assignee_id
			ORDER BY %s
		) AS rn
	FROM issue i
	WHERE %s
)
SELECT
	id, workspace_id, title, description, status, priority,
	assignee_type, assignee_id, creator_type, creator_id,
	parent_issue_id, position, start_date, due_date, created_at, updated_at,
	number, project_id, metadata, stage, properties, group_total
FROM ranked
WHERE rn > %s AND rn <= %s + %s
ORDER BY
	CASE assignee_type
		WHEN 'member' THEN 0
		ELSE 1
	END,
	assignee_type NULLS LAST,
	assignee_id NULLS LAST,
	rn`, intraGroupOrder, strings.Join(where, " AND "), offsetRef, offsetRef, limitRef)

	rows, err := h.DB.Query(ctx, query, args...)
	if err != nil {
		slog.Warn("ListGroupedIssues query failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list grouped issues")
		return
	}
	defer rows.Close()

	groupedRows := []groupedIssueRow{}
	for rows.Next() {
		var row groupedIssueRow
		if err := rows.Scan(
			&row.ID,
			&row.WorkspaceID,
			&row.Title,
			&row.Description,
			&row.Status,
			&row.Priority,
			&row.AssigneeType,
			&row.AssigneeID,
			&row.CreatorType,
			&row.CreatorID,
			&row.ParentIssueID,
			&row.Position,
			&row.StartDate,
			&row.DueDate,
			&row.CreatedAt,
			&row.UpdatedAt,
			&row.Number,
			&row.ProjectID,
			&row.Metadata,
			&row.Stage,
			&row.Properties,
			&row.GroupTotal,
		); err != nil {
			slog.Warn("ListGroupedIssues scan failed", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to list grouped issues")
			return
		}
		groupedRows = append(groupedRows, row)
	}
	if err := rows.Err(); err != nil {
		slog.Warn("ListGroupedIssues rows failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list grouped issues")
		return
	}

	ids := make([]pgtype.UUID, len(groupedRows))
	for i, row := range groupedRows {
		ids[i] = row.ID
	}
	labelsMap := h.labelsByIssue(ctx, wsUUID, ids)
	prefix := h.getIssuePrefix(ctx, wsUUID)

	groups := []IssueAssigneeGroupResponse{}
	groupIndex := map[string]int{}
	for _, row := range groupedRows {
		groupID := assigneeGroupID(row.AssigneeType, row.AssigneeID)
		idx, exists := groupIndex[groupID]
		if !exists {
			idx = len(groups)
			groupIndex[groupID] = idx
			groups = append(groups, IssueAssigneeGroupResponse{
				ID:           groupID,
				AssigneeType: textToPtr(row.AssigneeType),
				AssigneeID:   uuidToPtr(row.AssigneeID),
				Issues:       []IssueResponse{},
				Total:        row.GroupTotal,
			})
		}

		issue := issueListRowToResponse(row.ListIssuesRow, prefix)
		labels := labelsMap[issue.ID]
		if labels == nil {
			labels = []LabelResponse{}
		}
		issue.Labels = &labels
		groups[idx].Issues = append(groups[idx].Issues, issue)
	}

	writeJSON(w, http.StatusOK, GroupedIssuesResponse{Groups: groups})
}

func (h *Handler) GetIssue(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	issue, ok := h.loadIssueForUser(w, r, id)
	if !ok {
		return
	}
	prefix := h.getIssuePrefix(r.Context(), issue.WorkspaceID)
	resp := issueToResponse(issue, prefix)
	detailLabels := h.labelsByIssue(r.Context(), issue.WorkspaceID, []pgtype.UUID{issue.ID})[uuidToString(issue.ID)]
	if detailLabels == nil {
		detailLabels = []LabelResponse{}
	}
	resp.Labels = &detailLabels

	// Fetch issue reactions.
	reactions, err := h.Queries.ListIssueReactions(r.Context(), issue.ID)
	if err == nil && len(reactions) > 0 {
		resp.Reactions = make([]IssueReactionResponse, len(reactions))
		for i, rx := range reactions {
			resp.Reactions[i] = issueReactionToResponse(rx)
		}
	}

	// Fetch issue-level attachments.
	attachments, err := h.Queries.ListAttachmentsByIssue(r.Context(), db.ListAttachmentsByIssueParams{
		IssueID:     issue.ID,
		WorkspaceID: issue.WorkspaceID,
	})
	if err == nil && len(attachments) > 0 {
		mode := attachmentURLModeFromRequest(r)
		resp.Attachments = make([]AttachmentResponse, len(attachments))
		for i, a := range attachments {
			resp.Attachments[i] = h.attachmentToResponse(a, mode)
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListChildIssues(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	issue, ok := h.loadIssueForUser(w, r, id)
	if !ok {
		return
	}
	children, err := h.Queries.ListChildIssues(r.Context(), issue.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list child issues")
		return
	}
	prefix := h.getIssuePrefix(r.Context(), issue.WorkspaceID)
	ids := make([]pgtype.UUID, len(children))
	for i, child := range children {
		ids[i] = child.ID
	}
	labelsMap := h.labelsByIssue(r.Context(), issue.WorkspaceID, ids)
	resp := make([]IssueResponse, len(children))
	for i, child := range children {
		resp[i] = issueToResponse(child, prefix)
		labels := labelsMap[resp[i].ID]
		if labels == nil {
			labels = []LabelResponse{}
		}
		resp[i].Labels = &labels
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"issues": resp,
	})
}

// Cap on the number of parents we'll fan-out children for in one request.
// Swimlane's visible-lane count is naturally bounded by what fits on screen
// (typically <= 50), but cap explicitly so a malicious caller can't ANY()
// across the whole workspace's issue set in a single round trip.
const listChildrenByParentsLimit = 200

// ListChildrenByParents returns the union of children for the
// provided parent ids. Replaces the N-call fan-out Swimlane would otherwise
// have to make on mount (one /issues/:id/children per visible parent lane).
//
// Workspace scope is enforced at the query level — any parent_id that doesn't
// belong to the caller's workspace simply yields zero children, so callers
// can't probe parents across workspace boundaries.
func (h *Handler) ListChildrenByParents(w http.ResponseWriter, r *http.Request) {
	workspaceID := h.resolveWorkspaceID(r)
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace_id")
	if !ok {
		return
	}

	raw := r.URL.Query().Get("parent_ids")
	if raw == "" {
		// Empty input is a no-op response (not an error) — simplifies the
		// client which calls this unconditionally on Swimlane mount even
		// when there are zero visible parent lanes.
		writeJSON(w, http.StatusOK, map[string]any{"issues": []IssueResponse{}})
		return
	}

	parts := strings.Split(raw, ",")
	if len(parts) > listChildrenByParentsLimit {
		writeError(w, http.StatusBadRequest, "too many parent_ids")
		return
	}
	parentIDs := make([]pgtype.UUID, 0, len(parts))
	for _, s := range parts {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, ok := parseUUIDOrBadRequest(w, s, "parent_ids")
		if !ok {
			return
		}
		parentIDs = append(parentIDs, id)
	}
	if len(parentIDs) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"issues": []IssueResponse{}})
		return
	}

	children, err := h.Queries.ListChildrenByParents(r.Context(), db.ListChildrenByParentsParams{
		WorkspaceID: wsUUID,
		ParentIds:   parentIDs,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list child issues")
		return
	}
	prefix := h.getIssuePrefix(r.Context(), wsUUID)
	ids := make([]pgtype.UUID, len(children))
	for i, child := range children {
		ids[i] = child.ID
	}
	labelsMap := h.labelsByIssue(r.Context(), wsUUID, ids)
	resp := make([]IssueResponse, len(children))
	for i, child := range children {
		resp[i] = issueToResponse(child, prefix)
		labels := labelsMap[resp[i].ID]
		if labels == nil {
			labels = []LabelResponse{}
		}
		resp[i].Labels = &labels
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"issues": resp,
	})
}

func (h *Handler) ChildIssueProgress(w http.ResponseWriter, r *http.Request) {
	wsID := h.resolveWorkspaceID(r)
	wsUUID, ok := parseUUIDOrBadRequest(w, wsID, "workspace_id")
	if !ok {
		return
	}

	rows, err := h.Queries.ChildIssueProgress(r.Context(), wsUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get child issue progress")
		return
	}

	type progressEntry struct {
		ParentIssueID string `json:"parent_issue_id"`
		Total         int64  `json:"total"`
		Done          int64  `json:"done"`
	}
	resp := make([]progressEntry, len(rows))
	for i, row := range rows {
		resp[i] = progressEntry{
			ParentIssueID: uuidToString(row.ParentIssueID),
			Total:         row.Total,
			Done:          row.Done,
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"progress": resp,
	})
}

type CreateIssueRequest struct {
	Title         string   `json:"title"`
	Description   *string  `json:"description"`
	Status        string   `json:"status"`
	Priority      string   `json:"priority"`
	AssigneeType  *string  `json:"assignee_type"`
	AssigneeID    *string  `json:"assignee_id"`
	ParentIssueID *string  `json:"parent_issue_id"`
	ProjectID     *string  `json:"project_id"`
	Stage         *int32   `json:"stage,omitempty"`
	StartDate     *string  `json:"start_date"`
	DueDate       *string  `json:"due_date"`
	AttachmentIDs []string `json:"attachment_ids,omitempty"`
	// LabelIDs are issue-scoped labels to attach to the new issue in the same
	// transaction as the create. Unknown or non-issue ids are rejected with
	// 400 (service.ErrIssueLabelNotFound) rather than silently dropped.
	LabelIDs       []string `json:"label_ids,omitempty"`
	AllowDuplicate bool     `json:"allow_duplicate,omitempty"`
}

func duplicateIssueMessage(issue IssueResponse) string {
	return issueguard.DuplicateMessage(issue.Identifier, issue.Title, issue.Status)
}

func (h *Handler) CreateIssue(w http.ResponseWriter, r *http.Request) {
	var req CreateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	workspaceID := h.resolveWorkspaceID(r)
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace_id")
	if !ok {
		return
	}

	// Get creator from context (set by auth middleware)
	creatorID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	status := req.Status
	if status == "" {
		status = "todo"
	}
	priority := req.Priority
	if priority == "" {
		priority = "none"
	}
	if !validateIssueEnum(w, "status", status, validIssueStatuses) {
		return
	}
	if !validateIssueEnum(w, "priority", priority, validIssuePriorities) {
		return
	}
	if req.Stage != nil && *req.Stage < 1 {
		writeError(w, http.StatusBadRequest, "stage must be >= 1")
		return
	}

	var assigneeType pgtype.Text
	var assigneeID pgtype.UUID
	if req.AssigneeType != nil {
		assigneeType = pgtype.Text{String: *req.AssigneeType, Valid: true}
	}
	if req.AssigneeID != nil {
		id, ok := parseUUIDOrBadRequest(w, *req.AssigneeID, "assignee_id")
		if !ok {
			return
		}
		assigneeID = id
	}

	if status, msg := h.validateAssigneePair(r.Context(), r, workspaceID, assigneeType, assigneeID); status != 0 {
		writeError(w, status, msg)
		return
	}

	var parentIssueID pgtype.UUID
	var projectID pgtype.UUID
	if req.ProjectID != nil {
		id, ok := parseUUIDOrBadRequest(w, *req.ProjectID, "project_id")
		if !ok {
			return
		}
		projectID = id
	}
	if req.ParentIssueID != nil {
		id, ok := parseUUIDOrBadRequest(w, *req.ParentIssueID, "parent_issue_id")
		if !ok {
			return
		}
		parentIssueID = id
	}
	// Cross-workspace parent and project validation is enforced inside the
	// create transaction.

	attachmentIDs, ok := parseUUIDSliceOrBadRequest(w, req.AttachmentIDs, "attachment_ids")
	if !ok {
		return
	}

	labelIDs, ok := parseUUIDSliceOrBadRequest(w, req.LabelIDs, "label_ids")
	if !ok {
		return
	}

	var startDate pgtype.Date
	if req.StartDate != nil && *req.StartDate != "" {
		d, err := util.ParseCalendarDate(*req.StartDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid start_date format, expected YYYY-MM-DD")
			return
		}
		startDate = d
	}

	var dueDate pgtype.Date
	if req.DueDate != nil && *req.DueDate != "" {
		d, err := util.ParseCalendarDate(*req.DueDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid due_date format, expected YYYY-MM-DD")
			return
		}
		dueDate = d
	}

	creatorType, actualCreatorID := h.resolveActor(r, creatorID, workspaceID)

	// Prefix is workspace-level; pre-compute once so both the broadcast
	// payload builder and the HTTP response share the same value.
	prefix := h.getIssuePrefix(r.Context(), wsUUID)

	attachmentMode := attachmentURLModeFromRequest(r)
	buildAttachmentResponses := func(atts []db.Attachment) []AttachmentResponse {
		if len(atts) == 0 {
			return nil
		}
		out := make([]AttachmentResponse, len(atts))
		for i, a := range atts {
			out[i] = h.attachmentToResponse(a, attachmentMode)
		}
		return out
	}

	res, err := h.IssueService.Create(r.Context(), service.IssueCreateParams{
		WorkspaceID:    wsUUID,
		Title:          req.Title,
		Description:    ptrToText(req.Description),
		Status:         status,
		Priority:       priority,
		AssigneeType:   assigneeType,
		AssigneeID:     assigneeID,
		CreatorType:    creatorType,
		CreatorID:      parseUUID(actualCreatorID),
		ParentIssueID:  parentIssueID,
		ProjectID:      projectID,
		StartDate:      startDate,
		DueDate:        dueDate,
		Stage:          ptrToInt4(req.Stage),
		AttachmentIDs:  attachmentIDs,
		LabelIDs:       labelIDs,
		AllowDuplicate: req.AllowDuplicate,
	}, service.IssueCreateOpts{
		ActorID: actualCreatorID,
		BeforeCommit: func(ctx context.Context, tx pgx.Tx, issue db.Issue) error {
			return h.enqueueKnowledgeEvidence(
				ctx,
				tx,
				issueKnowledgeEvidence(issue, actualCreatorID, "issue.created"),
			)
		},
		BroadcastPayload: func(issue db.Issue, atts []db.Attachment, labels []db.IssueLabel) map[string]any {
			payload := issueToResponse(issue, prefix)
			payload.Attachments = buildAttachmentResponses(atts)
			// Carry the authoritative label snapshot so every online client
			// renders the new issue already labeled. Non-nil (even empty)
			// pointer = authoritative list; the old flow's separate
			// issue_labels:changed broadcast is gone.
			labelResponses := labelsToResponse(labels)
			payload.Labels = &labelResponses
			return map[string]any{"issue": payload}
		},
	})

	if errors.Is(err, service.ErrActiveDuplicate) {
		dup := *res.DuplicateIssue
		existing := issueToResponse(dup, h.getIssuePrefix(r.Context(), dup.WorkspaceID))
		writeJSON(w, http.StatusConflict, map[string]any{
			"code":  "active_duplicate_issue",
			"error": duplicateIssueMessage(existing),
			"issue": existing,
		})
		return
	}
	if errors.Is(err, service.ErrParentIssueNotFound) {
		writeError(w, http.StatusBadRequest, "parent issue not found in this workspace")
		return
	}
	if errors.Is(err, service.ErrProjectNotFound) {
		writeError(w, http.StatusBadRequest, "project not found in this workspace")
		return
	}
	if errors.Is(err, service.ErrIssueLabelNotFound) {
		writeError(w, http.StatusBadRequest, "one or more labels not found in this workspace")
		return
	}
	if err != nil {
		slog.Warn("create issue failed", append(logger.RequestAttrs(r), "error", err, "workspace_id", workspaceID)...)
		writeError(w, http.StatusInternalServerError, "failed to create issue: "+err.Error())
		return
	}

	issue := res.Issue
	slog.Info("issue created", append(logger.RequestAttrs(r), "issue_id", uuidToString(issue.ID), "title", issue.Title, "status", issue.Status, "workspace_id", workspaceID)...)

	resp := issueToResponse(issue, prefix)
	resp.Attachments = buildAttachmentResponses(res.Attachments)
	// Echo the authoritative labels attached in the create transaction. Always
	// non-nil (empty slice when none) so a newer client can tell the backend
	// understood label_ids and skip its legacy post-create attach fallback.
	labelResponses := labelsToResponse(res.Labels)
	resp.Labels = &labelResponses
	writeJSON(w, http.StatusCreated, resp)
}

type UpdateIssueRequest struct {
	Title         *string  `json:"title"`
	Description   *string  `json:"description"`
	Status        *string  `json:"status"`
	Priority      *string  `json:"priority"`
	AssigneeType  *string  `json:"assignee_type"`
	AssigneeID    *string  `json:"assignee_id"`
	Position      *float64 `json:"position"`
	StartDate     *string  `json:"start_date"`
	DueDate       *string  `json:"due_date"`
	ParentIssueID *string  `json:"parent_issue_id"`
	ProjectID     *string  `json:"project_id"`
	Stage         *int32   `json:"stage"`
	// AttachmentIDs lets the description editor bind newly uploaded files to
	// this issue so they surface in `GET /api/issues/:id/attachments` and the
	// editor's preview Eye keeps working past a refresh. Existing bindings
	// are idempotent — re-sending the same id is a no-op.
	AttachmentIDs []string `json:"attachment_ids"`
}

func (h *Handler) UpdateIssue(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	prevIssue, ok := h.loadIssueForUser(w, r, id)
	if !ok {
		return
	}
	userID := requestUserID(r)
	workspaceID := uuidToString(prevIssue.WorkspaceID)

	// Read body as raw bytes so we can detect which fields were explicitly sent.
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req UpdateIssueRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Track which fields were explicitly present in JSON (even if null)
	var rawFields map[string]json.RawMessage
	json.Unmarshal(bodyBytes, &rawFields)

	// Pre-fill nullable fields (bare sqlc.narg) with current values
	params := db.UpdateIssueParams{
		ID:            prevIssue.ID,
		AssigneeType:  prevIssue.AssigneeType,
		AssigneeID:    prevIssue.AssigneeID,
		StartDate:     prevIssue.StartDate,
		DueDate:       prevIssue.DueDate,
		ParentIssueID: prevIssue.ParentIssueID,
		ProjectID:     prevIssue.ProjectID,
		Stage:         prevIssue.Stage,
	}

	// COALESCE fields — only set when explicitly provided
	if req.Title != nil {
		params.Title = pgtype.Text{String: *req.Title, Valid: true}
	}
	if req.Description != nil {
		params.Description = pgtype.Text{String: *req.Description, Valid: true}
	}
	if req.Status != nil {
		if !validateIssueEnum(w, "status", *req.Status, validIssueStatuses) {
			return
		}
		params.Status = pgtype.Text{String: *req.Status, Valid: true}
	}
	if req.Priority != nil {
		if !validateIssueEnum(w, "priority", *req.Priority, validIssuePriorities) {
			return
		}
		params.Priority = pgtype.Text{String: *req.Priority, Valid: true}
	}
	if req.Position != nil {
		params.Position = pgtype.Float8{Float64: *req.Position, Valid: true}
	}
	// Nullable fields — only override when explicitly present in JSON
	if _, ok := rawFields["assignee_type"]; ok {
		if req.AssigneeType != nil {
			params.AssigneeType = pgtype.Text{String: *req.AssigneeType, Valid: true}
		} else {
			params.AssigneeType = pgtype.Text{Valid: false} // explicit null = unassign
		}
	}
	if _, ok := rawFields["assignee_id"]; ok {
		if req.AssigneeID != nil {
			id, ok := parseUUIDOrBadRequest(w, *req.AssigneeID, "assignee_id")
			if !ok {
				return
			}
			params.AssigneeID = id
		} else {
			params.AssigneeID = pgtype.UUID{Valid: false} // explicit null = unassign
		}
	}
	if _, ok := rawFields["start_date"]; ok {
		if req.StartDate != nil && *req.StartDate != "" {
			d, err := util.ParseCalendarDate(*req.StartDate)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid start_date format, expected YYYY-MM-DD")
				return
			}
			params.StartDate = d
		} else {
			params.StartDate = pgtype.Date{Valid: false} // explicit null = clear date
		}
	}
	if _, ok := rawFields["due_date"]; ok {
		if req.DueDate != nil && *req.DueDate != "" {
			d, err := util.ParseCalendarDate(*req.DueDate)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid due_date format, expected YYYY-MM-DD")
				return
			}
			params.DueDate = d
		} else {
			params.DueDate = pgtype.Date{Valid: false} // explicit null = clear date
		}
	}
	if _, ok := rawFields["parent_issue_id"]; ok {
		if req.ParentIssueID != nil {
			newParentID, ok := parseUUIDOrBadRequest(w, *req.ParentIssueID, "parent_issue_id")
			if !ok {
				return
			}
			// Cannot set self as parent. Compare against prevIssue.ID (the
			// resolved entity), not the raw URL string — `id` may be an
			// identifier like "MUL-7".
			if newParentID == prevIssue.ID {
				writeError(w, http.StatusBadRequest, "an issue cannot be its own parent")
				return
			}
			// Validate parent exists in the same workspace.
			if _, err := h.Queries.GetIssueInWorkspace(r.Context(), db.GetIssueInWorkspaceParams{
				ID:          newParentID,
				WorkspaceID: prevIssue.WorkspaceID,
			}); err != nil {
				writeError(w, http.StatusBadRequest, "parent issue not found in this workspace")
				return
			}
			// Cycle detection: walk up from the new parent to ensure we don't reach this issue.
			cursor := newParentID
			for depth := 0; depth < 10; depth++ {
				ancestor, err := h.Queries.GetIssue(r.Context(), cursor)
				if err != nil || !ancestor.ParentIssueID.Valid {
					break
				}
				if ancestor.ParentIssueID == prevIssue.ID {
					writeError(w, http.StatusBadRequest, "circular parent relationship detected")
					return
				}
				cursor = ancestor.ParentIssueID
			}
			params.ParentIssueID = newParentID
		} else {
			params.ParentIssueID = pgtype.UUID{Valid: false} // explicit null = remove parent
		}
	}
	if _, ok := rawFields["project_id"]; ok {
		if req.ProjectID != nil {
			projectUUID, ok := parseUUIDOrBadRequest(w, *req.ProjectID, "project_id")
			if !ok {
				return
			}
			params.ProjectID = projectUUID
		} else {
			params.ProjectID = pgtype.UUID{Valid: false}
		}
	}
	if _, ok := rawFields["stage"]; ok {
		if req.Stage != nil {
			if *req.Stage < 1 {
				writeError(w, http.StatusBadRequest, "stage must be >= 1")
				return
			}
			params.Stage = pgtype.Int4{Int32: *req.Stage, Valid: true}
		} else {
			params.Stage = pgtype.Int4{Valid: false} // explicit null = unstage
		}
	}

	// Validate the resulting (assignee_type, assignee_id) pair when the caller
	// touches either field. Existing data on the issue is left alone if the
	// caller is not changing it.
	_, touchedType := rawFields["assignee_type"]
	_, touchedID := rawFields["assignee_id"]
	if touchedType || touchedID {
		if status, msg := h.validateAssigneePair(r.Context(), r, workspaceID, params.AssigneeType, params.AssigneeID); status != 0 {
			writeError(w, status, msg)
			return
		}
	}

	attachmentIDs, ok := parseUUIDSliceOrBadRequest(w, req.AttachmentIDs, "attachment_ids")
	if !ok {
		return
	}

	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start issue transaction")
		return
	}
	defer tx.Rollback(r.Context())
	qtx := h.Queries.WithTx(tx)
	issue, err := qtx.UpdateIssue(r.Context(), params)
	if err != nil {
		slog.Warn("update issue failed", append(logger.RequestAttrs(r), "error", err, "issue_id", id, "workspace_id", workspaceID)...)
		writeError(w, http.StatusInternalServerError, "failed to update issue: "+err.Error())
		return
	}
	if prevIssue.Status != "done" && issue.Status == "done" {
		if err := h.enqueueKnowledgeEvidence(
			r.Context(),
			tx,
			issueKnowledgeEvidence(issue, userID, "issue.accepted"),
		); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to record issue evidence")
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit issue update")
		return
	}

	if len(attachmentIDs) > 0 {
		h.linkAttachmentsByIssueIDs(r.Context(), issue.ID, issue.WorkspaceID, attachmentIDs)
	}

	prefix := h.getIssuePrefix(r.Context(), issue.WorkspaceID)
	resp := issueToResponse(issue, prefix)
	slog.Info("issue updated", append(logger.RequestAttrs(r), "issue_id", id, "workspace_id", workspaceID)...)

	assigneeChanged := (req.AssigneeType != nil || req.AssigneeID != nil) &&
		(prevIssue.AssigneeType.String != issue.AssigneeType.String || uuidToString(prevIssue.AssigneeID) != uuidToString(issue.AssigneeID))
	statusChanged := req.Status != nil && prevIssue.Status != issue.Status
	priorityChanged := req.Priority != nil && prevIssue.Priority != issue.Priority
	// project_changed gates the client's per-project issue-list refetch the way
	// status/assignee flags gate theirs. Without it the client must diff
	// project_id against its own cache, which breaks once an optimistic local
	// move has overwritten the cached value (MUL-3669 / #4548).
	projectChanged := req.ProjectID != nil && uuidToString(prevIssue.ProjectID) != uuidToString(issue.ProjectID)
	descriptionChanged := req.Description != nil && textToPtr(prevIssue.Description) != resp.Description
	titleChanged := req.Title != nil && prevIssue.Title != issue.Title
	prevStartDate := dateToPtr(prevIssue.StartDate)
	startDateChanged := prevStartDate != resp.StartDate && (prevStartDate == nil) != (resp.StartDate == nil) ||
		(prevStartDate != nil && resp.StartDate != nil && *prevStartDate != *resp.StartDate)
	prevDueDate := dateToPtr(prevIssue.DueDate)
	dueDateChanged := prevDueDate != resp.DueDate && (prevDueDate == nil) != (resp.DueDate == nil) ||
		(prevDueDate != nil && resp.DueDate != nil && *prevDueDate != *resp.DueDate)

	actorType, actorID := h.resolveActor(r, userID, workspaceID)

	h.publish(protocol.EventIssueUpdated, workspaceID, actorType, actorID, map[string]any{
		"issue":               resp,
		"assignee_changed":    assigneeChanged,
		"status_changed":      statusChanged,
		"priority_changed":    priorityChanged,
		"project_changed":     projectChanged,
		"start_date_changed":  startDateChanged,
		"due_date_changed":    dueDateChanged,
		"description_changed": descriptionChanged,
		"title_changed":       titleChanged,
		"prev_title":          prevIssue.Title,
		"prev_assignee_type":  textToPtr(prevIssue.AssigneeType),
		"prev_assignee_id":    uuidToPtr(prevIssue.AssigneeID),
		"prev_status":         prevIssue.Status,
		"prev_priority":       prevIssue.Priority,
		"prev_start_date":     prevStartDate,
		"prev_due_date":       prevDueDate,
		"prev_description":    textToPtr(prevIssue.Description),
		"creator_type":        prevIssue.CreatorType,
		"creator_id":          uuidToString(prevIssue.CreatorID),
	})

	writeJSON(w, http.StatusOK, resp)
}

// validateAssigneePair verifies that the assignee is a workspace member.
// Returns (statusCode, errorMessage). statusCode == 0 means the pair is valid;
// callers should treat any non-zero status as a rejection and surface it back
// to the client.
func (h *Handler) validateAssigneePair(ctx context.Context, r *http.Request, workspaceID string, assigneeType pgtype.Text, assigneeID pgtype.UUID) (int, string) {
	// Both unset → unassigned issue, valid.
	if !assigneeType.Valid && !assigneeID.Valid {
		return 0, ""
	}
	// Exactly one of type/id provided → callers must always pair them.
	if assigneeType.Valid != assigneeID.Valid {
		return http.StatusBadRequest, "assignee_type and assignee_id must be provided together"
	}
	wsUUID, err := util.ParseUUID(workspaceID)
	if err != nil {
		return http.StatusBadRequest, "invalid workspace_id"
	}
	switch assigneeType.String {
	case "member":
		if _, err := h.Queries.GetMemberByUserAndWorkspace(ctx, db.GetMemberByUserAndWorkspaceParams{
			UserID:      assigneeID,
			WorkspaceID: wsUUID,
		}); err != nil {
			return http.StatusBadRequest, "assignee_id does not refer to a member of this workspace"
		}
		return 0, ""
	default:
		return http.StatusBadRequest, "assignee_type must be 'member'"
	}
}

func (h *Handler) DeleteIssue(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	issue, ok := h.loadIssueForUser(w, r, id)
	if !ok {
		return
	}

	// Collect all attachment URLs (issue-level + comment-level) before CASCADE delete.
	attachmentURLs, _ := h.Queries.ListAttachmentURLsByIssueOrComments(r.Context(), issue.ID)

	err := h.Queries.DeleteIssue(r.Context(), db.DeleteIssueParams{
		ID:          issue.ID,
		WorkspaceID: issue.WorkspaceID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete issue")
		return
	}

	h.deleteS3Objects(r.Context(), attachmentURLs)
	userID := requestUserID(r)
	actorType, actorID := h.resolveActor(r, userID, uuidToString(issue.WorkspaceID))
	// Always emit the resolved UUID — frontend caches key by UUID, so an
	// identifier-style payload ("MUL-123") would leave stale entries on
	// other clients after an identifier-path delete.
	resolvedID := uuidToString(issue.ID)
	h.publish(protocol.EventIssueDeleted, uuidToString(issue.WorkspaceID), actorType, actorID, map[string]any{"issue_id": resolvedID})
	slog.Info("issue deleted", append(logger.RequestAttrs(r), "issue_id", resolvedID, "workspace_id", uuidToString(issue.WorkspaceID))...)
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Batch operations
// ---------------------------------------------------------------------------

type BatchUpdateIssuesRequest struct {
	IssueIDs []string           `json:"issue_ids"`
	Updates  UpdateIssueRequest `json:"updates"`
}

func (h *Handler) BatchUpdateIssues(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req BatchUpdateIssuesRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.IssueIDs) == 0 {
		writeError(w, http.StatusBadRequest, "issue_ids is required")
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	// Detect which fields in "updates" were explicitly set (including null).
	var rawTop map[string]json.RawMessage
	json.Unmarshal(bodyBytes, &rawTop)
	var rawUpdates map[string]json.RawMessage
	if raw, exists := rawTop["updates"]; exists {
		json.Unmarshal(raw, &rawUpdates)
	}

	// Short-circuit when no mutation field is present in `updates`. Without
	// this, the loop below runs N no-op UPDATEs (every if-guard skips, every
	// COALESCE preserves the existing value) and reports `{"updated": N}` —
	// the response cheerfully claims success while nothing changed. Most
	// real-world cases that hit this path are caller mistakes (status placed
	// at the top level, "update" misspelled as singular). Telling the truth
	// here — `{"updated": 0}` — keeps the wire shape stable while making the
	// count match reality. See multica-ai/multica#1660.
	hasMutation := req.Updates.Title != nil ||
		req.Updates.Description != nil ||
		req.Updates.Status != nil ||
		req.Updates.Priority != nil ||
		req.Updates.Position != nil
	if !hasMutation {
		for _, k := range []string{"assignee_type", "assignee_id", "start_date", "due_date", "parent_issue_id", "project_id", "stage"} {
			if _, ok := rawUpdates[k]; ok {
				hasMutation = true
				break
			}
		}
	}
	if !hasMutation {
		writeJSON(w, http.StatusOK, map[string]any{"updated": 0})
		return
	}
	if req.Updates.Status != nil {
		if !validateIssueEnum(w, "status", *req.Updates.Status, validIssueStatuses) {
			return
		}
	}
	if req.Updates.Priority != nil {
		if !validateIssueEnum(w, "priority", *req.Updates.Priority, validIssuePriorities) {
			return
		}
	}

	workspaceID := h.resolveWorkspaceID(r)
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace_id")
	if !ok {
		return
	}
	updated := 0
	// Children that transitioned into a terminal status this batch, collected so
	// the parent/stage notification is evaluated once against the final state
	// after the loop (MUL-4155) rather than per-child mid-batch.
	for _, issueID := range req.IssueIDs {
		issueUUID, err := util.ParseUUID(issueID)
		if err != nil {
			continue
		}
		prevIssue, err := h.Queries.GetIssueInWorkspace(r.Context(), db.GetIssueInWorkspaceParams{
			ID:          issueUUID,
			WorkspaceID: wsUUID,
		})
		if err != nil {
			continue
		}

		params := db.UpdateIssueParams{
			ID:            prevIssue.ID,
			AssigneeType:  prevIssue.AssigneeType,
			AssigneeID:    prevIssue.AssigneeID,
			StartDate:     prevIssue.StartDate,
			DueDate:       prevIssue.DueDate,
			ParentIssueID: prevIssue.ParentIssueID,
			ProjectID:     prevIssue.ProjectID,
			Stage:         prevIssue.Stage,
		}

		if req.Updates.Title != nil {
			params.Title = pgtype.Text{String: *req.Updates.Title, Valid: true}
		}
		if req.Updates.Description != nil {
			params.Description = pgtype.Text{String: *req.Updates.Description, Valid: true}
		}
		if req.Updates.Status != nil {
			params.Status = pgtype.Text{String: *req.Updates.Status, Valid: true}
		}
		if req.Updates.Priority != nil {
			params.Priority = pgtype.Text{String: *req.Updates.Priority, Valid: true}
		}
		if req.Updates.Position != nil {
			params.Position = pgtype.Float8{Float64: *req.Updates.Position, Valid: true}
		}
		if _, ok := rawUpdates["assignee_type"]; ok {
			if req.Updates.AssigneeType != nil {
				params.AssigneeType = pgtype.Text{String: *req.Updates.AssigneeType, Valid: true}
			} else {
				params.AssigneeType = pgtype.Text{Valid: false}
			}
		}
		if _, ok := rawUpdates["assignee_id"]; ok {
			if req.Updates.AssigneeID != nil {
				assigneeUUID, err := util.ParseUUID(*req.Updates.AssigneeID)
				if err != nil {
					continue
				}
				params.AssigneeID = assigneeUUID
			} else {
				params.AssigneeID = pgtype.UUID{Valid: false}
			}
		}
		if _, ok := rawUpdates["start_date"]; ok {
			if req.Updates.StartDate != nil && *req.Updates.StartDate != "" {
				d, err := util.ParseCalendarDate(*req.Updates.StartDate)
				if err != nil {
					continue
				}
				params.StartDate = d
			} else {
				params.StartDate = pgtype.Date{Valid: false}
			}
		}
		if _, ok := rawUpdates["due_date"]; ok {
			if req.Updates.DueDate != nil && *req.Updates.DueDate != "" {
				d, err := util.ParseCalendarDate(*req.Updates.DueDate)
				if err != nil {
					continue
				}
				params.DueDate = d
			} else {
				params.DueDate = pgtype.Date{Valid: false}
			}
		}

		if _, ok := rawUpdates["parent_issue_id"]; ok {
			if req.Updates.ParentIssueID != nil {
				newParentID, err := util.ParseUUID(*req.Updates.ParentIssueID)
				if err != nil {
					continue
				}
				// Cannot set self as parent.
				if newParentID == prevIssue.ID {
					continue
				}
				// Validate parent exists in the same workspace.
				if _, err := h.Queries.GetIssueInWorkspace(r.Context(), db.GetIssueInWorkspaceParams{
					ID:          newParentID,
					WorkspaceID: prevIssue.WorkspaceID,
				}); err != nil {
					continue
				}
				// Cycle detection: walk up from the new parent to ensure we don't reach this issue.
				cycleDetected := false
				cursor := newParentID
				for depth := 0; depth < 10; depth++ {
					ancestor, err := h.Queries.GetIssue(r.Context(), cursor)
					if err != nil || !ancestor.ParentIssueID.Valid {
						break
					}
					if ancestor.ParentIssueID == prevIssue.ID {
						cycleDetected = true
						break
					}
					cursor = ancestor.ParentIssueID
				}
				if cycleDetected {
					continue
				}
				params.ParentIssueID = newParentID
			} else {
				params.ParentIssueID = pgtype.UUID{Valid: false}
			}
		}
		if _, ok := rawUpdates["project_id"]; ok {
			if req.Updates.ProjectID != nil {
				projectUUID, err := util.ParseUUID(*req.Updates.ProjectID)
				if err != nil {
					continue
				}
				params.ProjectID = projectUUID
			} else {
				params.ProjectID = pgtype.UUID{Valid: false}
			}
		}
		if _, ok := rawUpdates["stage"]; ok {
			if req.Updates.Stage != nil {
				if *req.Updates.Stage < 1 {
					continue
				}
				params.Stage = pgtype.Int4{Int32: *req.Updates.Stage, Valid: true}
			} else {
				params.Stage = pgtype.Int4{Valid: false} // explicit null = unstage
			}
		}

		// Validate the resulting assignee pair when this batch update touches
		// either assignee field. Skip the issue silently on failure.
		_, batchTouchedType := rawUpdates["assignee_type"]
		_, batchTouchedID := rawUpdates["assignee_id"]
		if batchTouchedType || batchTouchedID {
			if status, _ := h.validateAssigneePair(r.Context(), r, workspaceID, params.AssigneeType, params.AssigneeID); status != 0 {
				continue
			}
		}

		issue, err := h.Queries.UpdateIssue(r.Context(), params)
		if err != nil {
			slog.Warn("batch update issue failed", "issue_id", issueID, "error", err)
			continue
		}

		prefix := h.getIssuePrefix(r.Context(), issue.WorkspaceID)
		resp := issueToResponse(issue, prefix)
		actorType, actorID := h.resolveActor(r, userID, workspaceID)

		assigneeChanged := (req.Updates.AssigneeType != nil || req.Updates.AssigneeID != nil) &&
			(prevIssue.AssigneeType.String != issue.AssigneeType.String || uuidToString(prevIssue.AssigneeID) != uuidToString(issue.AssigneeID))
		statusChanged := req.Updates.Status != nil && prevIssue.Status != issue.Status
		priorityChanged := req.Updates.Priority != nil && prevIssue.Priority != issue.Priority
		projectChanged := req.Updates.ProjectID != nil && uuidToString(prevIssue.ProjectID) != uuidToString(issue.ProjectID)

		h.publish(protocol.EventIssueUpdated, workspaceID, actorType, actorID, map[string]any{
			"issue":            resp,
			"assignee_changed": assigneeChanged,
			"status_changed":   statusChanged,
			"priority_changed": priorityChanged,
			"project_changed":  projectChanged,
		})

		updated++
	}

	slog.Info("batch update issues", append(logger.RequestAttrs(r), "count", updated)...)
	writeJSON(w, http.StatusOK, map[string]any{"updated": updated})
}

type BatchDeleteIssuesRequest struct {
	IssueIDs []string `json:"issue_ids"`
}

func (h *Handler) BatchDeleteIssues(w http.ResponseWriter, r *http.Request) {
	var req BatchDeleteIssuesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.IssueIDs) == 0 {
		writeError(w, http.StatusBadRequest, "issue_ids is required")
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	workspaceID := h.resolveWorkspaceID(r)
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace_id")
	if !ok {
		return
	}
	deleted := 0
	for _, issueID := range req.IssueIDs {
		issueUUID, err := util.ParseUUID(issueID)
		if err != nil {
			continue
		}
		issue, err := h.Queries.GetIssueInWorkspace(r.Context(), db.GetIssueInWorkspaceParams{
			ID:          issueUUID,
			WorkspaceID: wsUUID,
		})
		if err != nil {
			continue
		}

		// Collect attachment URLs before CASCADE delete to clean up S3 objects.
		attachmentURLs, _ := h.Queries.ListAttachmentURLsByIssueOrComments(r.Context(), issue.ID)

		if err := h.Queries.DeleteIssue(r.Context(), db.DeleteIssueParams{
			ID:          issue.ID,
			WorkspaceID: issue.WorkspaceID,
		}); err != nil {
			slog.Warn("batch delete issue failed", "issue_id", issueID, "error", err)
			continue
		}

		h.deleteS3Objects(r.Context(), attachmentURLs)

		// Always emit the resolved UUID — frontend caches key by UUID.
		actorType, actorID := h.resolveActor(r, userID, workspaceID)
		h.publish(protocol.EventIssueDeleted, workspaceID, actorType, actorID, map[string]any{"issue_id": uuidToString(issue.ID)})
		deleted++
	}

	slog.Info("batch delete issues", append(logger.RequestAttrs(r), "count", deleted)...)
	writeJSON(w, http.StatusOK, map[string]any{"deleted": deleted})
}
