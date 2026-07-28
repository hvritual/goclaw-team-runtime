package teamcontrol

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

func (s *Service) RegisterArtifact(
	actorID string,
	input RegisterArtifactInput,
) (Artifact, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return Artifact{}, err
	}
	projectID, err := requireID(input.ProjectID, "project_id")
	if err != nil {
		return Artifact{}, err
	}
	id, err := normalizeID(input.ID, "artifact")
	if err != nil {
		return Artifact{}, err
	}
	if !validArtifactKind(input.Kind) {
		return Artifact{}, fmt.Errorf("unsupported artifact kind %q", input.Kind)
	}
	resourceType := input.ResourceType
	if resourceType == "" {
		resourceType = resourceTypeForArtifactKind(input.Kind)
	}
	if !artifactBackedResourceType(resourceType) {
		return Artifact{}, fmt.Errorf(
			"resource type %q cannot be registered as an artifact",
			resourceType,
		)
	}
	name, err := requireText(input.Name, "name", 300)
	if err != nil {
		return Artifact{}, err
	}
	uri, err := validateURI(input.URI, "uri")
	if err != nil {
		return Artifact{}, err
	}
	checksum, err := validateOptionalSHA256(input.SHA256)
	if err != nil {
		return Artifact{}, err
	}
	contentType, err := optionalText(input.ContentType, "content_type", 300)
	if err != nil {
		return Artifact{}, err
	}
	metadata, err := cleanStringMap(input.Metadata)
	if err != nil {
		return Artifact{}, err
	}
	var created Artifact
	err = s.store.update(func(st *state) error {
		if err := authorizeProject(st, actorID, projectID, ActionArtifactWrite); err != nil {
			return err
		}
		if _, exists := st.Artifacts[id]; exists {
			return conflict("artifact %q already exists", id)
		}
		for _, artifact := range st.Artifacts {
			if artifact.ProjectID == projectID && artifact.URI == uri &&
				artifact.SHA256 == checksum {
				return conflict("artifact URI and checksum are already registered")
			}
		}
		created = Artifact{
			ID:           id,
			ProjectID:    projectID,
			ResourceType: resourceType,
			Kind:         input.Kind,
			Name:         name,
			URI:          uri,
			SHA256:       checksum,
			ContentType:  contentType,
			Metadata:     metadata,
			CreatedBy:    actorID,
			CreatedAt:    time.Now().UTC(),
		}
		st.Artifacts[id] = created
		return nil
	})
	return created, err
}

func (s *Service) GetArtifact(
	userID, projectID, artifactID string,
) (Artifact, error) {
	artifactID, err := requireID(artifactID, "artifact_id")
	if err != nil {
		return Artifact{}, err
	}
	var result Artifact
	err = s.readProject(userID, projectID, ActionArtifactRead, func(st state, _ Project) error {
		artifact, ok := st.Artifacts[artifactID]
		if !ok || artifact.ProjectID != projectID {
			return entityNotFound("artifact", artifactID)
		}
		result = artifact
		return nil
	})
	return result, err
}

