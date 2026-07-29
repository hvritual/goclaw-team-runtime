package teamcontrol

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type Service struct {
	store *fileStore
	cfg   Config
}

// Open loads or creates a team control plane. If path does not end in .json,
// the state is stored as <path>/teamcontrol.json.
func Open(path string) (*Service, error) {
	return openService(path, true)
}

func openService(path string, enabled bool) (*Service, error) {
	root, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return nil, err
	}
	store, err := openFileStore(path)
	if err != nil {
		return nil, err
	}
	return &Service{
		store: store,
		cfg: Config{
			Enabled: enabled,
			Root:    root,
		},
	}, nil
}

// Config returns the effective service configuration. Root is always absolute,
// including when the service was opened with a relative path.
func (s *Service) Config() Config {
	return s.cfg
}

func (s *Service) CreateUser(input CreateUserInput) (User, error) {
	id, err := normalizeID(input.ID, "usr")
	if err != nil {
		return User{}, err
	}
	if isNonLoginPrincipal(id) {
		return User{}, fmt.Errorf(
			"user id %q is reserved for a non-login service principal",
			id,
		)
	}
	name, err := requireText(input.DisplayName, "display_name", 200)
	if err != nil {
		return User{}, err
	}
	email, err := validateEmail(input.Email)
	if err != nil {
		return User{}, err
	}
	var created User
	err = s.store.update(func(st *state) error {
		if _, exists := st.Users[id]; exists {
			return conflict("user %q already exists", id)
		}
		for _, user := range st.Users {
			if email != "" && strings.EqualFold(user.Email, email) {
				return conflict("email %q is already registered", email)
			}
		}
		now := time.Now().UTC()
		created = User{
			ID:          id,
			DisplayName: name,
			Email:       email,
			Status:      UserActive,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		st.Users[id] = created
		return nil
	})
	return created, err
}

func isNonLoginPrincipal(userID string) bool {
	return strings.EqualFold(
		strings.TrimSpace(userID),
		PlannerServicePrincipal,
	)
}

func (s *Service) SetUserStatus(userID string, status UserStatus) (User, error) {
	userID, err := requireID(userID, "user_id")
	if err != nil {
		return User{}, err
	}
	if !validUserStatus(status) {
		return User{}, fmt.Errorf("unsupported user status %q", status)
	}
	var updated User
	err = s.store.update(func(st *state) error {
		user, ok := st.Users[userID]
		if !ok {
			return entityNotFound("user", userID)
		}
		user.Status = status
		user.UpdatedAt = time.Now().UTC()
		st.Users[userID] = user
		updated = user
		return nil
	})
	return updated, err
}

func (s *Service) GetUser(userID string) (User, error) {
	userID, err := requireID(userID, "user_id")
	if err != nil {
		return User{}, err
	}
	var result User
	err = s.store.view(func(st state) error {
		user, ok := st.Users[userID]
		if !ok {
			return entityNotFound("user", userID)
		}
		result = user
		return nil
	})
	return result, err
}

// HasUsers is an unauthenticated bootstrap probe. It intentionally reveals
// only whether the control plane has been initialized, not any user details.
func (s *Service) HasUsers() (bool, error) {
	var result bool
	err := s.store.view(func(st state) error {
		result = len(st.Users) != 0
		return nil
	})
	return result, err
}

// ListUsers returns active users that share at least one active team with the
// actor. Before joining a team, an active actor can only see itself.
func (s *Service) ListUsers(actorID string) ([]User, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return nil, err
	}
	var result []User
	err = s.store.view(func(st state) error {
		if err := requireActiveUser(&st, actorID); err != nil {
			return err
		}
		visibleUserIDs := map[string]struct{}{actorID: {}}
		teamIDs := make(map[string]struct{})
		for _, membership := range st.TeamMemberships {
			if membership.UserID == actorID && membership.Status == MembershipActive {
				teamIDs[membership.TeamID] = struct{}{}
			}
		}
		for _, membership := range st.TeamMemberships {
			if membership.Status != MembershipActive {
				continue
			}
			if _, visible := teamIDs[membership.TeamID]; visible {
				visibleUserIDs[membership.UserID] = struct{}{}
			}
		}
		for userID := range visibleUserIDs {
			user, ok := st.Users[userID]
			if ok && user.Status == UserActive {
				result = append(result, user)
			}
		}
		slices.SortFunc(result, func(a, b User) int {
			return strings.Compare(a.ID, b.ID)
		})
		return nil
	})
	return result, err
}

