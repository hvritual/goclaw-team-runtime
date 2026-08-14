package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
)

type IssueCatalogHandler struct {
	service      contract.IssueCatalogService
	identity     contract.WorkspaceHTTPIdentityResolver
	authenticate func(*http.Request) (string, error)
	mutation     func(*http.Request) error
}

func NewIssueCatalogHandler(service contract.IssueCatalogService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error) *IssueCatalogHandler {
	return &IssueCatalogHandler{service: service, identity: identity, authenticate: authenticate, mutation: mutation}
}

func (h *IssueCatalogHandler) Register(server *kratoshttp.Server) {
	router := server.Route("/")
	router.GET("/api/labels", h.listLabels)
	router.POST("/api/labels", h.createLabel)
	router.GET("/api/labels/{id}", h.getLabel)
	router.PUT("/api/labels/{id}", h.updateLabel)
	router.DELETE("/api/labels/{id}", h.deleteLabel)
	router.GET("/api/issues/{id}/labels", h.listIssueLabels)
	router.POST("/api/issues/{id}/labels", h.attachIssueLabel)
	router.DELETE("/api/issues/{id}/labels/{labelId}", h.detachIssueLabel)

	router.GET("/api/properties", h.listProperties)
	router.POST("/api/properties", h.createProperty)
	router.GET("/api/properties/{id}", h.getProperty)
	router.PATCH("/api/properties/{id}", h.updateProperty)
	router.PUT("/api/issues/{id}/properties/{propertyId}", h.setIssueProperty)
	router.DELETE("/api/issues/{id}/properties/{propertyId}", h.unsetIssueProperty)

	router.GET("/api/issues/{id}/acceptance-conclusions", h.listAcceptanceConclusions)
	router.POST("/api/issues/{id}/acceptance-conclusions", h.createAcceptanceConclusion)
}

