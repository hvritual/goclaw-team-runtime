package catalog

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/smallnest/goclaw/governance"
)

func (s *Service) UpsertAuthority(input AuthorityInput, review governance.Review) (Authority, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	input.ProjectID = s.projectID(input.ProjectID)
	input.ID = cleanID(input.ID)
	input.PreferredLabel = strings.TrimSpace(input.PreferredLabel)
	input.Aliases = cleanStrings(input.Aliases)
	input.Description = strings.TrimSpace(input.Description)
	input.ExternalIDs = cleanStringMap(input.ExternalIDs)
	input.CreatedBy = strings.TrimSpace(input.CreatedBy)
	if input.CreatedBy == "" {
		input.CreatedBy = "catalog-importer"
	}
	if input.PreferredLabel == "" {
		return Authority{}, errors.New("authority preferred label is required")
	}
	if !validAuthorityType(input.Type) {
		return Authority{}, fmt.Errorf("unsupported authority type %q", input.Type)
	}
	if err := governance.ValidateRole(review, governance.RoleAuthorityManage); err != nil {
		return Authority{}, err
	}
	if err := governance.ValidateApproval(s.governance, review, input.CreatedBy); err != nil {
		return Authority{}, err
	}
	decision := governance.ToDecision(review, "approved")
	now := decision.CreatedAt.UTC()
	if input.ID == "" {
		input.ID = "auth-" + uuid.NewString()
	}
	existing, err := s.getAuthority(input.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Authority{}, err
	}
	if err == nil && existing.ProjectID != input.ProjectID {
		return Authority{}, fmt.Errorf(
			"authority %s belongs to project %s, not %s",
			existing.ID,
			existing.ProjectID,
			input.ProjectID,
		)
	}
	createdAt := now
	if err == nil {
		createdAt = existing.CreatedAt
	}
	authority := Authority{
		SchemaVersion:  SchemaVersion,
		ID:             input.ID,
		ProjectID:      input.ProjectID,
		Type:           input.Type,
		PreferredLabel: input.PreferredLabel,
		Aliases:        input.Aliases,
		Description:    input.Description,
		ExternalIDs:    input.ExternalIDs,
		Status:         AuthorityActive,
		CreatedBy:      input.CreatedBy,
		CreatedAt:      createdAt,
		UpdatedAt:      now,
		Decision:       &decision,
	}
	aliasesJSON, _ := json.Marshal(nonNilStrings(authority.Aliases))
	externalJSON, _ := json.Marshal(nonNilStringMap(authority.ExternalIDs))
	decisionJSON, _ := json.Marshal(decision)
	_, err = s.db.Exec(
		`INSERT INTO catalog_authorities(
			id, schema_version, project_id, type, preferred_label, aliases_json,
			description, external_ids_json, status, redirect_to, created_by,
			created_at, updated_at, decision_json
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			project_id=excluded.project_id,
			type=excluded.type,
			preferred_label=excluded.preferred_label,
			aliases_json=excluded.aliases_json,
			description=excluded.description,
			external_ids_json=excluded.external_ids_json,
			status=excluded.status,
			redirect_to=excluded.redirect_to,
			updated_at=excluded.updated_at,
			decision_json=excluded.decision_json`,
		authority.ID,
		authority.SchemaVersion,
		authority.ProjectID,
		authority.Type,
		authority.PreferredLabel,
		string(aliasesJSON),
		authority.Description,
		string(externalJSON),
		authority.Status,
		"",
		authority.CreatedBy,
		formatTime(authority.CreatedAt),
		formatTime(authority.UpdatedAt),
		string(decisionJSON),
	)
	if err != nil {
		return Authority{}, err
	}
	return s.getAuthority(authority.ID)
}

func (s *Service) GetAuthority(id string) (Authority, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	authority, err := s.getAuthority(strings.TrimSpace(id))
	if errors.Is(err, sql.ErrNoRows) {
		return Authority{}, fmt.Errorf("authority %s not found", id)
	}
	return authority, err
}

