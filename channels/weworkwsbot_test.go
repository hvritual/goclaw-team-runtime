package channels

import (
	"testing"

	"github.com/smallnest/goclaw/bus"
	"github.com/smallnest/goclaw/config"
)

func TestNewWeWorkWsBotChannelRequiresCredentials(t *testing.T) {
	tests := []struct {
		name   string
		config config.WeWorkWsBotChannelConfig
	}{
		{
			name: "missing bot ID",
			config: config.WeWorkWsBotChannelConfig{
				SecretID: "test-secret-id",
			},
		},
		{
			name: "missing secret ID",
			config: config.WeWorkWsBotChannelConfig{
				BotID: "test-bot-id",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel, err := NewWeWorkWsBotChannel(tt.config, nil)
			if err == nil {
				t.Fatalf("constructor returned channel %v without required credentials", channel)
			}
			if channel != nil {
				t.Fatal("constructor returned a channel with invalid configuration")
			}
		})
	}
}

func TestNewWeWorkWsBotChannelInitializesDefaults(t *testing.T) {
	messageBus := bus.NewMessageBus(1)
	t.Cleanup(func() {
		if err := messageBus.Close(); err != nil {
			t.Errorf("close message bus: %v", err)
		}
	})

	channel, err := NewWeWorkWsBotChannel(config.WeWorkWsBotChannelConfig{
		Enabled:  true,
		BotID:    "test-bot-id",
		SecretID: "test-secret-id",
	}, messageBus)
	if err != nil {
		t.Fatalf("constructor returned error: %v", err)
	}
	if channel == nil || channel.BaseChannelImpl == nil {
		t.Fatal("constructor returned an incomplete channel")
	}
	if got := channel.Name(); got != "wework_wsbot" {
		t.Errorf("name = %q, want wework_wsbot", got)
	}
	if got := channel.AccountID(); got != "test-bot-id" {
		t.Errorf("account ID = %q, want test-bot-id", got)
	}
	if channel.BaseChannelImpl.bus != messageBus {
		t.Error("constructor did not retain the provided message bus")
	}
	if channel.config.URL == "" {
		t.Error("default URL is empty")
	}
	if got := channel.config.ReconnectDelay; got != 3 {
		t.Errorf("reconnect delay = %d, want 3", got)
	}
	if got := channel.config.Heartbeat; got != 30 {
		t.Errorf("heartbeat = %d, want 30", got)
	}
	if channel.stopChan == nil {
		t.Error("stop channel is nil")
	}
	if channel.waitResponseMsg == nil {
		t.Error("response map is nil")
	}
	if channel.conn != nil || channel.ctx != nil || channel.cancel != nil {
		t.Error("constructor initialized runtime connection state")
	}
	if channel.connected {
		t.Error("new channel is unexpectedly connected")
	}
}