func (s *Service) CreateLink(
	actorID string,
	input CreateLinkInput,
) (CorrelationLink, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return CorrelationLink{}, err
	}
	projectID, err := requireID(input.ProjectID, "project_id")
	if err != nil {
		return CorrelationLink{}, err
	}
	id, err := normalizeID(input.ID, "link")
	if err != nil {
		return CorrelationLink{}, err
	}
	if !validResourceType(input.SourceType) || !validResourceType(input.TargetType) {
		return CorrelationLink{}, fmt.Errorf("unsupported link endpoint type")
	}
	sourceID, err := requireID(input.SourceID, "source_id")
	if err != nil {
		return CorrelationLink{}, err
	}
	targetID, err := requireID(input.TargetID, "target_id")
	if err != nil {
		return CorrelationLink{}, err
	}
	if input.SourceType == input.TargetType && sourceID == targetID {
		return CorrelationLink{}, fmt.Errorf("link cannot point a resource to itself")
	}
	relation, err := requireKey(input.Relation, "relation")
	if err != nil {
		return CorrelationLink{}, err
	}
	metadata, err := cleanStringMap(input.Metadata)
	if err != nil {
		return CorrelationLink{}, err
	}
	var created CorrelationLink
	err = s.store.update(func(st *state) error {
		if err := authorizeProject(st, actorID, projectID, ActionArtifactWrite); err != nil {
			return err
		}
		if _, exists := st.Links[id]; exists {
			return conflict("link %q already exists", id)
		}
		sourceProject, err := resourceProject(st, input.SourceType, sourceID)
		if err != nil {
			return err
		}
		targetProject, err := resourceProject(st, input.TargetType, targetID)
		if err != nil {
			return err
		}
		if sourceProject != projectID || targetProject != projectID {
			return fmt.Errorf(
				"%w: correlation endpoints must both belong to project %q",
				ErrForbidden,
				projectID,
			)
		}
		for _, link := range st.Links {
			if link.ProjectID == projectID &&
				link.SourceType == input.SourceType && link.SourceID == sourceID &&
				link.TargetType == input.TargetType && link.TargetID == targetID &&
				link.Relation == relation {
				return conflict("equivalent correlation link already exists")
			}
		}
		created = CorrelationLink{
			ID:         id,
			ProjectID:  projectID,
			SourceType: input.SourceType,
			SourceID:   sourceID,
			TargetType: input.TargetType,
			TargetID:   targetID,
			Relation:   relation,
			Metadata:   metadata,
			CreatedBy:  actorID,
			CreatedAt:  time.Now().UTC(),
		}
		st.Links[id] = created
		return nil
	})
	return created, err
}

func (s *Service) ListLinks(
	userID, projectID string,
	resourceType ResourceType,
	resourceID string,
) ([]CorrelationLink, error) {
	if !validResourceType(resourceType) {
		return nil, fmt.Errorf("unsupported resource type %q", resourceType)
	}
	resourceID, err := requireID(resourceID, "resource_id")
	if err != nil {
		return nil, err
	}
	var result []CorrelationLink
	err = s.readProject(userID, projectID, ActionArtifactRead, func(st state, _ Project) error {
		resolvedProject, err := resourceProject(&st, resourceType, resourceID)
		if err != nil || resolvedProject != projectID {
			return entityNotFound(string(resourceType), resourceID)
		}
		for _, link := range st.Links {
			if link.ProjectID == projectID &&
				((link.SourceType == resourceType && link.SourceID == resourceID) ||
					(link.TargetType == resourceType && link.TargetID == resourceID)) {
				result = append(result, link)
			}
		}
		slices.SortFunc(result, func(a, b CorrelationLink) int {
			return strings.Compare(a.ID, b.ID)
		})
		return nil
	})
	return result, err
}