func (s *Service) getAuthority(id string) (Authority, error) {
	row := s.db.QueryRow(
		`SELECT id, schema_version, project_id, type, preferred_label,
		        aliases_json, description, external_ids_json, status, redirect_to,
		        created_by, created_at, updated_at, decision_json
		 FROM catalog_authorities WHERE id = ?`,
		id,
	)
	return scanAuthority(row)
}

func (s *Service) ListAuthorities(projectID string, includeRedirected bool) ([]Authority, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	projectID = s.projectID(projectID)
	query := `SELECT id, schema_version, project_id, type, preferred_label,
	                 aliases_json, description, external_ids_json, status, redirect_to,
	                 created_by, created_at, updated_at, decision_json
	          FROM catalog_authorities WHERE project_id IN (?, '*')`
	args := []any{projectID}
	if !includeRedirected {
		query += ` AND status = ?`
		args = append(args, AuthorityActive)
	}
	query += ` ORDER BY preferred_label`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Authority, 0)
	for rows.Next() {
		authority, err := scanAuthority(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, authority)
	}
	return result, rows.Err()
}

func (s *Service) ResolveAuthority(projectID, label string) (Authority, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	projectID = s.projectID(projectID)
	needle := strings.ToLower(strings.TrimSpace(label))
	if needle == "" {
		return Authority{}, errors.New("authority label is required")
	}
	authorities, err := s.listAuthoritiesUnlocked(projectID, true)
	if err != nil {
		return Authority{}, err
	}
	for _, authority := range authorities {
		if strings.ToLower(authority.PreferredLabel) != needle &&
			!containsFold(authority.Aliases, needle) {
			continue
		}
		if authority.Status == AuthorityRedirected && authority.RedirectTo != "" {
			return s.getAuthority(authority.RedirectTo)
		}
		return authority, nil
	}
	return Authority{}, fmt.Errorf("authority %q not found", label)
}

func (s *Service) MergeAuthority(sourceID, targetID string, review governance.Review) (Authority, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sourceID = strings.TrimSpace(sourceID)
	targetID = strings.TrimSpace(targetID)
	if sourceID == "" || targetID == "" || sourceID == targetID {
		return Authority{}, errors.New("distinct source and target authority ids are required")
	}
	if err := governance.ValidateRole(review, governance.RoleAuthorityManage); err != nil {
		return Authority{}, err
	}
	source, err := s.getAuthority(sourceID)
	if err != nil {
		return Authority{}, err
	}
	target, err := s.getAuthority(targetID)
	if err != nil {
		return Authority{}, err
	}
	if source.ProjectID != target.ProjectID &&
		source.ProjectID != "*" &&
		target.ProjectID != "*" {
		return Authority{}, errors.New("cannot merge authorities from unrelated projects")
	}
	if err := governance.ValidateApproval(s.governance, review, source.CreatedBy); err != nil {
		return Authority{}, err
	}
	decision := governance.ToDecision(review, "merged")
	now := decision.CreatedAt.UTC()
	tx, err := s.db.Begin()
	if err != nil {
		return Authority{}, err
	}
	defer func() { _ = tx.Rollback() }()
	decisionJSON, _ := json.Marshal(decision)
	if _, err := tx.Exec(
		`UPDATE catalog_authorities
		 SET status = ?, redirect_to = ?, updated_at = ?, decision_json = ?
		 WHERE id = ?`,
		AuthorityRedirected,
		target.ID,
		formatTime(now),
		string(decisionJSON),
		source.ID,
	); err != nil {
		return Authority{}, err
	}
	rows, err := tx.Query(
		`SELECT id, authority_ids_json FROM catalog_records
		 WHERE project_id IN (?, '*')`,
		source.ProjectID,
	)
	if err != nil {
		return Authority{}, err
	}
	type replacement struct {
		id   string
		json string
	}
	replacements := make([]replacement, 0)
	for rows.Next() {
		var id, raw string
		if err := rows.Scan(&id, &raw); err != nil {
			_ = rows.Close()
			return Authority{}, err
		}
		var ids []string
		if err := json.Unmarshal([]byte(raw), &ids); err != nil {
			_ = rows.Close()
			return Authority{}, err
		}
		changed := false
		for index := range ids {
			if ids[index] == source.ID {
				ids[index] = target.ID
				changed = true
			}
		}
		if changed {
			data, _ := json.Marshal(cleanStrings(ids))
			replacements = append(replacements, replacement{id: id, json: string(data)})
		}
	}
	if err := rows.Close(); err != nil {
		return Authority{}, err
	}
	for _, item := range replacements {
		if _, err := tx.Exec(
			`UPDATE catalog_records SET authority_ids_json = ?, updated_at = ? WHERE id = ?`,
			item.json,
			formatTime(now),
			item.id,
		); err != nil {
			return Authority{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Authority{}, err
	}
	return s.getAuthority(target.ID)
}

func (s *Service) authorityLabels(projectID string) (map[string][]string, error) {
	authorities, err := s.listAuthoritiesUnlocked(projectID, true)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]string)
	for _, authority := range authorities {
		labels := append([]string{authority.PreferredLabel}, authority.Aliases...)
		if authority.Status == AuthorityRedirected && authority.RedirectTo != "" {
			target, targetErr := s.getAuthority(authority.RedirectTo)
			if targetErr == nil {
				labels = append(labels, target.PreferredLabel)
				labels = append(labels, target.Aliases...)
			}
		}
		result[authority.ID] = cleanStrings(labels)
	}
	return result, nil
}

