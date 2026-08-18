package realtime

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/gorilla/websocket"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type IdentityResolver func(*http.Request) (contract.WorkspaceHTTPIdentity, error)
type EventAccessResolver func(workspaceID, actorType, actorID, eventType string) bool

type Hub struct {
	identity IdentityResolver
	access   EventAccessResolver
	mu       sync.RWMutex
	clients  map[string]map[*client]struct{}
}

type client struct {
	writer    frameWriter
	identity  contract.WorkspaceHTTPIdentity
	outbound  chan []byte
	done      chan struct{}
	closeOnce sync.Once
}

type frameWriter interface {
	WriteMessage(int, []byte) error
	SetWriteDeadline(time.Time) error
	Close() error
}

const (
	clientQueueCapacity  = 64
	writeTimeout         = 5 * time.Second
	maxInboundFrameBytes = 64 * 1024
)

func NewHub(identity IdentityResolver, access ...EventAccessResolver) *Hub {
	var resolver EventAccessResolver
	if len(access) > 0 {
		resolver = access[0]
	}
	return &Hub{identity: identity, access: resolver, clients: make(map[string]map[*client]struct{})}
}

func (h *Hub) RegisterHTTP(server *kratoshttp.Server) {
	server.Route("/").GET("/ws", h.connect)
}

func (h *Hub) connect(ctx kratoshttp.Context) error {
	request := withWorkspaceSlug(ctx.Request())
	_, cookieErr := request.Cookie("multica_auth")
	hasCookie := cookieErr == nil
	identity := contract.WorkspaceHTTPIdentity{}
	identityErr := errors.New("token first frame required")
	if hasCookie {
		identity, identityErr = h.identity(request)
	}
	if hasCookie && identityErr != nil {
		if errors.Is(identityErr, contract.ErrWorkspaceNotFound) || errors.Is(identityErr, contract.ErrActorOutsideWorkspace) {
			return ctx.JSON(http.StatusNotFound, map[string]string{"error": "workspace not found"})
		}
		return ctx.JSON(http.StatusUnauthorized, map[string]string{"error": "user not authenticated"})
	}
	upgrader := websocket.Upgrader{CheckOrigin: sameOrigin}
	connection, err := upgrader.Upgrade(ctx.Response(), ctx.Request(), nil)
	if err != nil {
		return nil
	}
	connection.SetReadLimit(maxInboundFrameBytes)
	if identityErr != nil {
		identity, err = h.readTokenIdentity(connection, request)
		if err != nil {
			_ = connection.SetWriteDeadline(time.Now().Add(writeTimeout))
			_ = writeFrame(connection, struct {
				Type  string `json:"type"`
				Error string `json:"error"`
			}{Type: "auth_error", Error: "authentication failed"})
			_ = connection.Close()
			return nil
		}
	}
	connected := newClient(connection, clientQueueCapacity)
	connected.identity = identity
	h.add(identity.WorkspaceID, connected)
	go connected.writePump()
	defer func() { h.remove(identity.WorkspaceID, connected); connected.close() }()
	if !hasCookie {
		if !connected.enqueue(mustFrame(struct {
			Type string `json:"type"`
		}{Type: "auth_ack"})) {
			return nil
		}
	}
	for {
		if _, _, err := connection.ReadMessage(); err != nil {
			return nil
		}
	}
}

func (h *Hub) readTokenIdentity(connection *websocket.Conn, request *http.Request) (contract.WorkspaceHTTPIdentity, error) {
	_ = connection.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, data, err := connection.ReadMessage()
	if err != nil {
		return contract.WorkspaceHTTPIdentity{}, err
	}
	if len(data) > 8192 {
		return contract.WorkspaceHTTPIdentity{}, errors.New("auth frame too large")
	}
	var frame struct {
		Type    string `json:"type"`
		Payload struct {
			Token string `json:"token"`
		} `json:"payload"`
	}
	if json.Unmarshal(data, &frame) != nil || frame.Type != "auth" || strings.TrimSpace(frame.Payload.Token) == "" {
		return contract.WorkspaceHTTPIdentity{}, errors.New("invalid auth frame")
	}
	clone := request.Clone(request.Context())
	clone.Header.Set("Authorization", "Bearer "+strings.TrimSpace(frame.Payload.Token))
	identity, err := h.identity(clone)
	_ = connection.SetReadDeadline(time.Time{})
	return identity, err
}

