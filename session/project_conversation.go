package session

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	DefaultProjectTopic          = "inbox"
	projectConversationKeyPrefix = "project-v2."
)

var (
	ErrAmbiguousLegacyProjectConversation = errors.New(
		"legacy project conversation ownership is ambiguous",
	)
	projectConversationSegment = regexp.MustCompile(
		`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`,
	)
)

// ProjectConversationKey returns the versioned, canonical session boundary
// shared by Agent and Gateway. Each segment is independently base64url encoded;
// "." is not in that alphabet, so the boundary remains unambiguous on disk.
func ProjectConversationKey(projectID, topicID string) (string, string, error) {
	projectID = strings.TrimSpace(projectID)
	topicID = strings.TrimSpace(topicID)
	if topicID == "" {
		topicID = DefaultProjectTopic
	}
	if !projectConversationSegment.MatchString(projectID) {
		return "", "", fmt.Errorf("project_id must be a valid conversation segment")
	}
	if !projectConversationSegment.MatchString(topicID) {
		return "", "", fmt.Errorf("topic_id must be a valid conversation segment")
	}
	encode := base64.RawURLEncoding.EncodeToString
	return projectConversationKeyPrefix + encode([]byte(projectID)) + "." +
		encode([]byte(topicID)), topicID, nil
}

// MergeMetadata updates session metadata under the same lock used by history
// readers and persistence.
func (s *Session) MergeMetadata(values map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Metadata == nil {
		s.Metadata = make(map[string]interface{})
	}
	for key, value := range values {
		s.Metadata[key] = value
	}
	s.UpdatedAt = time.Now()
}

// ProjectScope returns a consistent snapshot of the stored project boundary.
func (s *Session) ProjectScope() (projectID, topicID string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	projectID, _ = s.Metadata["project_id"].(string)
	topicID, _ = s.Metadata["topic_id"].(string)
	return strings.TrimSpace(projectID), strings.TrimSpace(topicID)
}

// GetExistingProjectConversation reads or safely migrates a project
// conversation without creating a missing conversation.
func (m *Manager) GetExistingProjectConversation(
	projectID, topicID string,
) (*Session, string, bool, error) {
	return m.projectConversation(projectID, topicID, false)
}

// GetOrCreateProjectConversation is the only supported write entry point for
// project-scoped conversations.
func (m *Manager) GetOrCreateProjectConversation(
	projectID, topicID string,
) (*Session, string, error) {
	conversation, normalizedTopic, _, err := m.projectConversation(
		projectID,
		topicID,
		true,
	)
	return conversation, normalizedTopic, err
}

func (m *Manager) projectConversation(
	projectID, topicID string,
	create bool,
) (*Session, string, bool, error) {
	key, topicID, err := ProjectConversationKey(projectID, topicID)
	if err != nil {
		return nil, "", false, err
	}
	projectID = strings.TrimSpace(projectID)

	m.mu.Lock()
	defer m.mu.Unlock()
	if cached, ok := m.sessions[key]; ok {
		return cached, topicID, true, nil
	}
	loaded, err := m.load(key)
	if err == nil {
		m.sessions[key] = loaded
		return loaded, topicID, true, nil
	}
	if !os.IsNotExist(err) {
		return nil, "", false, fmt.Errorf("load project conversation: %w", err)
	}

	legacyKey := legacyProjectConversationKey(projectID, topicID)
	legacyPath := m.sessionPath(legacyKey)
	if _, statErr := os.Stat(legacyPath); statErr == nil {
		migrated, migrationErr := m.migrateLegacyProjectConversationLocked(
			key,
			legacyKey,
			projectID,
			topicID,
		)
		if migrationErr != nil {
			return nil, "", false, migrationErr
		}
		m.sessions[key] = migrated
		return migrated, topicID, true, nil
	} else if !os.IsNotExist(statErr) {
		return nil, "", false, fmt.Errorf(
			"inspect legacy project conversation: %w",
			statErr,
		)
	}

	if !create {
		return nil, topicID, false, nil
	}
	now := time.Now()
	created := &Session{
		Key:       key,
		Messages:  []Message{},
		CreatedAt: now,
		UpdatedAt: now,
		Metadata: map[string]interface{}{
			"project_id": projectID,
			"topic_id":   topicID,
		},
	}
	m.sessions[key] = created
	return created, topicID, false, nil
}

func (m *Manager) migrateLegacyProjectConversationLocked(
	newKey, legacyKey, projectID, topicID string,
) (*Session, error) {
	if _, active := m.sessions[legacyKey]; active {
		return nil, fmt.Errorf(
			"%w: legacy conversation is active",
			ErrAmbiguousLegacyProjectConversation,
		)
	}
	// The legacy filename joined segments with "_". Any "_" in either source
	// segment therefore permits at least two valid interpretations.
	if strings.ContainsRune(projectID, '_') || strings.ContainsRune(topicID, '_') {
		return nil, fmt.Errorf(
			"%w: legacy key contains an ambiguous separator",
			ErrAmbiguousLegacyProjectConversation,
		)
	}
	legacy, err := m.load(legacyKey)
	if err != nil {
		return nil, fmt.Errorf("load legacy project conversation: %w", err)
	}
	storedProject, storedTopic := legacy.ProjectScope()
	if storedProject != projectID || storedTopic != topicID {
		return nil, fmt.Errorf(
			"%w: stored scope does not match requested scope",
			ErrAmbiguousLegacyProjectConversation,
		)
	}
	oldPath := m.sessionPath(legacyKey)
	newPath := m.sessionPath(newKey)
	if _, err := os.Stat(newPath); err == nil {
		return nil, fmt.Errorf("new project conversation already exists")
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect migration target: %w", err)
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return nil, fmt.Errorf("atomically migrate project conversation: %w", err)
	}
	legacy.Key = newKey
	return legacy, nil
}

func legacyProjectConversationKey(projectID, topicID string) string {
	return fmt.Sprintf("project:%s:%s", projectID, topicID)
}
