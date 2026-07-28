package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	neturl "net/url"
	"os"
	"strings"
	"time"

	"github.com/smallnest/goclaw/bus"
	"github.com/smallnest/goclaw/config"
	"github.com/smallnest/goclaw/gateway"
	"github.com/spf13/cobra"
)

var systemCmd = &cobra.Command{
	Use:   "system",
	Short: "System control",
}

var systemEventCmd = &cobra.Command{
	Use:   "event",
	Short: "Enqueue a system event",
	Run:   runSystemEvent,
}

var systemHeartbeatCmd = &cobra.Command{
	Use:   "heartbeat",
	Short: "Control heartbeat settings",
	Args:  cobra.ExactArgs(1),
	Run:   runSystemHeartbeat,
}

var systemPresenceCmd = &cobra.Command{
	Use:   "presence",
	Short: "List system presence entries",
	Run:   runSystemPresence,
}

// System flags
var (
	systemEventText string
	systemEventMode string
)

func init() {
	// Register system commands
	rootCmd.AddCommand(systemCmd)
	systemCmd.AddCommand(systemEventCmd)
	systemCmd.AddCommand(systemHeartbeatCmd)
	systemCmd.AddCommand(systemPresenceCmd)

	// system event flags
	systemEventCmd.Flags().StringVar(&systemEventText, "text", "", "Event text (required)")
	systemEventCmd.Flags().StringVar(&systemEventMode, "mode", "normal", "Event mode")
	_ = systemEventCmd.MarkFlagRequired("text")
}

