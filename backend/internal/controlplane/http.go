package controlplane

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const apiSchemaVersion = 1

type HTTPAPI struct {
	service  *Service
	kernel   *DeliveryKernel
	flows    *P2Flows
	identity IdentityResolver
}

type CommandRequest struct {
	Type         string          `json:"type"`
	CommandID    string          `json:"command_id"`
	ExpectedHead int64           `json:"expected_head"`
	Payload      json.RawMessage `json:"payload"`
}

type IDText struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}
type DoneCommand struct {
	SubjectID string   `json:"subject_id"`
	Revision  int64    `json:"revision"`
	Policies  []string `json:"policies"`
}
type RunCommand struct {
	ID           string   `json:"id"`
	WorkspaceRef string   `json:"workspace_ref"`
	SecretRefs   []string `json:"secret_refs"`
	MaxAttempts  int      `json:"max_attempts"`
}

type WorkspaceResponse struct {
	SchemaVersion int       `json:"schema_version"`
	Workspace     Workspace `json:"workspace"`
}

type MembersResponse struct {
	SchemaVersion int      `json:"schema_version"`
	Members       []Member `json:"members"`
}

func NewHTTPAPI(service *Service, kernel *DeliveryKernel, flows *P2Flows, identity IdentityResolver) (*HTTPAPI, error) {
	if service == nil || kernel == nil || flows == nil || identity == nil {
		return nil, invalid("new HTTP API", "dependency", "service, kernel, flows, and identity resolver are required")
	}
	return &HTTPAPI{service: service, kernel: kernel, flows: flows, identity: identity}, nil
}

func (a *HTTPAPI) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /v1/workspaces/{workspace}", a.workspace)
	mux.HandleFunc("GET /v1/workspaces/{workspace}/members", a.members)
	mux.HandleFunc("GET /v1/workspaces/{workspace}/projects/{project}/projection", a.projection)
	mux.HandleFunc("GET /v1/workspaces/{workspace}/projects/{project}/events", a.events)
	mux.HandleFunc("POST /v1/workspaces/{workspace}/projects/{project}/commands", a.command)
	return mux
}

func (a *HTTPAPI) actor(request *http.Request) (Actor, error) {
	identity, err := a.identity(request)
	if err != nil {
		return Actor{}, err
	}
	actor := identity.Actor
	if actor.WorkspaceID != request.PathValue("workspace") {
		return Actor{}, denied("resolve HTTP identity", "workspace mismatch")
	}
	if identity.Snapshot != nil {
		if identity.Snapshot.ID != actor.WorkspaceID || identity.Snapshot.ActorID != actor.ID {
			return Actor{}, denied("resolve HTTP identity", "trusted snapshot mismatch")
		}
		if err := a.service.reconcileTrustedSnapshot(request.Context(), *identity.Snapshot); err != nil {
			return Actor{}, err
		}
	}
	return actor, nil
}

func (a *HTTPAPI) workspace(w http.ResponseWriter, request *http.Request) {
	actor, err := a.actor(request)
	if err != nil {
		writeProblem(w, err)
		return
	}
	workspace, err := a.service.GetWorkspace(request.Context(), actor)
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeJSON(w, http.StatusOK, WorkspaceResponse{SchemaVersion: apiSchemaVersion, Workspace: workspace})
}

func (a *HTTPAPI) members(w http.ResponseWriter, request *http.Request) {
	actor, err := a.actor(request)
	if err != nil {
		writeProblem(w, err)
		return
	}
	members, err := a.service.ListMembers(request.Context(), actor, false)
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeJSON(w, http.StatusOK, MembersResponse{SchemaVersion: apiSchemaVersion, Members: members})
}

func (a *HTTPAPI) projection(w http.ResponseWriter, request *http.Request) {
	actor, err := a.actor(request)
	if err != nil {
		writeProblem(w, err)
		return
	}
	if err := a.kernel.allow(request.Context(), actor, PermissionRead); err != nil {
		writeProblem(w, err)
		return
	}
	projection, err := a.kernel.Replay(request.Context(), actor.WorkspaceID, request.PathValue("project"))
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeJSON(w, http.StatusOK, projection)
}

