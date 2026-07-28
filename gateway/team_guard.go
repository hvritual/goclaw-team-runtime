package gateway

import (
	"fmt"
	"strings"

	"github.com/smallnest/goclaw/teamcontrol"
)

// authorizeTeamMethod is the deny-by-default boundary for JSON-RPC methods in
// multi-user mode. Domain handlers in the explicitly safe families perform
// their own resource authorization. Legacy process-global methods are blocked,
// while older extension services are bound here to their project resources.
func (h *Handler) authorizeTeamMethod(
	sessionID string,
	method string,
	params map[string]interface{},
) error {
	if h.teamSvc == nil {
		return nil
	}
	if params == nil {
		params = map[string]interface{}{}
	}
	switch method {
	case "health":
		return nil
	case "chat", "agent", "agent.wait", "chat.history":
		return h.authorizeChat(sessionID, params)
	case "docs.summary", "components.summary":
		return nil // the registered TeamControl handlers authorize project_id
	case "config.get", "config.set", "logs.get",
		"channels.list", "channels.status", "send", "chat.send",
		"sessions.list", "sessions.get", "sessions.clear":
		return fmt.Errorf(
			"%w: process-global method %q is disabled in team mode",
			teamcontrol.ErrForbidden,
			method,
		)
	}
	for _, prefix := range []string{
		"team.",
		"project.",
		"repository.",
		"issue.",
		"work.",
		"assignment.",
		"artifact.",
		"correlation.",
		"document.",
		"component.",
		"policy.",
		"runner.",
		"dev.",
		"control.",
	} {
		if strings.HasPrefix(method, prefix) {
			return nil // these handlers contain resource-specific authorization
		}
	}
	switch {
	case strings.HasPrefix(method, "browser."),
		strings.HasPrefix(method, "cron."):
		return fmt.Errorf(
			"%w: process-global method %q is disabled in team mode",
			teamcontrol.ErrForbidden,
			method,
		)
	case strings.HasPrefix(method, "harness."),
		strings.HasPrefix(method, "knowledge."):
		return h.authorizeHarnessMethod(sessionID, method, params)
	case strings.HasPrefix(method, "memory."):
		return h.authorizeMemoryMethod(sessionID, method, params)
	case strings.HasPrefix(method, "ouroboros."):
		return h.authorizeOuroborosMethod(sessionID, method, params)
	default:
		return fmt.Errorf(
			"%w: method %q has no team-mode authorization policy",
			teamcontrol.ErrForbidden,
			method,
		)
	}
}

func (h *Handler) authorizeHarnessMethod(
	sessionID string,
	method string,
	params map[string]interface{},
) error {
	if h.harnessSvc == nil {
		return fmt.Errorf("harness service is not enabled")
	}
	projectID := h.harnessSvc.ProjectID()
	if requested := strings.TrimSpace(stringParam(params["project_id"])); requested != "" &&
		requested != projectID {
		return fmt.Errorf(
			"%w: harness project %q does not match requested project %q",
			teamcontrol.ErrForbidden,
			projectID,
			requested,
		)
	}
	action := teamcontrol.ActionArtifactRead
	switch method {
	case "harness.feedback",
		"harness.experiment.create",
		"harness.experiment.validate",
		"harness.experiment.approve",
		"harness.experiment.reject",
		"harness.experiment.promote",
		"harness.rollback":
		action = teamcontrol.ActionArtifactWrite
	case "knowledge.proposal.create",
		"knowledge.proposal.approve",
		"knowledge.proposal.reject":
		action = teamcontrol.ActionDocumentWrite
	case "knowledge.proposals", "knowledge.proposal.get":
		action = teamcontrol.ActionDocumentRead
	}
	_, err := h.authorizeProject(sessionID, projectID, action)
	return err
}

func (h *Handler) authorizeMemoryMethod(
	sessionID string,
	method string,
	params map[string]interface{},
) error {
	if h.catalogSvc == nil {
		return fmt.Errorf("memory catalog service is not enabled")
	}
	switch method {
	case "memory.catalog.status",
		"memory.catalog.list",
		"memory.catalog.search",
		"memory.authority.list",
		"memory.authority.resolve":
		return h.authorizeRequestedProject(
			sessionID,
			params,
			teamcontrol.ActionDocumentRead,
		)
	case "memory.catalog.candidate.create",
		"memory.authority.upsert":
		return h.authorizeRequestedProject(
			sessionID,
			params,
			teamcontrol.ActionDocumentWrite,
		)
	case "memory.catalog.get":
		return h.authorizeMemoryRecord(
			sessionID,
			params,
			teamcontrol.ActionDocumentRead,
		)
	case "memory.catalog.candidate.approve",
		"memory.catalog.candidate.reject",
		"memory.catalog.withdraw",
		"memory.catalog.review.renew":
		return h.authorizeMemoryRecord(
			sessionID,
			params,
			teamcontrol.ActionDocumentWrite,
		)
	case "memory.catalog.usage":
		return h.authorizeMemoryRecord(
			sessionID,
			params,
			teamcontrol.ActionDocumentRead,
		)
	case "memory.authority.merge":
		source, err := h.catalogSvc.GetAuthority(stringParam(params["source_id"]))
		if err != nil {
			return err
		}
		target, err := h.catalogSvc.GetAuthority(stringParam(params["target_id"]))
		if err != nil {
			return err
		}
		if source.ProjectID != target.ProjectID {
			return fmt.Errorf(
				"%w: authorities belong to different projects",
				teamcontrol.ErrForbidden,
			)
		}
		return h.authorizeResolvedProject(
			sessionID,
			params,
			source.ProjectID,
			teamcontrol.ActionDocumentWrite,
		)
	default:
		return fmt.Errorf(
			"%w: method %q has no memory authorization policy",
			teamcontrol.ErrForbidden,
			method,
		)
	}
}