func (s *Service) CreateTeam(actorID string, input CreateTeamInput) (Team, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return Team{}, err
	}
	id, err := normalizeID(input.ID, "team")
	if err != nil {
		return Team{}, err
	}
	name, err := requireText(input.Name, "name", 200)
	if err != nil {
		return Team{}, err
	}
	description, err := optionalText(input.Description, "description", 4000)
	if err != nil {
		return Team{}, err
	}
	var created Team
	err = s.store.update(func(st *state) error {
		if err := requireActiveUser(st, actorID); err != nil {
			return err
		}
		if _, exists := st.Teams[id]; exists {
			return conflict("team %q already exists", id)
		}
		now := time.Now().UTC()
		created = Team{
			ID:          id,
			Name:        name,
			Description: description,
			CreatedBy:   actorID,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		st.Teams[id] = created
		membership := TeamMembership{
			ID:        newID("tm"),
			TeamID:    id,
			UserID:    actorID,
			Role:      TeamOwner,
			Status:    MembershipActive,
			CreatedBy: actorID,
			CreatedAt: now,
			UpdatedAt: now,
		}
		st.TeamMemberships[membership.ID] = membership
		return nil
	})
	return created, err
}

func (s *Service) AddTeamMember(
	actorID, teamID string,
	input AddTeamMemberInput,
) (TeamMembership, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return TeamMembership{}, err
	}
	teamID, err = requireID(teamID, "team_id")
	if err != nil {
		return TeamMembership{}, err
	}
	userID, err := requireID(input.UserID, "user_id")
	if err != nil {
		return TeamMembership{}, err
	}
	if !validTeamRole(input.Role) {
		return TeamMembership{}, fmt.Errorf("unsupported team role %q", input.Role)
	}
	var created TeamMembership
	err = s.store.update(func(st *state) error {
		actorMembership, err := requireTeamAdmin(st, actorID, teamID)
		if err != nil {
			return err
		}
		if input.Role != TeamRegularMember && actorMembership.Role != TeamOwner {
			return fmt.Errorf("%w: only a team owner may grant %q", ErrForbidden, input.Role)
		}
		if err := requireActiveUser(st, userID); err != nil {
			return err
		}
		if existing := findTeamMembership(st, teamID, userID); existing != nil &&
			existing.Status != MembershipRemoved {
			return conflict("user %q already belongs to team %q", userID, teamID)
		}
		now := time.Now().UTC()
		created = TeamMembership{
			ID:        newID("tm"),
			TeamID:    teamID,
			UserID:    userID,
			Role:      input.Role,
			Status:    MembershipActive,
			CreatedBy: actorID,
			CreatedAt: now,
			UpdatedAt: now,
		}
		st.TeamMemberships[created.ID] = created
		return nil
	})
	return created, err
}

func (s *Service) CreateProject(actorID string, input CreateProjectInput) (Project, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return Project{}, err
	}
	teamID, err := requireID(input.TeamID, "team_id")
	if err != nil {
		return Project{}, err
	}
	id, err := normalizeID(input.ID, "prj")
	if err != nil {
		return Project{}, err
	}
	key, err := requireKey(input.Key, "key")
	if err != nil {
		return Project{}, err
	}
	key = strings.ToLower(key)
	name, err := requireText(input.Name, "name", 200)
	if err != nil {
		return Project{}, err
	}
	description, err := optionalText(input.Description, "description", 4000)
	if err != nil {
		return Project{}, err
	}
	var created Project
	err = s.store.update(func(st *state) error {
		if _, err := requireTeamAdmin(st, actorID, teamID); err != nil {
			return err
		}
		if _, exists := st.Projects[id]; exists {
			return conflict("project %q already exists", id)
		}
		for _, project := range st.Projects {
			if project.TeamID == teamID && strings.EqualFold(project.Key, key) {
				return conflict("project key %q already exists in team %q", key, teamID)
			}
		}
		now := time.Now().UTC()
		created = Project{
			ID:          id,
			TeamID:      teamID,
			Key:         key,
			Name:        name,
			Description: description,
			Status:      ProjectActive,
			CreatedBy:   actorID,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		st.Projects[id] = created
		membership := ProjectMembership{
			ID:        newID("pm"),
			ProjectID: id,
			UserID:    actorID,
			Role:      ProjectOwner,
			Status:    MembershipActive,
			CreatedBy: actorID,
			CreatedAt: now,
			UpdatedAt: now,
		}
		st.ProjectMemberships[membership.ID] = membership
		return nil
	})
	return created, err
}