func (s *Service) RegisterDocument(
	actorID string,
	input RegisterDocumentInput,
) (Document, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return Document{}, err
	}
	projectID, err := requireID(input.ProjectID, "project_id")
	if err != nil {
		return Document{}, err
	}
	id, err := normalizeID(input.ID, "doc")
	if err != nil {
		return Document{}, err
	}
	key, err := requireKey(input.Key, "key")
	if err != nil {
		return Document{}, err
	}
	key = strings.ToLower(key)
	title, err := requireText(input.Title, "title", 300)
	if err != nil {
		return Document{}, err
	}
	if !validDocumentKind(input.Kind) {
		return Document{}, fmt.Errorf("unsupported document kind %q", input.Kind)
	}
	status := input.Status
	if status == "" {
		status = DocumentDraft
	}
	switch status {
	case DocumentDraft, DocumentActive, DocumentArchived:
	default:
		return Document{}, fmt.Errorf("new document status %q is not allowed", status)
	}
	uri, err := validateURI(input.URI, "uri")
	if err != nil {
		return Document{}, err
	}
	revision, err := optionalText(input.Revision, "revision", 200)
	if err != nil {
		return Document{}, err
	}
	checksum, err := validateOptionalSHA256(input.SHA256)
	if err != nil {
		return Document{}, err
	}
	ownerID := strings.TrimSpace(input.OwnerID)
	if ownerID != "" {
		if ownerID, err = requireID(ownerID, "owner_id"); err != nil {
			return Document{}, err
		}
	}
	supersedes := strings.TrimSpace(input.Supersedes)
	if supersedes != "" {
		if supersedes, err = requireID(supersedes, "supersedes"); err != nil {
			return Document{}, err
		}
	}
	var created Document
	err = s.store.update(func(st *state) error {
		if err := authorizeProject(st, actorID, projectID, ActionDocumentWrite); err != nil {
			return err
		}
		if _, exists := st.Documents[id]; exists {
			return conflict("document %q already exists", id)
		}
		if ownerID != "" {
			if membership := findProjectMembership(st, projectID, ownerID); membership == nil ||
				membership.Status != MembershipActive {
				return fmt.Errorf("%w: document owner is not an active project member", ErrForbidden)
			}
		}
		if supersedes != "" {
			previous, ok := st.Documents[supersedes]
			if !ok || previous.ProjectID != projectID {
				return entityNotFound("document", supersedes)
			}
			if previous.Key != key {
				return conflict("superseded document has a different key")
			}
			if status != DocumentActive {
				return conflict("only an active document may supersede another version")
			}
		}
		for _, document := range st.Documents {
			if document.ProjectID == projectID && document.Key == key &&
				document.Revision == revision && revision != "" {
				return conflict("document key %q revision %q already exists", key, revision)
			}
			if status == DocumentActive && document.ProjectID == projectID &&
				document.Key == key && document.Status == DocumentActive &&
				document.ID != supersedes {
				return conflict("document key %q already has an active version", key)
			}
		}
		now := time.Now().UTC()
		created = Document{
			ID:         id,
			ProjectID:  projectID,
			Key:        key,
			Title:      title,
			Kind:       input.Kind,
			Status:     status,
			URI:        uri,
			Revision:   revision,
			SHA256:     checksum,
			OwnerID:    ownerID,
			Supersedes: supersedes,
			CreatedBy:  actorID,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if supersedes != "" {
			previous := st.Documents[supersedes]
			previous.Status = DocumentSuperseded
			previous.UpdatedAt = now
			st.Documents[supersedes] = previous
		}
		st.Documents[id] = created
		return nil
	})
	return created, err
}

func (s *Service) GetDocument(
	userID, projectID, documentID string,
) (Document, error) {
	documentID, err := requireID(documentID, "document_id")
	if err != nil {
		return Document{}, err
	}
	var result Document
	err = s.readProject(userID, projectID, ActionDocumentRead, func(st state, _ Project) error {
		document, ok := st.Documents[documentID]
		if !ok || document.ProjectID != projectID {
			return entityNotFound("document", documentID)
		}
		result = document
		return nil
	})
	return result, err
}

func (s *Service) RegisterComponent(
	actorID string,
	input RegisterComponentInput,
) (Component, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return Component{}, err
	}
	projectID, err := requireID(input.ProjectID, "project_id")
	if err != nil {
		return Component{}, err
	}
	id, err := normalizeID(input.ID, "component")
	if err != nil {
		return Component{}, err
	}
	name, err := requireText(input.Name, "name", 300)
	if err != nil {
		return Component{}, err
	}
	if !validComponentKind(input.Kind) {
		return Component{}, fmt.Errorf("unsupported component kind %q", input.Kind)
	}
	repositoryID := strings.TrimSpace(input.RepositoryID)
	if repositoryID != "" {
		if repositoryID, err = requireID(repositoryID, "repository_id"); err != nil {
			return Component{}, err
		}
	}
	rootPath, err := validateRelativePath(input.RootPath, "root_path")
	if err != nil {
		return Component{}, err
	}
	description, err := optionalText(input.Description, "description", 10_000)
	if err != nil {
		return Component{}, err
	}
	ownerIDs, err := uniqueIDs(input.OwnerIDs, "owner_id")
	if err != nil {
		return Component{}, err
	}
	dependencyIDs, err := uniqueIDs(input.DependencyIDs, "dependency_id")
	if err != nil {
		return Component{}, err
	}
	if slices.Contains(dependencyIDs, id) {
		return Component{}, fmt.Errorf("component cannot depend on itself")
	}
	metadata, err := cleanStringMap(input.Metadata)
	if err != nil {
		return Component{}, err
	}
	var created Component
	err = s.store.update(func(st *state) error {
		if err := authorizeProject(st, actorID, projectID, ActionComponentWrite); err != nil {
			return err
		}
		if _, exists := st.Components[id]; exists {
			return conflict("component %q already exists", id)
		}
		for _, component := range st.Components {
			if component.ProjectID == projectID && strings.EqualFold(component.Name, name) {
				return conflict("component name %q already exists in project", name)
			}
		}
		if repositoryID != "" {
			repository, ok := st.Repositories[repositoryID]
			if !ok || repository.ProjectID != projectID {
				return entityNotFound("repository", repositoryID)
			}
		}
		for _, ownerID := range ownerIDs {
			membership := findProjectMembership(st, projectID, ownerID)
			if membership == nil || membership.Status != MembershipActive {
				return fmt.Errorf("%w: component owner %q is not a project member", ErrForbidden, ownerID)
			}
		}
		if err := requireProjectComponents(st, projectID, dependencyIDs); err != nil {
			return err
		}
		now := time.Now().UTC()
		created = Component{
			ID:            id,
			ProjectID:     projectID,
			RepositoryID:  repositoryID,
			Name:          name,
			Kind:          input.Kind,
			RootPath:      rootPath,
			Description:   description,
			OwnerIDs:      ownerIDs,
			DependencyIDs: dependencyIDs,
			Metadata:      metadata,
			CreatedBy:     actorID,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		st.Components[id] = created
		return nil
	})
	return created, err
}

