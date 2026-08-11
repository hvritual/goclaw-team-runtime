package controlplane

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

type IdentityResolver func(*http.Request) (Actor, error)

type HTTPAPI struct {
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

func NewHTTPAPI(kernel *DeliveryKernel, flows *P2Flows, identity IdentityResolver) (*HTTPAPI, error) {
	if kernel == nil || flows == nil || identity == nil {
		return nil, invalid("new HTTP API", "dependency", "kernel, flows, and identity resolver are required")
	}
	return &HTTPAPI{kernel: kernel, flows: flows, identity: identity}, nil
}

func (a *HTTPAPI) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /v1/workspaces/{workspace}/projects/{project}/projection", a.projection)
	mux.HandleFunc("POST /v1/workspaces/{workspace}/projects/{project}/commands", a.command)
	return mux
}

func (a *HTTPAPI) actor(request *http.Request) (Actor, error) {
	actor, err := a.identity(request)
	if err != nil {
		return Actor{}, err
	}
	if actor.WorkspaceID != request.PathValue("workspace") {
		return Actor{}, denied("resolve HTTP identity", "workspace mismatch")
	}
	return actor, nil
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
	status, code := http.StatusInternalServerError, "internal"
	switch {
	case errors.Is(err, ErrInvalid):
		status, code = http.StatusBadRequest, "invalid_argument"
	case errors.Is(err, ErrDenied):
		status, code = http.StatusForbidden, "denied"
	case errors.Is(err, ErrNotFound):
		status, code = http.StatusNotFound, "not_found"
	case errors.Is(err, ErrConflict):
		status, code = http.StatusConflict, "conflict"
	case errors.Is(err, ErrInvariant):
		status, code = http.StatusUnprocessableEntity, "invariant"
	}
	writeJSON(w, status, map[string]any{"type": "about:blank", "title": code, "status": status, "detail": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