func (s *Service) AddProjectMember(
	actorID, projectID string,
	input AddProjectMemberInput,
) (ProjectMembership, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return ProjectMembership{}, err
	}
	projectID, err = requireID(projectID, "project_id")
	if err != nil {
		return ProjectMembership{}, err
	}
	userID, err := requireID(input.UserID, "user_id")
	if err != nil {
		return ProjectMembership{}, err
	}
	if !validProjectRole(input.Role) {
		return ProjectMembership{}, fmt.Errorf("unsupported project role %q", input.Role)
	}
	businessDomains, err := cleanBusinessDomains(input.BusinessDomains)
	if err != nil {
		return ProjectMembership{}, err
	}
	if input.CapacityPoints < 0 || input.CapacityPoints > 10_000 {
		return ProjectMembership{}, fmt.Errorf(
			"capacity_points must be between 0 and 10000",
		)
	}
	var created ProjectMembership
	err = s.store.update(func(st *state) error {
		if err := authorizeProject(st, actorID, projectID, ActionProjectManage); err != nil {
			return err
		}
		actorMembership := findProjectMembership(st, projectID, actorID)
		if input.Role == ProjectOwner && actorMembership.Role != ProjectOwner {
			return fmt.Errorf("%w: only a project owner may grant owner", ErrForbidden)
		}
		project := st.Projects[projectID]
		teamMembership := findTeamMembership(st, project.TeamID, userID)
		if teamMembership == nil || teamMembership.Status != MembershipActive {
			return fmt.Errorf("%w: user %q is not an active team member", ErrForbidden, userID)
		}
		if err := requireActiveUser(st, userID); err != nil {
			return err
		}
		if existing := findProjectMembership(st, projectID, userID); existing != nil &&
			existing.Status != MembershipRemoved {
			return conflict("user %q already belongs to project %q", userID, projectID)
		}
		now := time.Now().UTC()
		created = ProjectMembership{
			ID:              newID("pm"),
			ProjectID:       projectID,
			UserID:          userID,
			Role:            input.Role,
			Status:          MembershipActive,
			BusinessDomains: businessDomains,
			CapacityPoints:  input.CapacityPoints,
			CreatedBy:       actorID,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		st.ProjectMemberships[created.ID] = created
		return nil
	})
	return created, err
}

func (s *Service) Authorize(userID, projectID string, action Action) error {
	userID, err := requireID(userID, "user_id")
	if err != nil {
		return err
	}
	projectID, err = requireID(projectID, "project_id")
	if err != nil {
		return err
	}
	return s.store.view(func(st state) error {
		return authorizeProject(&st, userID, projectID, action)
	})
}

func (s *Service) GetProject(userID, projectID string) (Project, error) {
	var result Project
	err := s.readProject(userID, projectID, ActionProjectRead, func(st state, project Project) error {
		result = project
		return nil
	})
	return result, err
}

func (s *Service) ListProjects(userID string) ([]Project, error) {
	userID, err := requireID(userID, "user_id")
	if err != nil {
		return nil, err
	}
	var result []Project
	err = s.store.view(func(st state) error {
		if err := requireActiveUser(&st, userID); err != nil {
			return err
		}
		for _, membership := range st.ProjectMemberships {
			if membership.UserID != userID || membership.Status != MembershipActive {
				continue
			}
			project, ok := st.Projects[membership.ProjectID]
			if ok {
				result = append(result, project)
			}
		}
		slices.SortFunc(result, func(a, b Project) int {
			return strings.Compare(a.ID, b.ID)
		})
		return nil
	})
	return result, err
}

func (s *Service) CreateRepository(
	actorID string,
	input CreateRepositoryInput,
) (Repository, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return Repository{}, err
	}
	projectID, err := requireID(input.ProjectID, "project_id")
	if err != nil {
		return Repository{}, err
	}
	id, err := normalizeID(input.ID, "repo")
	if err != nil {
		return Repository{}, err
	}
	name, err := requireText(input.Name, "name", 200)
	if err != nil {
		return Repository{}, err
	}
	remoteURL := strings.TrimSpace(input.RemoteURL)
	localPath := strings.TrimSpace(input.LocalPath)
	if remoteURL == "" && localPath == "" {
		return Repository{}, errors.New("remote_url or local_path is required")
	}
	if remoteURL != "" {
		if _, err := validateURI(remoteURL, "remote_url"); err != nil {
			return Repository{}, err
		}
	}
	defaultBranch := strings.TrimSpace(input.DefaultBranch)
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	defaultBranch, err = requireText(defaultBranch, "default_branch", 200)
	if err != nil {
		return Repository{}, err
	}
	var created Repository
	err = s.store.update(func(st *state) error {
		if err := authorizeProject(st, actorID, projectID, ActionRepositoryManage); err != nil {
			return err
		}
		if _, exists := st.Repositories[id]; exists {
			return conflict("repository %q already exists", id)
		}
		for _, repository := range st.Repositories {
			if repository.ProjectID == projectID && strings.EqualFold(repository.Name, name) {
				return conflict("repository name %q already exists in project", name)
			}
		}
		now := time.Now().UTC()
		created = Repository{
			ID:            id,
			ProjectID:     projectID,
			Name:          name,
			RemoteURL:     remoteURL,
			LocalPath:     localPath,
			DefaultBranch: defaultBranch,
			CreatedBy:     actorID,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		st.Repositories[id] = created
		return nil
	})
	return created, err
}

