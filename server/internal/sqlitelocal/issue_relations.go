package sqlitelocal

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (s *Server) listIssueChildren(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	parentIDs := make([]string, 0)
	if parentID := strings.TrimSpace(chi.URLParam(r, "id")); parentID != "" {
		parentIDs = append(parentIDs, parentID)
	} else {
		for _, parentID := range strings.Split(r.URL.Query().Get("parent_ids"), ",") {
			if parentID = strings.TrimSpace(parentID); parentID != "" {
				parentIDs = append(parentIDs, parentID)
			}
		}
	}
	if len(parentIDs) == 0 {
		writeError(w, http.StatusBadRequest, "parent_ids is required")
		return
	}
	placeholders := make([]string, len(parentIDs))
	args := make([]any, 0, len(parentIDs)+1)
	args = append(args, workspaceValue.ID)
	for index, parentID := range parentIDs {
		placeholders[index] = "?"
		args = append(args, parentID)
	}
	rows, err := s.db.QueryContext(r.Context(), `SELECT `+issueColumns()+` FROM issues
		WHERE workspace_id = ? AND parent_issue_id IN (`+strings.Join(placeholders, ",")+`)
		ORDER BY position ASC, created_at DESC, id DESC`, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list child issues")
		return
	}
	defer rows.Close()
	children := make([]map[string]any, 0)
	for rows.Next() {
		value, err := scanIssue(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list child issues")
			return
		}
		children = append(children, value.response(workspaceValue.IssuePrefix))
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list child issues")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"issues": children})
}

func (s *Server) listIssueChildProgress(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	rows, err := s.db.QueryContext(r.Context(), `SELECT parent_issue_id, COUNT(*),
		SUM(CASE WHEN status = 'done' THEN 1 ELSE 0 END)
		FROM issues WHERE workspace_id = ? AND parent_issue_id IS NOT NULL
		GROUP BY parent_issue_id ORDER BY parent_issue_id`, workspaceValue.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list child issue progress")
		return
	}
	defer rows.Close()
	progress := make([]map[string]any, 0)
	for rows.Next() {
		var parentID string
		var total, done int
		if err := rows.Scan(&parentID, &total, &done); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list child issue progress")
			return
		}
		progress = append(progress, map[string]any{
			"parent_issue_id": parentID, "total": total, "done": done,
		})
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list child issue progress")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"progress": progress})
}
