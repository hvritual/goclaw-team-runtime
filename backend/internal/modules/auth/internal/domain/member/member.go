package member

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrIDRequired                = errors.New("member id is required")
	ErrWorkspaceRequired         = errors.New("workspace id is required")
	ErrUserRequired              = errors.New("user id is required")
	ErrInvalidRole               = errors.New("invalid member role")
	ErrInsufficientWorkspaceRole = errors.New("insufficient workspace role")
	ErrOwnerRoleRequiresOwner    = errors.New("owner role requires owner permission")
	ErrLastOwner                 = errors.New("workspace must retain an owner")
	ErrRootMemberRequired        = errors.New("membership root member id is required")
)

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

func ParseRole(value string) (Role, error) {
	role := Role(strings.TrimSpace(value))
	switch role {
	case RoleOwner, RoleAdmin, RoleMember:
		return role, nil
	default:
		return "", ErrInvalidRole
	}
}

type User struct {
	id        string
	name      string
	email     string
	avatarURL *string
}

func RehydrateUser(id, name, email string, avatarURL *string) (User, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return User{}, ErrUserRequired
	}
	return User{id: id, name: name, email: email, avatarURL: copyString(avatarURL)}, nil
}

func (u User) ID() string         { return u.id }
func (u User) Name() string       { return u.name }
func (u User) Email() string      { return u.email }
func (u User) AvatarURL() *string { return copyString(u.avatarURL) }

type Member struct {
	id          string
	workspaceID string
	userID      string
	role        Role
	createdAt   time.Time
	name        string
	email       string
	avatarURL   *string
}

func NewInitialOwner(id, workspaceID string, user User, now time.Time) (Member, error) {
	return newMember(id, workspaceID, user.ID(), RoleOwner, now, user.Name(), user.Email(), user.AvatarURL())
}

func Rehydrate(id, workspaceID, userID, role string, createdAt time.Time, name, email string, avatarURL *string) (Member, error) {
	parsedRole, err := ParseRole(role)
	if err != nil {
		return Member{}, err
	}
	return newMember(id, workspaceID, userID, parsedRole, createdAt, name, email, avatarURL)
}

func newMember(id, workspaceID, userID string, role Role, createdAt time.Time, name, email string, avatarURL *string) (Member, error) {
	id = strings.TrimSpace(id)
	workspaceID = strings.TrimSpace(workspaceID)
	userID = strings.TrimSpace(userID)
	if id == "" {
		return Member{}, ErrIDRequired
	}
	if workspaceID == "" {
		return Member{}, ErrWorkspaceRequired
	}
	if userID == "" {
		return Member{}, ErrUserRequired
	}
	if _, err := ParseRole(string(role)); err != nil {
		return Member{}, err
	}
	return Member{
		id: id, workspaceID: workspaceID, userID: userID, role: role,
		createdAt: createdAt.UTC(), name: name, email: email, avatarURL: copyString(avatarURL),
	}, nil
}

func (m Member) ChangeRole(requester, next Role, ownerCount int) (Member, error) {
	if err := ValidateRoleChange(requester, m.role, next, ownerCount); err != nil {
		return Member{}, err
	}
	updated := m
	updated.role = next
	return updated, nil
}

func ValidateManager(role Role) error {
	if role != RoleOwner && role != RoleAdmin {
		return ErrInsufficientWorkspaceRole
	}
	return nil
}

func ValidateRoleChange(requester, target, next Role, ownerCount int) error {
	if err := ValidateManager(requester); err != nil {
		return err
	}
	if (target == RoleOwner || next == RoleOwner) && requester != RoleOwner {
		return ErrOwnerRoleRequiresOwner
	}
	if target == RoleOwner && next != RoleOwner && ownerCount <= 1 {
		return ErrLastOwner
	}
	return nil
}

func (m Member) ID() string           { return m.id }
func (m Member) WorkspaceID() string  { return m.workspaceID }
func (m Member) UserID() string       { return m.userID }
func (m Member) Role() Role           { return m.role }
func (m Member) CreatedAt() time.Time { return m.createdAt }
func (m Member) Name() string         { return m.name }
func (m Member) Email() string        { return m.email }
func (m Member) AvatarURL() *string   { return copyString(m.avatarURL) }

type WorkspaceRoot struct {
	workspaceID string
	userID      string
	memberID    string
	createdAt   time.Time
}

func NewWorkspaceRoot(workspaceID, userID, memberID string, createdAt time.Time) (WorkspaceRoot, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	userID = strings.TrimSpace(userID)
	memberID = strings.TrimSpace(memberID)
	if workspaceID == "" {
		return WorkspaceRoot{}, ErrWorkspaceRequired
	}
	if userID == "" {
		return WorkspaceRoot{}, ErrUserRequired
	}
	if memberID == "" {
		return WorkspaceRoot{}, ErrRootMemberRequired
	}
	return WorkspaceRoot{workspaceID: workspaceID, userID: userID, memberID: memberID, createdAt: createdAt.UTC()}, nil
}

func (r WorkspaceRoot) WorkspaceID() string  { return r.workspaceID }
func (r WorkspaceRoot) UserID() string       { return r.userID }
func (r WorkspaceRoot) MemberID() string     { return r.memberID }
func (r WorkspaceRoot) CreatedAt() time.Time { return r.createdAt }

func copyString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