func (s *Service) GetComponent(
	userID, projectID, componentID string,
) (Component, error) {
	componentID, err := requireID(componentID, "component_id")
	if err != nil {
		return Component{}, err
	}
	var result Component
	err = s.readProject(userID, projectID, ActionComponentRead, func(st state, _ Project) error {
		component, ok := st.Components[componentID]
		if !ok || component.ProjectID != projectID {
			return entityNotFound("component", componentID)
		}
		result = component
		return nil
	})
	return result, err
}

func resourceProject(st *state, kind ResourceType, id string) (string, error) {
	switch kind {
	case ResourceIssue:
		value, ok := st.Issues[id]
		if ok {
			return value.ProjectID, nil
		}
	case ResourceWorkItem:
		value, ok := st.WorkItems[id]
		if ok {
			return value.ProjectID, nil
		}
	case ResourceArtifact:
		value, ok := st.Artifacts[id]
		if ok && value.ResourceType == ResourceArtifact {
			return value.ProjectID, nil
		}
	case ResourceTask, ResourceRun, ResourceTrace, ResourceCommit,
		ResourcePullRequest, ResourceCI, ResourceRelease, ResourceRegressionCase:
		value, ok := st.Artifacts[id]
		if ok && value.ResourceType == kind {
			return value.ProjectID, nil
		}
	case ResourceSpec:
		if value, ok := st.Artifacts[id]; ok && value.ResourceType == ResourceSpec {
			return value.ProjectID, nil
		}
		if value, ok := st.Documents[id]; ok {
			return value.ProjectID, nil
		}
	case ResourceDocument:
		value, ok := st.Documents[id]
		if ok {
			return value.ProjectID, nil
		}
	case ResourceComponent:
		value, ok := st.Components[id]
		if ok {
			return value.ProjectID, nil
		}
	case ResourceRepository:
		value, ok := st.Repositories[id]
		if ok {
			return value.ProjectID, nil
		}
	case ResourcePolicy:
		value, ok := st.Policies[id]
		if ok && value.ProjectID != "" {
			return value.ProjectID, nil
		}
	}
	return "", entityNotFound(string(kind), id)
}

func artifactBackedResourceType(value ResourceType) bool {
	switch value {
	case ResourceTask, ResourceRun, ResourceTrace, ResourceCommit,
		ResourcePullRequest, ResourceCI, ResourceRelease, ResourceRegressionCase,
		ResourceSpec, ResourceArtifact:
		return true
	default:
		return false
	}
}

func resourceTypeForArtifactKind(kind ArtifactKind) ResourceType {
	switch kind {
	case ArtifactTrace:
		return ResourceTrace
	case ArtifactCommit:
		return ResourceCommit
	case ArtifactPR:
		return ResourcePullRequest
	default:
		return ResourceArtifact
	}
}
