package main

import (
	"io"
	"testing"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name         string
		arguments    []string
		wantHTTP     string
		wantGRPC     string
		wantName     string
		wantSQLite   string
		wantCode     string
		wantMetadata bool
		wantErr      bool
	}{
		{
			name:         "defaults",
			wantHTTP:     "127.0.0.1:8000",
			wantGRPC:     "127.0.0.1:9000",
			wantName:     "hvritual-workspace-backend",
			wantSQLite:   "data/multica-canonical.db",
			wantCode:     "888888",
			wantMetadata: true,
		},
		{
			name: "overrides",
			arguments: []string{
				"-http-addr", "127.0.0.1:18080",
				"-grpc-addr", "127.0.0.1:19090",
				"-name", "test-backend",
				"-sqlite-path", "test.db",
				"-dev-verification-code", "123456",
				"-issue-metadata=false",
			},
			wantHTTP:     "127.0.0.1:18080",
			wantGRPC:     "127.0.0.1:19090",
			wantName:     "test-backend",
			wantSQLite:   "test.db",
			wantCode:     "123456",
			wantMetadata: false,
		},
		{
			name:      "invalid address",
			arguments: []string{"-http-addr", "invalid"},
			wantErr:   true,
		},
		{
			name:      "unexpected argument",
			arguments: []string{"extra"},
			wantErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := parseConfig(test.arguments, io.Discard)
			if (err != nil) != test.wantErr {
				t.Fatalf("parseConfig() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.wantErr {
				return
			}
			if config.HTTPAddress != test.wantHTTP || config.GRPCAddress != test.wantGRPC || config.Name != test.wantName || config.SQLitePath != test.wantSQLite || config.LocalAuth.VerificationCode != test.wantCode {
				t.Fatalf("parseConfig() = %#v", config)
			}
			if config.IssueMetadataEnabled == nil || *config.IssueMetadataEnabled != test.wantMetadata {
				t.Fatalf("metadata selector = %#v", config.IssueMetadataEnabled)
			}
		})
	}
}