type createLabelHTTPRequest struct {
	ResourceType string `json:"resource_type"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Color        string `json:"color"`
}

type updateLabelHTTPRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Color       *string `json:"color"`
}

type attachLabelHTTPRequest struct {
	LabelID string `json:"label_id"`
}

type propertyConfigHTTPRequest struct {
	Options []contract.IssuePropertyOption `json:"options"`
}

type createPropertyHTTPRequest struct {
	Name        string                     `json:"name"`
	Type        string                     `json:"type"`
	Description string                     `json:"description"`
	Icon        string                     `json:"icon"`
	Config      *propertyConfigHTTPRequest `json:"config"`
}

type updatePropertyHTTPRequest struct {
	Name        *string                    `json:"name"`
	Description *string                    `json:"description"`
	Icon        *string                    `json:"icon"`
	Config      *propertyConfigHTTPRequest `json:"config"`
	Archived    *bool                      `json:"archived"`
}

type setPropertyHTTPRequest struct {
	Value json.RawMessage `json:"value"`
}

type acceptanceConclusionHTTPRequest struct {
	Result       string   `json:"result"`
	Rationale    string   `json:"rationale"`
	EvidenceRefs []string `json:"evidence_refs"`
}

func (h *IssueCatalogHandler) listLabels(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.readIdentity(ctx)
	if !ok {
		return nil
	}
	resourceType := strings.TrimSpace(ctx.Request().URL.Query().Get("resource_type"))
	if resourceType != "" && resourceType != "issue" {
		return writeError(ctx, http.StatusNotFound, "resource labels are not enabled")
	}
	values, err := h.service.ListIssueLabels(requestContext, workspaceID)
	if err != nil {
		return h.writeCatalogError(ctx, err, "label", "list labels")
	}
	labels := make([]map[string]any, len(values))
	for index := range values {
		labels[index] = publicIssueLabel(values[index])
	}
	return ctx.JSON(http.StatusOK, map[string]any{"labels": labels, "total": len(labels)})
}

func (h *IssueCatalogHandler) getLabel(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.readIdentity(ctx)
	if !ok {
		return nil
	}
	value, err := h.service.GetIssueLabel(requestContext, workspaceID, ctx.Vars().Get("id"))
	if err != nil {
		return h.writeCatalogError(ctx, err, "label", "get label")
	}
	return ctx.JSON(http.StatusOK, publicIssueLabel(value))
}

func (h *IssueCatalogHandler) createLabel(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.mutationIdentity(ctx)
	if !ok {
		return nil
	}
	var request createLabelHTTPRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	value, err := h.service.CreateIssueLabel(requestContext, contract.CreateIssueLabelRequest{WorkspaceID: workspaceID, ResourceType: request.ResourceType, Name: request.Name, Description: request.Description, Color: request.Color})
	if err != nil {
		return h.writeCatalogError(ctx, err, "label", "create label")
	}
	return ctx.JSON(http.StatusCreated, publicIssueLabel(value))
}

func (h *IssueCatalogHandler) updateLabel(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.mutationIdentity(ctx)
	if !ok {
		return nil
	}
	if _, err := h.service.GetIssueLabel(requestContext, workspaceID, ctx.Vars().Get("id")); err != nil {
		return h.writeCatalogError(ctx, err, "label", "update label")
	}
	var request updateLabelHTTPRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	value, err := h.service.UpdateIssueLabel(requestContext, contract.UpdateIssueLabelRequest{WorkspaceID: workspaceID, LabelID: ctx.Vars().Get("id"), Name: request.Name, Description: request.Description, Color: request.Color})
	if err != nil {
		return h.writeCatalogError(ctx, err, "label", "update label")
	}
	return ctx.JSON(http.StatusOK, publicIssueLabel(value))
}

func (h *IssueCatalogHandler) deleteLabel(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.mutationIdentity(ctx)
	if !ok {
		return nil
	}
	if err := h.service.DeleteIssueLabel(requestContext, workspaceID, ctx.Vars().Get("id")); err != nil {
		return h.writeCatalogError(ctx, err, "label", "delete label")
	}
	ctx.Response().WriteHeader(http.StatusNoContent)
	return nil
}

func (h *IssueCatalogHandler) listIssueLabels(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.readIdentity(ctx)
	if !ok {
		return nil
	}
	_, values, err := h.service.ListLabelsForIssue(requestContext, workspaceID, ctx.Vars().Get("id"))
	if err != nil {
		return h.writeCatalogError(ctx, err, "issue", "list labels")
	}
	return ctx.JSON(http.StatusOK, map[string]any{"labels": publicIssueLabels(values)})
}

func (h *IssueCatalogHandler) attachIssueLabel(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.mutationIdentity(ctx)
	if !ok {
		return nil
	}
	var request attachLabelHTTPRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	_, values, err := h.service.AttachLabelToIssue(requestContext, workspaceID, ctx.Vars().Get("id"), request.LabelID)
	if err != nil {
		return h.writeCatalogError(ctx, err, "label", "attach label")
	}
	return ctx.JSON(http.StatusOK, map[string]any{"labels": publicIssueLabels(values)})
}

func (h *IssueCatalogHandler) detachIssueLabel(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.mutationIdentity(ctx)
	if !ok {
		return nil
	}
	_, values, err := h.service.DetachLabelFromIssue(requestContext, workspaceID, ctx.Vars().Get("id"), ctx.Vars().Get("labelId"))
	if err != nil {
		return h.writeCatalogError(ctx, err, "label", "detach label")
	}
	return ctx.JSON(http.StatusOK, map[string]any{"labels": publicIssueLabels(values)})
}

func (h *IssueCatalogHandler) listProperties(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.readIdentity(ctx)
	if !ok {
		return nil
	}
	values, err := h.service.ListIssueProperties(requestContext, workspaceID, ctx.Request().URL.Query().Get("include_archived") == "true")
	if err != nil {
		return h.writeCatalogError(ctx, err, "property", "list properties")
	}
	properties := make([]map[string]any, len(values))
	for index := range values {
		properties[index] = publicIssueProperty(values[index])
	}
	return ctx.JSON(http.StatusOK, map[string]any{"properties": properties, "total": len(properties)})
}

func (h *IssueCatalogHandler) getProperty(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.readIdentity(ctx)
	if !ok {
		return nil
	}
	value, err := h.service.GetIssueProperty(requestContext, workspaceID, ctx.Vars().Get("id"))
	if err != nil {
		return h.writeCatalogError(ctx, err, "property", "get property")
	}
	return ctx.JSON(http.StatusOK, publicIssueProperty(value))
}

func (h *IssueCatalogHandler) createProperty(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.mutationIdentity(ctx)
	if !ok {
		return nil
	}
	var request createPropertyHTTPRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	value, err := h.service.CreateIssueProperty(requestContext, contract.CreateIssuePropertyRequest{WorkspaceID: workspaceID, Name: request.Name, Type: request.Type, Description: request.Description, Icon: request.Icon, Config: request.propertyConfig()})
	if err != nil {
		return h.writeCatalogError(ctx, err, "property", "create property")
	}
	return ctx.JSON(http.StatusCreated, publicIssueProperty(value))
}

func (h *IssueCatalogHandler) updateProperty(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.mutationIdentity(ctx)
	if !ok {
		return nil
	}
	if _, err := h.service.GetIssueProperty(requestContext, workspaceID, ctx.Vars().Get("id")); err != nil {
		return h.writeCatalogError(ctx, err, "property", "update property")
	}
	var request updatePropertyHTTPRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	value, err := h.service.UpdateIssueProperty(requestContext, contract.UpdateIssuePropertyRequest{WorkspaceID: workspaceID, PropertyID: ctx.Vars().Get("id"), Name: request.Name, Description: request.Description, Icon: request.Icon, Config: request.propertyConfig(), Archived: request.Archived})
	if err != nil {
		return h.writeCatalogError(ctx, err, "property", "update property")
	}
	return ctx.JSON(http.StatusOK, publicIssueProperty(value))
}

func (h *IssueCatalogHandler) setIssueProperty(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.mutationIdentity(ctx)
	if !ok {
		return nil
	}
	var request setPropertyHTTPRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	_, values, err := h.service.SetIssueProperty(requestContext, workspaceID, ctx.Vars().Get("id"), ctx.Vars().Get("propertyId"), request.Value)
	if err != nil {
		return h.writeCatalogError(ctx, err, "property", "set property")
	}
	return ctx.JSON(http.StatusOK, map[string]any{"properties": values})
}

func (h *IssueCatalogHandler) unsetIssueProperty(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.mutationIdentity(ctx)
	if !ok {
		return nil
	}
	_, values, err := h.service.UnsetIssueProperty(requestContext, workspaceID, ctx.Vars().Get("id"), ctx.Vars().Get("propertyId"))
	if err != nil {
		return h.writeCatalogError(ctx, err, "property", "unset property")
	}
	return ctx.JSON(http.StatusOK, map[string]any{"properties": values})
}

func (h *IssueCatalogHandler) listAcceptanceConclusions(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.readIdentity(ctx)
	if !ok {
		return nil
	}
	_, values, err := h.service.ListAcceptanceConclusions(requestContext, workspaceID, ctx.Vars().Get("id"))
	if err != nil {
		return h.writeCatalogError(ctx, err, "issue", "list acceptance conclusions")
	}
	conclusions := make([]map[string]any, len(values))
	for index := range values {
		conclusions[index] = publicAcceptanceConclusion(values[index])
	}
	return ctx.JSON(http.StatusOK, map[string]any{"acceptance_conclusions": conclusions, "total": len(conclusions)})
}

func (h *IssueCatalogHandler) createAcceptanceConclusion(ctx kratoshttp.Context) error {
	requestContext, workspaceID, ok := h.mutationIdentity(ctx)
	if !ok {
		return nil
	}
	if _, _, err := h.service.ListAcceptanceConclusions(requestContext, workspaceID, ctx.Vars().Get("id")); err != nil {
		return h.writeCatalogError(ctx, err, "issue", "record acceptance conclusion")
	}
	var request acceptanceConclusionHTTPRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	value, err := h.service.CreateAcceptanceConclusion(requestContext, workspaceID, ctx.Vars().Get("id"), request.contract())
	if err != nil {
		return h.writeCatalogError(ctx, err, "issue", "record acceptance conclusion")
	}
	return ctx.JSON(http.StatusCreated, publicAcceptanceConclusion(value.Conclusion))
}

func (h *IssueCatalogHandler) readIdentity(ctx kratoshttp.Context) (context.Context, string, bool) {
	if h.authenticate == nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return nil, "", false
	}
	if _, err := h.authenticate(ctx.Request()); err != nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return nil, "", false
	}
	if !hasWorkspaceIdentity(ctx) {
		_ = writeError(ctx, http.StatusBadRequest, "workspace is required")
		return nil, "", false
	}
	if h.identity == nil {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return nil, "", false
	}
	identity, err := h.identity(ctx.Request())
	if err != nil {
		_ = issueReadIdentityError(ctx, err)
		return nil, "", false
	}
	if strings.TrimSpace(identity.WorkspaceID) == "" || strings.TrimSpace(identity.ActorID) == "" {
		_ = writeError(ctx, http.StatusUnauthorized, "user not authenticated")
		return nil, "", false
	}
	return contract.WithWorkspaceActor(ctx.Request().Context(), identity.ActorType, identity.ActorID), identity.WorkspaceID, true
}

func (h *IssueCatalogHandler) mutationIdentity(ctx kratoshttp.Context) (context.Context, string, bool) {
	requestContext, workspaceID, ok := h.readIdentity(ctx)
	if !ok {
		return nil, "", false
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		_ = writeError(ctx, http.StatusForbidden, "invalid CSRF token")
		return nil, "", false
	}
	return requestContext, workspaceID, true
}

func (h *IssueCatalogHandler) writeCatalogError(ctx kratoshttp.Context, err error, target, operation string) error {
	var optionsInUse *application.IssuePropertyOptionsInUseError
	if errors.As(err, &optionsInUse) {
		return writeError(ctx, http.StatusConflict, optionsInUse.Error())
	}
	if errors.Is(err, application.ErrIssueCatalogInvalid) || errors.Is(err, contract.ErrInvalidIssue) {
		return writeError(ctx, http.StatusBadRequest, "invalid request")
	}
	if errors.Is(err, contract.ErrWorkspacePermissionDenied) {
		return writeError(ctx, http.StatusForbidden, "insufficient permissions")
	}
	if errors.Is(err, application.ErrIssueAcceptanceConflict) {
		return writeError(ctx, http.StatusConflict, "issue must be done before recording an acceptance conclusion")
	}
	if errors.Is(err, application.ErrIssuePropertyLimit) {
		return writeError(ctx, http.StatusBadRequest, "a workspace cannot have more than 20 active properties; archive unused ones first")
	}
	if errors.Is(err, application.ErrIssueCatalogConflict) {
		if target == "label" {
			return writeError(ctx, http.StatusConflict, "a label with that name already exists")
		}
		return writeError(ctx, http.StatusConflict, "a property with that name already exists")
	}
	if errors.Is(err, application.ErrIssueLabelNotFound) {
		return writeError(ctx, http.StatusNotFound, "label not found")
	}
	if errors.Is(err, application.ErrIssuePropertyNotFound) {
		return writeError(ctx, http.StatusNotFound, "property not found")
	}
	if errors.Is(err, application.ErrIssueRecordNotFound) || errors.Is(err, contract.ErrIssueNotFound) || errors.Is(err, contract.ErrActorOutsideWorkspace) || errors.Is(err, contract.ErrWorkspaceNotFound) {
		return writeError(ctx, http.StatusNotFound, target+" not found")
	}
	return writeError(ctx, http.StatusInternalServerError, "failed to "+operation)
}

func (r createPropertyHTTPRequest) propertyConfig() *contract.IssuePropertyConfig {
	if r.Config == nil {
		return nil
	}
	return &contract.IssuePropertyConfig{Options: r.Config.Options}
}

func (r updatePropertyHTTPRequest) propertyConfig() *contract.IssuePropertyConfig {
	if r.Config == nil {
		return nil
	}
	return &contract.IssuePropertyConfig{Options: r.Config.Options}
}

func (r acceptanceConclusionHTTPRequest) contract() contract.AcceptanceConclusionInput {
	return contract.AcceptanceConclusionInput{Result: r.Result, Rationale: r.Rationale, EvidenceRefs: r.EvidenceRefs}
}

func publicIssueLabels(values []contract.IssueLabel) []map[string]any {
	result := make([]map[string]any, len(values))
	for index := range values {
		result[index] = publicIssueLabel(values[index])
	}
	return result
}

func publicIssueLabel(value contract.IssueLabel) map[string]any {
	return map[string]any{"id": value.ID, "workspace_id": value.WorkspaceID, "resource_type": value.ResourceType, "name": value.Name, "description": value.Description, "color": value.Color, "usage_count": value.UsageCount, "created_at": value.CreatedAt, "updated_at": value.UpdatedAt}
}

func publicIssueProperty(value contract.IssuePropertyDefinition) map[string]any {
	options := value.Config.Options
	if options == nil {
		options = []contract.IssuePropertyOption{}
	}
	return map[string]any{"id": value.ID, "workspace_id": value.WorkspaceID, "name": value.Name, "type": value.Type, "description": value.Description, "icon": value.Icon, "config": map[string]any{"options": options}, "position": value.Position, "archived": value.Archived, "archived_at": value.ArchivedAt, "usage_count": value.UsageCount, "created_at": value.CreatedAt, "updated_at": value.UpdatedAt}
}

func publicAcceptanceConclusion(value contract.AcceptanceConclusion) map[string]any {
	refs := value.EvidenceRefs
	if refs == nil {
		refs = []string{}
	}
	return map[string]any{"id": value.ID, "workspace_id": value.WorkspaceID, "issue_id": value.IssueID, "result": value.Result, "rationale": value.Rationale, "evidence_refs": refs, "actor_id": value.ActorID, "created_at": value.CreatedAt, "updated_at": value.UpdatedAt}
}