// runSystemEvent handles the system event command
func runSystemEvent(cmd *cobra.Command, args []string) {
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create system event message
	msg := &bus.InboundMessage{
		Channel:  "system",
		SenderID: "cli",
		ChatID:   "system",
		Content:  systemEventText,
		Metadata: map[string]interface{}{
			"event_type": "system_event",
			"mode":       systemEventMode,
			"timestamp":  time.Now().Unix(),
		},
		Timestamp: time.Now(),
	}

	// Publish to message bus via gateway
	if err := publishViaGateway(cfg, msg); err != nil {
		fmt.Fprintf(os.Stderr, "Error publishing system event: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("System event enqueued: %s (mode: %s)\n", systemEventText, systemEventMode)
}

// runSystemHeartbeat handles the system heartbeat command
func runSystemHeartbeat(cmd *cobra.Command, args []string) {
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	action := args[0]

	switch action {
	case "last":
		handleHeartbeatLast(cfg)
	case "enable":
		handleHeartbeatEnable(cfg)
	case "disable":
		handleHeartbeatDisable(cfg)
	default:
		fmt.Fprintf(os.Stderr, "Unknown heartbeat action: %s. Valid actions: last, enable, disable\n", action)
		os.Exit(1)
	}
}

// handleHeartbeatLast handles getting the last heartbeat
func handleHeartbeatLast(cfg *config.Config) {
	result, err := callGatewayRPC(cfg, "heartbeat.last", map[string]interface{}{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting last heartbeat: %v\n", err)
		os.Exit(1)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		fmt.Fprintln(os.Stderr, "Invalid response from gateway")
		os.Exit(1)
	}

	fmt.Println("Last Heartbeat:")
	if timestamp, ok := data["timestamp"].(float64); ok {
		fmt.Printf("  Timestamp: %s\n", time.Unix(int64(timestamp), 0).Format(time.RFC3339))
	}
	if status, ok := data["status"].(string); ok {
		fmt.Printf("  Status: %s\n", status)
	}
}

// handleHeartbeatEnable handles enabling heartbeat
func handleHeartbeatEnable(cfg *config.Config) {
	result, err := callGatewayRPC(cfg, "heartbeat.enable", map[string]interface{}{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error enabling heartbeat: %v\n", err)
		os.Exit(1)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		fmt.Fprintln(os.Stderr, "Invalid response from gateway")
		os.Exit(1)
	}

	if status, ok := data["status"].(string); ok {
		fmt.Printf("Heartbeat enabled: %s\n", status)
	} else {
		fmt.Println("Heartbeat enabled")
	}
}

// handleHeartbeatDisable handles disabling heartbeat
func handleHeartbeatDisable(cfg *config.Config) {
	result, err := callGatewayRPC(cfg, "heartbeat.disable", map[string]interface{}{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error disabling heartbeat: %v\n", err)
		os.Exit(1)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		fmt.Fprintln(os.Stderr, "Invalid response from gateway")
		os.Exit(1)
	}

	if status, ok := data["status"].(string); ok {
		fmt.Printf("Heartbeat disabled: %s\n", status)
	} else {
		fmt.Println("Heartbeat disabled")
	}
}

// runSystemPresence handles the system presence command
func runSystemPresence(cmd *cobra.Command, args []string) {
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	result, err := callGatewayRPC(cfg, "presence.list", map[string]interface{}{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing presence entries: %v\n", err)
		os.Exit(1)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		fmt.Fprintln(os.Stderr, "Invalid response from gateway")
		os.Exit(1)
	}

	entries, ok := data["entries"].([]interface{})
	if !ok {
		fmt.Println("No presence entries found")
		return
	}

	if len(entries) == 0 {
		fmt.Println("No presence entries found")
		return
	}

	fmt.Println("System Presence Entries:")

	for i, entry := range entries {
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}

		fmt.Printf("\n  %d. ", i+1)

		if sessionID, ok := entryMap["session_id"].(string); ok {
			fmt.Printf("Session: %s\n", sessionID)
		}

		if channel, ok := entryMap["channel"].(string); ok {
			fmt.Printf("     Channel: %s\n", channel)
		}

		if timestamp, ok := entryMap["last_seen"].(float64); ok {
			fmt.Printf("     Last Seen: %s\n", time.Unix(int64(timestamp), 0).Format(time.RFC3339))
		}

		if status, ok := entryMap["status"].(string); ok {
			fmt.Printf("     Status: %s\n", status)
		}
	}
}

// callGatewayRPC calls a gateway RPC method
func callGatewayRPC(cfg *config.Config, method string, params map[string]interface{}) (interface{}, error) {
	return callGatewayRPCContext(context.Background(), cfg, method, params)
}

// callGatewayRPCContext keeps cancellation connected to long-running callers
// such as workstation runners. Credentials must never be sent over cleartext
// HTTP to a non-loopback host.
func callGatewayRPCContext(
	ctx context.Context,
	cfg *config.Config,
	method string,
	params map[string]interface{},
) (interface{}, error) {
	url, err := gatewayRPCURL(cfg)
	if err != nil {
		return nil, err
	}

	// Create JSON-RPC request
	rpcRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      fmt.Sprintf("cli-%d", time.Now().UnixNano()),
		"method":  method,
		"params":  params,
	}

	requestBody, err := json.Marshal(rpcRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send HTTP request
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(requestBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	gatewayToken := strings.TrimSpace(os.Getenv("GOCLAW_GATEWAY_TOKEN"))
	if gatewayToken == "" {
		gatewayToken = strings.TrimSpace(cfg.Gateway.WebSocket.AuthToken)
	}
	if gatewayToken != "" {
		req.Header.Set("Authorization", "Bearer "+gatewayToken)
	}
	if userToken := strings.TrimSpace(os.Getenv("GOCLAW_USER_TOKEN")); userToken != "" {
		req.Header.Set("X-GoClaw-User-Token", userToken)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w (is the gateway running?)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gateway returned status %d", resp.StatusCode)
	}

	// Parse response
	var rpcResponse gateway.JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if rpcResponse.Error != nil {
		return nil, fmt.Errorf("RPC error: %s", rpcResponse.Error.Message)
	}

	return rpcResponse.Result, nil
}

func gatewayRPCURL(cfg *config.Config) (string, error) {
	if override := strings.TrimSpace(os.Getenv("GOCLAW_GATEWAY_HTTP_URL")); override != "" {
		parsed, err := neturl.Parse(override)
		if err != nil {
			return "", fmt.Errorf("parse GOCLAW_GATEWAY_HTTP_URL: %w", err)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return "", errors.New("GOCLAW_GATEWAY_HTTP_URL must use http or https")
		}
		if parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return "", errors.New(
				"GOCLAW_GATEWAY_HTTP_URL must be absolute and omit credentials, query, and fragment",
			)
		}
		if parsed.Scheme == "http" && !isLoopbackHost(parsed.Hostname()) {
			return "", errors.New(
				"GOCLAW_GATEWAY_HTTP_URL must use https for non-loopback hosts",
			)
		}
		parsed.Path = strings.TrimRight(parsed.Path, "/") + "/rpc"
		return parsed.String(), nil
	}

	// Build the local Gateway URL from application configuration.
	host := cfg.Gateway.Host
	if host == "" {
		host = "localhost"
	}
	if host == "0.0.0.0" || host == "::" || host == "[::]" {
		host = "127.0.0.1"
	}
	if !isLoopbackHost(strings.Trim(host, "[]")) {
		return "", errors.New(
			"configured Gateway host is non-loopback; set GOCLAW_GATEWAY_HTTP_URL to an https URL",
		)
	}
	port := cfg.Gateway.Port
	if port == 0 {
		// Use WebSocket port if HTTP port is not configured
		port = config.GetGatewayHTTPPort(cfg)
	}

	return fmt.Sprintf("http://%s:%d/rpc", host, port), nil
}

func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(strings.Trim(host, "[]"))
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// publishViaGateway publishes a message via the gateway RPC
func publishViaGateway(cfg *config.Config, msg *bus.InboundMessage) error {
	params := map[string]interface{}{
		"channel":   msg.Channel,
		"sender_id": msg.SenderID,
		"chat_id":   msg.ChatID,
		"content":   msg.Content,
		"metadata":  msg.Metadata,
	}

	_, err := callGatewayRPC(cfg, "agent.publish_inbound", params)
	return err
}

// _getGatewayStatus Helper function to get gateway status (未使用，保留供将来使用)
// nolint:unused
func _getGatewayStatus(cfg *config.Config) (map[string]interface{}, error) {
	result, err := callGatewayRPC(cfg, "health", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response")
	}

	return data, nil
}

// _listGatewaySessions Helper function to list gateway sessions (未使用，保留供将来使用)
// nolint:unused
func _listGatewaySessions(cfg *config.Config) ([]map[string]interface{}, error) {
	result, err := callGatewayRPC(cfg, "sessions.list", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	// Parse sessions array
	if sessions, ok := result.([]interface{}); ok {
		output := make([]map[string]interface{}, 0, len(sessions))
		for _, sess := range sessions {
			if sessMap, ok := sess.(map[string]interface{}); ok {
				output = append(output, sessMap)
			}
		}
		return output, nil
	}

	return nil, fmt.Errorf("invalid response format")
}

// _getChannelStatus Helper function to get channel status (未使用，保留供将来使用)
// nolint:unused
func _getChannelStatus(cfg *config.Config, channelName string) (map[string]interface{}, error) {
	result, err := callGatewayRPC(cfg, "channels.status", map[string]interface{}{
		"channel": channelName,
	})
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response")
	}

	return data, nil
}