func (s *Service) GetRepository(
	userID, projectID, repositoryID string,
) (Repository, error) {
	repositoryID, err := requireID(repositoryID, "repository_id")
	if err != nil {
		return Repository{}, err
	}
	var result Repository
	err = s.readProject(userID, projectID, ActionRepositoryRead, func(st state, _ Project) error {
		repository, ok := st.Repositories[repositoryID]
		if !ok || repository.ProjectID != projectID {
			return entityNotFound("repository", repositoryID)
		}
		result = repository
		return nil
	})
	return result, err
}

func (s *Service) readProject(
	userID, projectID string,
	action Action,
	fn func(state, Project) error,
) error {
	userID, err := requireID(userID, "user_id")
	if err != nil {
		return err
	}
	projectID, err = requireID(projectID, "project_id")
	if err != nil {
		return err
	}
	return s.store.view(func(st state) error {
		if err := authorizeProject(&st, userID, projectID, action); err != nil {
			return err
		}
		return fn(st, st.Projects[projectID])
	})
}

func requireActiveUser(st *state, userID string) error {
	user, ok := st.Users[userID]
	if !ok {
		return entityNotFound("user", userID)
	}
	if user.Status != UserActive {
		return fmt.Errorf("%w: user %q is not active", ErrForbidden, userID)
	}
	return nil
}

func requireTeamAdmin(st *state, userID, teamID string) (*TeamMembership, error) {
	if err := requireActiveUser(st, userID); err != nil {
		return nil, err
	}
	if _, ok := st.Teams[teamID]; !ok {
		return nil, entityNotFound("team", teamID)
	}
	membership := findTeamMembership(st, teamID, userID)
	if membership == nil || membership.Status != MembershipActive ||
		(membership.Role != TeamOwner && membership.Role != TeamAdmin) {
		return nil, fmt.Errorf("%w: team administration is required", ErrForbidden)
	}
	return membership, nil
}

func authorizeProject(st *state, userID, projectID string, action Action) error {
	if err := requireActiveUser(st, userID); err != nil {
		return err
	}
	project, ok := st.Projects[projectID]
	if !ok {
		return entityNotFound("project", projectID)
	}
	if project.Status != ProjectActive && action != ActionProjectRead {
		return fmt.Errorf("%w: project %q is archived", ErrForbidden, projectID)
	}
	membership := findProjectMembership(st, projectID, userID)
	if membership == nil || membership.Status != MembershipActive {
		return fmt.Errorf("%w: user %q is not a project member", ErrForbidden, userID)
	}
	if !roleAllows(membership.Role, action) {
		return fmt.Errorf(
			"%w: project role %q does not allow %q",
			ErrForbidden,
			membership.Role,
			action,
		)
	}
	return nil
}

func roleAllows(role ProjectRole, action Action) bool {
	if role == ProjectOwner {
		return true
	}
	readAction := action == ActionProjectRead ||
		action == ActionRepositoryRead ||
		action == ActionIssueRead ||
		action == ActionWorkItemRead ||
		action == ActionArtifactRead ||
		action == ActionDocumentRead ||
		action == ActionComponentRead ||
		action == ActionPolicyRead ||
		action == ActionBudgetRead ||
		action == ActionKnowledgeRead ||
		action == ActionSkillRead ||
		action == ActionRunnerReleaseRead ||
		action == ActionContextRead
	if readAction {
		return true
	}
	switch role {
	case ProjectMaintainer:
		return true
	case ProjectDeveloper:
		switch action {
		case ActionIssueWrite, ActionIssueTransition,
			ActionWorkItemWrite, ActionArtifactWrite,
			ActionDocumentWrite, ActionComponentWrite:
			return true
		}
	case ProjectReviewer:
		switch action {
		case ActionIssueTransition, ActionArtifactWrite, ActionDocumentWrite:
			return true
		}
	}
	return false
}

func findTeamMembership(st *state, teamID, userID string) *TeamMembership {
	for id, membership := range st.TeamMemberships {
		if membership.TeamID == teamID && membership.UserID == userID {
			copy := membership
			copy.ID = id
			return &copy
		}
	}
	return nil
}

func findProjectMembership(st *state, projectID, userID string) *ProjectMembership {
	for id, membership := range st.ProjectMemberships {
		if membership.ProjectID == projectID && membership.UserID == userID {
			copy := membership
			copy.ID = id
			return &copy
		}
	}
	return nil
}