func (h *Handler) authorizeMemoryRecord(
	sessionID string,
	params map[string]interface{},
	action teamcontrol.Action,
) error {
	record, err := h.catalogSvc.Get(stringParam(params["id"]))
	if err != nil {
		return err
	}
	return h.authorizeResolvedProject(
		sessionID,
		params,
		record.ProjectID,
		action,
	)
}

func (h *Handler) authorizeOuroborosMethod(
	sessionID string,
	method string,
	params map[string]interface{},
) error {
	if h.ouroSvc == nil {
		return fmt.Errorf("ouroboros service is not enabled")
	}
	switch method {
	case "ouroboros.sessions", "ouroboros.reference_class":
		return h.authorizeRequestedProject(
			sessionID,
			params,
			teamcontrol.ActionProjectRead,
		)
	case "ouroboros.session.start":
		return h.authorizeRequestedProject(
			sessionID,
			params,
			teamcontrol.ActionWorkItemWrite,
		)
	case "ouroboros.seed.get":
		seed, err := h.ouroSvc.GetSeed(stringParam(params["hash"]))
		if err != nil {
			return err
		}
		session, err := h.ouroSvc.GetSession(seed.SessionID)
		if err != nil {
			return err
		}
		return h.authorizeResolvedProject(
			sessionID,
			params,
			session.ProjectID,
			teamcontrol.ActionProjectRead,
		)
	}
	id := stringParam(params["id"])
	session, err := h.ouroSvc.GetSession(id)
	if err != nil {
		return err
	}
	action := teamcontrol.ActionWorkItemWrite
	switch method {
	case "ouroboros.session.get", "ouroboros.session.events":
		action = teamcontrol.ActionProjectRead
	case "ouroboros.seed.approve",
		"ouroboros.seed.reject",
		"ouroboros.evaluation.resolve",
		"ouroboros.evolution.approve",
		"ouroboros.evolution.reject",
		"ouroboros.readiness.resolve",
		"ouroboros.conflict.resolve",
		"ouroboros.outcome.record",
		"ouroboros.kill.trigger",
		"ouroboros.session.cancel":
		action = teamcontrol.ActionArtifactWrite
	case "ouroboros.session.answer",
		"ouroboros.session.reassess",
		"ouroboros.session.crystallize",
		"ouroboros.session.compile",
		"ouroboros.session.evaluate",
		"ouroboros.session.evolve":
	default:
		return fmt.Errorf(
			"%w: method %q has no ouroboros authorization policy",
			teamcontrol.ErrForbidden,
			method,
		)
	}
	if err := h.authorizeResolvedProject(
		sessionID,
		params,
		session.ProjectID,
		action,
	); err != nil {
		return err
	}
	if method == "ouroboros.session.evaluate" && h.devSvc != nil {
		task, err := h.devSvc.GetTask(stringParam(params["task_id"]))
		if err != nil {
			return err
		}
		if task.ProjectID != session.ProjectID {
			return fmt.Errorf(
				"%w: development task and Ouroboros session belong to different projects",
				teamcontrol.ErrForbidden,
			)
		}
	}
	return nil
}

func (h *Handler) authorizeRequestedProject(
	sessionID string,
	params map[string]interface{},
	action teamcontrol.Action,
) error {
	_, err := h.authorizeProject(
		sessionID,
		stringParam(params["project_id"]),
		action,
	)
	return err
}

func (h *Handler) authorizeResolvedProject(
	sessionID string,
	params map[string]interface{},
	resolvedProjectID string,
	action teamcontrol.Action,
) error {
	resolvedProjectID = strings.TrimSpace(resolvedProjectID)
	requested := strings.TrimSpace(stringParam(params["project_id"]))
	if resolvedProjectID == "*" {
		if requested == "" || requested == "*" {
			return fmt.Errorf(
				"%w: project_id is required for shared resources",
				teamcontrol.ErrForbidden,
			)
		}
		resolvedProjectID = requested
	} else if requested != "" && requested != resolvedProjectID {
		return fmt.Errorf(
			"%w: resource project %q does not match requested project %q",
			teamcontrol.ErrForbidden,
			resolvedProjectID,
			requested,
		)
	}
	_, err := h.authorizeProject(sessionID, resolvedProjectID, action)
	return err
}
