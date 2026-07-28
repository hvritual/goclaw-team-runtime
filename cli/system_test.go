package cli

import (
	"testing"

	"github.com/smallnest/goclaw/config"
)

func TestGatewayRPCURL(t *testing.T) {
	cfg := &config.Config{}
	cfg.Gateway.Host = "127.0.0.1"
	cfg.Gateway.Port = 8080

	t.Setenv("GOCLAW_GATEWAY_HTTP_URL", "")
	value, err := gatewayRPCURL(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if value != "http://127.0.0.1:8080/rpc" {
		t.Fatalf("local URL = %q", value)
	}

	t.Setenv("GOCLAW_GATEWAY_HTTP_URL", "https://goclaw.example.com/control/")
	value, err = gatewayRPCURL(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if value != "https://goclaw.example.com/control/rpc" {
		t.Fatalf("override URL = %q", value)
	}

	for _, invalid := range []string{
		"file:///tmp/socket",
		"https://user:password@goclaw.example.com",
		"https://goclaw.example.com?token=secret",
		"http://goclaw.example.com",
		"http://192.0.2.10:8080",
	} {
		t.Setenv("GOCLAW_GATEWAY_HTTP_URL", invalid)
		if _, err := gatewayRPCURL(cfg); err == nil {
			t.Fatalf("expected invalid override %q to fail", invalid)
		}
	}

	for _, allowed := range []string{
		"http://localhost:8080",
		"http://127.0.0.1:8080",
		"http://[::1]:8080",
		"https://goclaw.example.com",
	} {
		t.Setenv("GOCLAW_GATEWAY_HTTP_URL", allowed)
		if _, err := gatewayRPCURL(cfg); err != nil {
			t.Fatalf("expected override %q to be allowed: %v", allowed, err)
		}
	}

	t.Setenv("GOCLAW_GATEWAY_HTTP_URL", "")
	cfg.Gateway.Host = "0.0.0.0"
	value, err = gatewayRPCURL(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if value != "http://127.0.0.1:8080/rpc" {
		t.Fatalf("wildcard listener URL = %q", value)
	}

	cfg.Gateway.Host = "192.0.2.10"
	if _, err := gatewayRPCURL(cfg); err == nil {
		t.Fatal("expected configured non-loopback cleartext Gateway to fail")
	}
}
