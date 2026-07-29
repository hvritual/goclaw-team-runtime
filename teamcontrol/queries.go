package teamcontrol

import (
	"slices"
	"strings"
)

func (s *Service) GetTeam(userID, teamID string) (Team, error) {
	userID, err := requireID(userID, "user_id")
	if err != nil {
		return Team{}, err
	}
	teamID, err = requireID(teamID, "team_id")
	if err != nil {
		return Team{}, err
	}
	var result Team
	err = s.store.view(func(st state) error {
		if _, err := requireTeamMember(&st, userID, teamID); err != nil {
			return err
		}
		result = st.Teams[teamID]
		return nil
	})
	return result, err
}

func (s *Service) ListTeamMembers(userID, teamID string) ([]TeamMember, error) {
	userID, err := requireID(userID, "user_id")
	if err != nil {
		return nil, err
	}
	teamID, err = requireID(teamID, "team_id")
	if err != nil {
		return nil, err
	}
	var result []TeamMember
	err = s.store.view(func(st state) error {
		if _, err := requireTeamMember(&st, userID, teamID); err != nil {
			return err
		}
		for _, membership := range st.TeamMemberships {
			if membership.TeamID == teamID && membership.Status != MembershipRemoved {
				result = append(result, membership)
			}
		}
		slices.SortFunc(result, func(a, b TeamMember) int {
			return strings.Compare(a.UserID, b.UserID)
		})
		return nil
	})
	return result, err
}

func (s *Service) ListProjectMembers(userID, projectID string) ([]ProjectMember, error) {
	var result []ProjectMember
	err := s.readProject(userID, projectID, ActionProjectRead, func(st state, _ Project) error {
		for _, membership := range st.ProjectMemberships {
			if membership.ProjectID == projectID && membership.Status != MembershipRemoved {
				result = append(result, membership)
			}
		}
		slices.SortFunc(result, func(a, b ProjectMember) int {
			return strings.Compare(a.UserID, b.UserID)
		})
		return nil
	})
	return result, err
}

func (s *Service) ListRepositories(userID, projectID string) ([]Repository, error) {
	var result []Repository
	err := s.readProject(userID, projectID, ActionRepositoryRead, func(st state, _ Project) error {
		for _, repository := range st.Repositories {
			if repository.ProjectID == projectID {
				result = append(result, repository)
			}
		}
		slices.SortFunc(result, func(a, b Repository) int {
			return strings.Compare(a.ID, b.ID)
		})
		return nil
	})
	return result, err
}

func (s *Service) ListWorkItems(userID, projectID string) ([]WorkItem, error) {
	var result []WorkItem
	err := s.readProject(userID, projectID, ActionWorkItemRead, func(st state, _ Project) error {
		for _, item := range st.WorkItems {
			if item.ProjectID == projectID {
				result = append(result, item)
			}
		}
		slices.SortFunc(result, func(a, b WorkItem) int {
			return strings.Compare(a.ID, b.ID)
		})
		return nil
	})
	return result, err
}

func (s *Service) ListAssignments(
	userID, projectID string,
	targetType AssignmentTarget,
	targetID string,
) ([]Assignment, error) {
	targetID = strings.TrimSpace(targetID)
	if targetID != "" {
		var err error
		if targetID, err = requireID(targetID, "target_id"); err != nil {
			return nil, err
		}
	}
	var result []Assignment
	err := s.readProject(userID, projectID, ActionWorkItemRead, func(st state, _ Project) error {
		for _, assignment := range st.Assignments {
			if assignment.ProjectID != projectID {
				continue
			}
			if targetType != "" && assignment.TargetType != targetType {
				continue
			}
			if targetID != "" && assignment.TargetID != targetID {
				continue
			}
			result = append(result, assignment)
		}
		slices.SortFunc(result, func(a, b Assignment) int {
			return strings.Compare(a.ID, b.ID)
		})
		return nil
	})
	return result, err
}

func (s *Service) ListArtifacts(userID, projectID string) ([]Artifact, error) {
	var result []Artifact
	err := s.readProject(userID, projectID, ActionArtifactRead, func(st state, _ Project) error {
		for _, artifact := range st.Artifacts {
			if artifact.ProjectID == projectID {
				result = append(result, artifact)
			}
		}
		slices.SortFunc(result, func(a, b Artifact) int {
			return strings.Compare(a.ID, b.ID)
		})
		return nil
	})
	return result, err
}

func (s *Service) ListDocuments(userID, projectID string) ([]Document, error) {
	var result []Document
	err := s.readProject(userID, projectID, ActionDocumentRead, func(st state, _ Project) error {
		for _, document := range st.Documents {
			if document.ProjectID == projectID {
				result = append(result, document)
			}
		}
		slices.SortFunc(result, func(a, b Document) int {
			if a.Key != b.Key {
				return strings.Compare(a.Key, b.Key)
			}
			return strings.Compare(a.ID, b.ID)
		})
		return nil
	})
	return result, err
}

func (s *Service) ListComponents(userID, projectID string) ([]Component, error) {
	var result []Component
	err := s.readProject(userID, projectID, ActionComponentRead, func(st state, _ Project) error {
		for _, component := range st.Components {
			if component.ProjectID == projectID {
				result = append(result, component)
			}
		}
		slices.SortFunc(result, func(a, b Component) int {
			return strings.Compare(a.ID, b.ID)
		})
		return nil
	})
	return result, err
}

func (s *Service) ListPolicyBundles(userID, projectID string) ([]PolicyBundle, error) {
	var result []PolicyBundle
	err := s.readProject(userID, projectID, ActionPolicyRead, func(st state, project Project) error {
		for _, policy := range st.Policies {
			if policyAppliesToProject(&st, policy, project) {
				if err := validateStoredPolicy(policy); err != nil {
					return err
				}
				result = append(result, policy)
			}
		}
		slices.SortFunc(result, func(a, b PolicyBundle) int {
			if policyScopeRank(a.Scope) != policyScopeRank(b.Scope) {
				return policyScopeRank(a.Scope) - policyScopeRank(b.Scope)
			}
			if a.Name != b.Name {
				return strings.Compare(a.Name, b.Name)
			}
			return a.Version - b.Version
		})
		return nil
	})
	return result, err
}

func requireTeamMember(
	st *state,
	userID, teamID string,
) (*TeamMembership, error) {
	if err := requireActiveUser(st, userID); err != nil {
		return nil, err
	}
	if _, ok := st.Teams[teamID]; !ok {
		return nil, entityNotFound("team", teamID)
	}
	membership := findTeamMembership(st, teamID, userID)
	if membership == nil || membership.Status != MembershipActive {
		return nil, ErrForbidden
	}
	return membership, nil
}