func withWorkspaceSlug(request *http.Request) *http.Request {
	clone := request.Clone(request.Context())
	clone.Header.Set("X-Workspace-Slug", strings.TrimSpace(request.URL.Query().Get("workspace_slug")))
	return clone
}

func sameOrigin(request *http.Request) bool {
	origin := strings.TrimSpace(request.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	if strings.EqualFold(parsed.Host, request.Host) {
		return true
	}
	hostname := strings.ToLower(parsed.Hostname())
	return (hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1") && parsed.Port() == "3000"
}

func (h *Hub) add(workspaceID string, value *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[workspaceID] == nil {
		h.clients[workspaceID] = make(map[*client]struct{})
	}
	h.clients[workspaceID][value] = struct{}{}
}
func (h *Hub) remove(workspaceID string, value *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients[workspaceID], value)
	if len(h.clients[workspaceID]) == 0 {
		delete(h.clients, workspaceID)
	}
}

func (h *Hub) Publish(workspaceID, eventType string, payload any, actorID, actorType string) {
	switch eventType {
	case "issue:created", "issue:updated", "issue:deleted", "issue_metadata:changed",
		"issue_labels:changed", "issue_properties:changed", "issue_attachments:changed",
		"comment:created", "comment:updated", "comment:deleted", "comment:resolved", "comment:unresolved",
		"reaction:added", "reaction:removed", "issue_reaction:added", "issue_reaction:removed",
		"subscriber:added", "subscriber:removed", "activity:created",
		"label:created", "label:updated", "label:deleted", "property:created", "property:updated",
		"knowledge:candidate_updated":
	default:
		return
	}
	h.mu.RLock()
	clients := make([]*client, 0, len(h.clients[workspaceID]))
	for value := range h.clients[workspaceID] {
		clients = append(clients, value)
	}
	h.mu.RUnlock()
	for _, value := range clients {
		projection, visible := h.projectEvent(value, eventType, payload)
		if !visible {
			continue
		}
		message := map[string]any{"type": eventType, "payload": projection}
		if actorID != "" {
			message["actor_id"] = actorID
		}
		if actorType != "" {
			message["actor_type"] = actorType
		}
		frame, err := json.Marshal(message)
		if err != nil {
			continue
		}
		if !value.enqueue(frame) {
			h.remove(workspaceID, value)
			value.close()
		}
	}
}

func (h *Hub) projectEvent(value *client, eventType string, payload any) (any, bool) {
	if eventType != "knowledge:candidate_updated" {
		return payload, true
	}
	if h.access != nil && h.access(value.identity.WorkspaceID, value.identity.ActorType, value.identity.ActorID, eventType) {
		return payload, true
	}
	values, ok := payload.(map[string]any)
	if !ok || values["entry"] == nil {
		return nil, false
	}
	return map[string]any{"entry": values["entry"]}, true
}

func newClient(writer frameWriter, capacity int) *client {
	return &client{writer: writer, outbound: make(chan []byte, capacity), done: make(chan struct{})}
}

func (c *client) enqueue(frame []byte) bool {
	select {
	case <-c.done:
		return false
	default:
	}
	select {
	case c.outbound <- append([]byte(nil), frame...):
		return true
	default:
		return false
	}
}

func (c *client) writePump() {
	for {
		select {
		case <-c.done:
			return
		case frame := <-c.outbound:
			_ = c.writer.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.writer.WriteMessage(websocket.TextMessage, frame); err != nil {
				c.close()
				return
			}
		}
	}
}

func (c *client) close() {
	c.closeOnce.Do(func() { close(c.done); _ = c.writer.Close() })
}

func writeFrame(connection *websocket.Conn, message any) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return connection.WriteMessage(websocket.TextMessage, data)
}

func mustFrame(message any) []byte { data, _ := json.Marshal(message); return data }

func (h *Hub) Close() error {
	h.mu.Lock()
	allClients := make([]*client, 0)
	for _, workspaceClients := range h.clients {
		for value := range workspaceClients {
			allClients = append(allClients, value)
		}
	}
	h.clients = make(map[string]map[*client]struct{})
	h.mu.Unlock()
	for _, value := range allClients {
		value.close()
	}
	return nil
}