func (s *Service) listAuthoritiesUnlocked(projectID string, includeRedirected bool) ([]Authority, error) {
	query := `SELECT id, schema_version, project_id, type, preferred_label,
	                 aliases_json, description, external_ids_json, status, redirect_to,
	                 created_by, created_at, updated_at, decision_json
	          FROM catalog_authorities WHERE project_id IN (?, '*')`
	args := []any{projectID}
	if !includeRedirected {
		query += ` AND status = ?`
		args = append(args, AuthorityActive)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Authority, 0)
	for rows.Next() {
		authority, err := scanAuthority(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, authority)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].PreferredLabel < result[j].PreferredLabel
	})
	return result, rows.Err()
}

func scanAuthority(row scanner) (Authority, error) {
	var authority Authority
	var authorityType, status, aliasesJSON, externalJSON, createdAt, updatedAt string
	var decisionJSON string
	err := row.Scan(
		&authority.ID,
		&authority.SchemaVersion,
		&authority.ProjectID,
		&authorityType,
		&authority.PreferredLabel,
		&aliasesJSON,
		&authority.Description,
		&externalJSON,
		&status,
		&authority.RedirectTo,
		&authority.CreatedBy,
		&createdAt,
		&updatedAt,
		&decisionJSON,
	)
	if err != nil {
		return Authority{}, err
	}
	authority.Type = AuthorityType(authorityType)
	authority.Status = AuthorityStatus(status)
	if err := json.Unmarshal([]byte(aliasesJSON), &authority.Aliases); err != nil {
		return Authority{}, err
	}
	if err := json.Unmarshal([]byte(externalJSON), &authority.ExternalIDs); err != nil {
		return Authority{}, err
	}
	if strings.TrimSpace(decisionJSON) != "" && decisionJSON != "{}" {
		var decision governance.DecisionRecord
		if err := json.Unmarshal([]byte(decisionJSON), &decision); err != nil {
			return Authority{}, err
		}
		authority.Decision = &decision
	}
	authority.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return Authority{}, err
	}
	authority.UpdatedAt, err = parseTime(updatedAt)
	return authority, err
}

func validAuthorityType(value AuthorityType) bool {
	switch value {
	case AuthorityPerson, AuthorityOrganization, AuthorityProject, AuthoritySystem,
		AuthorityTopic, AuthorityPlace, AuthorityDevice:
		return true
	default:
		return false
	}
}

func containsFold(values []string, needle string) bool {
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == needle {
			return true
		}
	}
	return false
}