func (a *HTTPAPI) events(response http.ResponseWriter, request *http.Request) {
	actor, err := a.actor(request)
	if err != nil {
		writeProblem(response, err)
		return
	}
	if err := a.kernel.allow(request.Context(), actor, PermissionRead); err != nil {
		writeProblem(response, err)
		return
	}
	after, err := eventCursor(request)
	if err != nil {
		writeProblem(response, err)
		return
	}
	flusher, ok := response.(http.Flusher)
	if !ok {
		writeProblem(response, unavailable("stream project events", "response streaming is unsupported"))
		return
	}
	projectID := request.PathValue("project")
	events, err := a.kernel.store.ListSessionEvents(request.Context(), actor.WorkspaceID, projectID)
	if err != nil {
		writeProblem(response, err)
		return
	}

	response.Header().Set("Content-Type", "text/event-stream")
	response.Header().Set("Cache-Control", "no-cache, no-transform")
	response.Header().Set("Connection", "keep-alive")
	response.Header().Set("X-Accel-Buffering", "no")
	response.WriteHeader(http.StatusOK)
	after, err = writeEventBatch(response, events, after)
	if err != nil {
		return
	}
	_, _ = fmt.Fprint(response, ": connected\n\n")
	flusher.Flush()

	poll := time.NewTicker(time.Second)
	heartbeat := time.NewTicker(15 * time.Second)
	defer poll.Stop()
	defer heartbeat.Stop()
	for {
		select {
		case <-request.Context().Done():
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprint(response, ": heartbeat\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-poll.C:
			events, err := a.kernel.store.ListSessionEvents(request.Context(), actor.WorkspaceID, projectID)
			if err != nil {
				return
			}
			previous := after
			after, err = writeEventBatch(response, events, after)
			if err != nil {
				return
			}
			if after != previous {
				flusher.Flush()
			}
		}
	}
}

func eventCursor(request *http.Request) (int64, error) {
	value := request.URL.Query().Get("after")
	if value == "" {
		value = request.Header.Get("Last-Event-ID")
	}
	if value == "" {
		return 0, nil
	}
	cursor, err := strconv.ParseInt(value, 10, 64)
	if err != nil || cursor < 0 {
		return 0, invalid("stream project events", "after", "must be a non-negative event sequence")
	}
	return cursor, nil
}

func writeEventBatch(response http.ResponseWriter, events []SessionEvent, after int64) (int64, error) {
	for _, event := range events {
		if event.Sequence <= after {
			continue
		}
		payload, err := json.Marshal(event)
		if err != nil {
			return after, err
		}
		if _, err := fmt.Fprintf(response, "id: %d\nevent: session\ndata: %s\n\n", event.Sequence, payload); err != nil {
			return after, err
		}
		after = event.Sequence
	}
	return after, nil
}

func (a *HTTPAPI) command(w http.ResponseWriter, request *http.Request) {
	actor, err := a.actor(request)
	if err != nil {
		writeProblem(w, err)
		return
	}
	var command CommandRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&command); err != nil || command.Type == "" || command.CommandID == "" || !json.Valid(command.Payload) {
		writeProblem(w, invalid("decode command", "body", "a typed command, id, head, and valid payload are required"))
		return
	}
	projectID := request.PathValue("project")
	var result AppendResult
	switch command.Type {
	case "requirement.start":
		var value IDText
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.flows.StartRequirement(request.Context(), actor, command.CommandID, projectID, command.ExpectedHead, value.ID, value.Text)
		}
	case "requirement.intent":
		var value IDText
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.flows.FinalizeIntent(request.Context(), actor, command.CommandID, projectID, command.ExpectedHead, value.ID, value.Text)
		}
	case "requirement.solution":
		var value IDText
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.flows.ProposeSolution(request.Context(), actor, command.CommandID, projectID, command.ExpectedHead, value.ID, value.Text)
		}
	case "requirement.freeze":
		var value DoneCommand
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.flows.FreezeRequirement(request.Context(), actor, command.CommandID, projectID, value.SubjectID, value.Revision, command.ExpectedHead)
		}
	case "requirement.change":
		var value IDText
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.flows.ChangeIntent(request.Context(), actor, command.CommandID, projectID, command.ExpectedHead, value.ID, value.Text)
		}
	case "requirement.task":
		var value struct {
			RequirementID string `json:"requirement_id"`
			TaskID        string `json:"task_id"`
			AssigneeID    string `json:"assignee_id"`
			EdgeCommandID string `json:"edge_command_id"`
		}
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.flows.CreateRequirementTask(request.Context(), actor, command.CommandID, value.EdgeCommandID, projectID, command.ExpectedHead, value.RequirementID, value.TaskID, value.AssigneeID)
		}
	case "defect.create":
		var value struct {
			ID   string      `json:"id"`
			Data QualityData `json:"data"`
		}
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.flows.CreateDefect(request.Context(), actor, command.CommandID, projectID, command.ExpectedHead, value.ID, value.Data)
		}
	case "risk.create":
		var value struct {
			ID   string      `json:"id"`
			Data QualityData `json:"data"`
		}
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.flows.CreateRisk(request.Context(), actor, command.CommandID, projectID, command.ExpectedHead, value.ID, value.Data)
		}
	case "quality.close":
		var value DoneCommand
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.flows.CloseQualityItem(request.Context(), actor, command.CommandID, projectID, value.SubjectID, value.Revision, command.ExpectedHead)
		}
	case "finding.create":
		var value struct {
			ID   string            `json:"id"`
			Data ReviewFindingData `json:"data"`
		}
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.flows.CreateReviewFinding(request.Context(), actor, command.CommandID, projectID, command.ExpectedHead, value.ID, value.Data)
		}
	case "finding.resolve":
		var value DoneCommand
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.flows.ResolveFinding(request.Context(), actor, command.CommandID, projectID, value.SubjectID, value.Revision, command.ExpectedHead)
		}
	case "knowledge.create":
		var value struct {
			ID   string        `json:"id"`
			Data KnowledgeData `json:"data"`
		}
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.flows.CreateKnowledgeCandidate(request.Context(), actor, command.CommandID, projectID, command.ExpectedHead, value.ID, value.Data)
		}
	case "knowledge.publish":
		var value DoneCommand
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.flows.PublishKnowledge(request.Context(), actor, command.CommandID, projectID, value.SubjectID, value.Revision, command.ExpectedHead)
		}
	case "knowledge.invalidate":
		var value struct {
			ID string `json:"id"`
		}
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.flows.InvalidateKnowledge(request.Context(), actor, command.CommandID, projectID, command.ExpectedHead, value.ID)
		}
	case "run.queue":
		var value RunCommand
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.flows.QueueRun(request.Context(), actor, command.CommandID, projectID, command.ExpectedHead, value.ID, value.WorkspaceRef, value.SecretRefs, value.MaxAttempts)
		}
	case "run.claim":
		var value struct {
			ID           string `json:"id"`
			LeaseSeconds int64  `json:"lease_seconds"`
		}
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.flows.ClaimRun(request.Context(), actor, command.CommandID, projectID, command.ExpectedHead, value.ID, time.Duration(value.LeaseSeconds)*time.Second)
		}
	case "run.heartbeat":
		var value struct {
			ID           string `json:"id"`
			LeaseSeconds int64  `json:"lease_seconds"`
		}
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.flows.HeartbeatRun(request.Context(), actor, command.CommandID, projectID, command.ExpectedHead, value.ID, time.Duration(value.LeaseSeconds)*time.Second)
		}
	case "run.complete":
		var value struct {
			ID string `json:"id"`
		}
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.flows.CompleteRun(request.Context(), actor, command.CommandID, projectID, command.ExpectedHead, value.ID)
		}
	case "run.cancel":
		var value struct {
			ID string `json:"id"`
		}
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.flows.CancelRun(request.Context(), actor, command.CommandID, projectID, command.ExpectedHead, value.ID)
		}
	case "run.retry":
		var value struct {
			ID string `json:"id"`
		}
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.flows.RetryRun(request.Context(), actor, command.CommandID, projectID, command.ExpectedHead, value.ID)
		}
	case "evidence.attach":
		var value EvidenceRef
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.kernel.AttachEvidence(request.Context(), actor, command.CommandID, projectID, command.ExpectedHead, value)
		}
	case "check.record":
		var value CheckResult
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.kernel.RecordCheck(request.Context(), actor, command.CommandID, projectID, command.ExpectedHead, value)
		}
	case "done.accept":
		var value DoneCommand
		if err = json.Unmarshal(command.Payload, &value); err == nil {
			result, err = a.kernel.AcceptDone(request.Context(), actor, command.CommandID, projectID, value.SubjectID, value.Revision, command.ExpectedHead, value.Policies)
		}
	default:
		err = invalid("execute command", "type", "is unsupported")
	}
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func writeProblem(w http.ResponseWriter, err error) {
	status, code, detail := http.StatusInternalServerError, "internal", "the request could not be completed"
	switch {
	case errors.Is(err, ErrInvalid):
		status, code, detail = http.StatusBadRequest, "invalid_argument", "the request is invalid"
	case errors.Is(err, ErrDenied):
		status, code, detail = http.StatusForbidden, "denied", "the request is not authorized"
	case errors.Is(err, ErrNotFound):
		status, code, detail = http.StatusNotFound, "not_found", "the requested resource was not found"
	case errors.Is(err, ErrConflict):
		status, code, detail = http.StatusConflict, "conflict", "the authoritative state changed; refresh and retry"
	case errors.Is(err, ErrInvariant):
		status, code, detail = http.StatusUnprocessableEntity, "invariant", "a control-plane invariant rejected the request"
	case errors.Is(err, ErrUnavailable):
		status, code, detail = http.StatusServiceUnavailable, "unavailable", "an identity or persistence dependency is unavailable"
	}
	problem := map[string]any{"type": "about:blank", "title": code, "status": status, "code": code, "detail": detail}
	var operationError *OpError
	if errors.As(err, &operationError) && operationError.Field != "" {
		problem["field"] = operationError.Field
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
