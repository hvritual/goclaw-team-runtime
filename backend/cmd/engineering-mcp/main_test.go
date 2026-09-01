package main

import (
	"io"
	"testing"
)

func TestParseConfigRequiresFixedUserIdentity(t *testing.T) {
	if _, err := parseConfig(nil, io.Discard); err == nil {
		t.Fatal("expected missing --user-id to fail")
	}
	cfg, err := parseConfig([]string{"--sqlite-path", ":memory:", "--user-id", " user-1 "}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SQLitePath != ":memory:" || cfg.UserID != "user-1" {
		t.Fatalf("config = %#v", cfg)
	}
}

func TestOpenCanonicalSQLite(t *testing.T) {
	db, err := openCanonicalSQLite(t.Context(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.PingContext(t.Context()); err != nil {
		t.Fatal(err)
	}
}
