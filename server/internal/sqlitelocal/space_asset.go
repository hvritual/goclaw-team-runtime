package sqlitelocal

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	authcontract "github.com/multica-ai/multica/server/internal/modules/auth/contract"
	spacecontract "github.com/multica-ai/multica/server/internal/modules/space/contract"
	spacehttp "github.com/multica-ai/multica/server/internal/modules/space/interfaces/http"
	"github.com/multica-ai/multica/server/internal/storage"
)

type sqliteSpaceObjectStore struct {
	storage storage.Storage
}

func (s *sqliteSpaceObjectStore) Available() bool { return s != nil && s.storage != nil }
func (s *sqliteSpaceObjectStore) Upload(
	ctx context.Context,
	key string,
	data []byte,
	mediaType string,
	filename string,
) (string, error) {
	return s.storage.Upload(ctx, key, data, mediaType, filename)
}
func (s *sqliteSpaceObjectStore) DeleteObject(ctx context.Context, key string) error {
	return s.storage.DeleteObject(ctx, key)
}
func (s *sqliteSpaceObjectStore) KeyFromURL(rawURL string) string {
	return s.storage.KeyFromURL(rawURL)
}
func (s *sqliteSpaceObjectStore) GetReader(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.storage.GetReader(ctx, key)
}

type sqliteSpaceUploader struct {
	db      *sql.DB
	assets  spacecontract.AssetUploadService
	members authcontract.MemberService
}

func (u *sqliteSpaceUploader) Available() bool {
	return u != nil && u.assets != nil && u.assets.Available()
}

func (u *sqliteSpaceUploader) Upload(
	ctx context.Context,
	request spacehttp.UploadRequest,
) (spacehttp.UploadResult, error) {
	if request.IssueID != nil && request.WorkspaceID != "" {
		_, memberErr := u.members.ListMembers(
			authcontract.WithMemberActor(ctx, request.UserID),
			authcontract.Member_ListMembersRequest{WorkspaceId: request.WorkspaceID},
		)
		if errors.Is(memberErr, authcontract.ErrWorkspaceMembershipHidden) ||
			errors.Is(memberErr, authcontract.ErrAuthUserNotFound) {
			return spacehttp.UploadResult{}, spacecontract.ErrAssetWorkspaceForbidden
		}
		if memberErr != nil {
			return spacehttp.UploadResult{}, memberErr
		}
		if _, err := uuid.Parse(*request.IssueID); err != nil {
			return spacehttp.UploadResult{}, spacehttp.ErrInvalidIssueID
		}
		var exists int
		err := u.db.QueryRowContext(ctx, `SELECT 1 FROM issues WHERE id = ? AND workspace_id = ?`,
			*request.IssueID, request.WorkspaceID,
		).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return spacehttp.UploadResult{}, spacehttp.ErrIssueNotAccessible
		}
		if err != nil {
			return spacehttp.UploadResult{}, err
		}
	}
	result, err := u.assets.UploadAsset(
		spacecontract.WithAssetActor(ctx, request.UserID),
		spacecontract.Asset_UploadAssetRequest{
			WorkspaceId: request.WorkspaceID, Filename: request.Filename,
			MediaType: request.ContentType, Content: request.Content,
		},
	)
	if err != nil {
		return spacehttp.UploadResult{}, err
	}
	if result.Asset == nil {
		return spacehttp.UploadResult{
			ID: result.DirectObjectId, URL: result.DirectUrl, Filename: result.Filename,
		}, nil
	}
	if request.IssueID != nil && request.WorkspaceID != "" {
		if _, err := u.db.ExecContext(ctx, `INSERT INTO issue_asset_refs(
			asset_id, issue_id, workspace_id, created_by, created_at
		) VALUES (?, ?, ?, ?, ?)`,
			result.Asset.Id, *request.IssueID, request.WorkspaceID, request.UserID, now(),
		); err != nil {
			return spacehttp.UploadResult{}, errors.Join(spacehttp.ErrAssetRelation, err)
		}
	}
	return spacehttp.UploadResult{Asset: result.Asset, IssueID: request.IssueID}, nil
}

func (s *Server) resolveUploadWorkspace(request *http.Request) (string, error) {
	workspaceID := strings.TrimSpace(request.Header.Get("X-Workspace-ID"))
	if workspaceID == "" {
		workspaceID = strings.TrimSpace(request.URL.Query().Get("workspace_id"))
	}
	if workspaceID != "" {
		var resolved string
		if err := s.db.QueryRowContext(request.Context(), `SELECT id FROM workspaces WHERE id = ?`, workspaceID).Scan(&resolved); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", spacehttp.ErrWorkspaceNotFound
			}
			return "", err
		}
		return resolved, nil
	}
	slug := strings.TrimSpace(request.Header.Get("X-Workspace-Slug"))
	if slug == "" {
		return "", nil
	}
	if err := s.db.QueryRowContext(request.Context(), `SELECT id FROM workspaces WHERE slug = ?`, slug).Scan(&workspaceID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", spacehttp.ErrWorkspaceNotFound
		}
		return "", err
	}
	return workspaceID, nil
}

func (s *Server) listIssueAssets(writer http.ResponseWriter, request *http.Request) {
	value, _, ok := s.loadIssue(writer, request, chi.URLParam(request, "id"))
	if !ok {
		return
	}
	rows, err := s.db.QueryContext(request.Context(), `SELECT asset_id
		FROM issue_asset_refs WHERE issue_id = ? AND workspace_id = ? ORDER BY created_at`,
		value.ID, value.WorkspaceID,
	)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "failed to list attachments")
		return
	}
	assetIDs := make([]string, 0)
	for rows.Next() {
		var assetID string
		if err := rows.Scan(&assetID); err != nil {
			_ = rows.Close()
			writeError(writer, http.StatusInternalServerError, "failed to list attachments")
			return
		}
		assetIDs = append(assetIDs, assetID)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		writeError(writer, http.StatusInternalServerError, "failed to list attachments")
		return
	}
	_ = rows.Close()
	responses := make([]spacehttp.AttachmentResponse, 0, len(assetIDs))
	actorContext := spacecontract.WithAssetActor(request.Context(), currentUserID(request))
	for _, assetID := range assetIDs {
		result, err := s.spaceAssets.GetAsset(
			actorContext,
			spacecontract.Asset_GetAssetRequest{AssetId: assetID},
		)
		if err != nil || result.Asset == nil {
			continue
		}
		issueID := value.ID
		responses = append(responses, s.spaceURLPolicy.ResponseFor(*result.Asset, &issueID))
	}
	writeJSON(writer, http.StatusOK, responses)
}
