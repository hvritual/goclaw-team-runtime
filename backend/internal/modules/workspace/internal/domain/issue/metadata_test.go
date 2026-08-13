package issue

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateMetadataKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{name: "simple", key: "build_id"},
		{name: "qualified", key: "integration.release-id"},
		{name: "digit first", key: "1bad", wantErr: true},
		{name: "space", key: "bad key", wantErr: true},
		{name: "too long", key: "a" + strings.Repeat("b", 64), wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateMetadataKey(test.key)
			if (err != nil) != test.wantErr {
				t.Fatalf("ValidateMetadataKey(%q) error = %v, wantErr %v", test.key, err, test.wantErr)
			}
		})
	}
}

func TestParseMetadataValueJSONPreservesPrimitiveTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		raw     string
		want    any
		wantErr bool
	}{
		{name: "string", raw: `"v1"`, want: "v1"},
		{name: "integer", raw: `42`, want: json.Number("42")},
		{name: "decimal", raw: `1.25`, want: json.Number("1.25")},
		{name: "boolean", raw: `true`, want: true},
		{name: "null", raw: `null`, wantErr: true},
		{name: "object", raw: `{}`, wantErr: true},
		{name: "array", raw: `[]`, wantErr: true},
		{name: "invalid", raw: `{`, wantErr: true},
		{name: "malformed trailing token", raw: `true{`, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseMetadataValueJSON(test.raw)
			if (err != nil) != test.wantErr {
				t.Fatalf("ParseMetadataValueJSON(%q) error = %v, wantErr %v", test.raw, err, test.wantErr)
			}
			if !test.wantErr && got != test.want {
				t.Fatalf("ParseMetadataValueJSON(%q) = %#v, want %#v", test.raw, got, test.want)
			}
		})
	}
}

func TestMetadataBagPutDeleteAndDefensiveCopy(t *testing.T) {
	t.Parallel()
	bag := NewMetadataBag(map[string]any{"source": "legacy"})
	put, err := bag.Put("attempt", json.Number("3"))
	if err != nil {
		t.Fatal(err)
	}
	put["source"] = "mutated"
	if got := bag.Snapshot()["source"]; got != "legacy" {
		t.Fatalf("returned map mutated bag: %v", got)
	}
	deleted, err := bag.Delete("absent")
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted) != 2 {
		t.Fatalf("delete absent changed bag: %#v", deleted)
	}
}

func TestMetadataBagLimits(t *testing.T) {
	t.Parallel()
	values := make(map[string]any, 50)
	for index := 0; index < 50; index++ {
		values["key_"+strings.Repeat("x", index/10)+string(rune('a'+index%10))] = true
	}
	bag := NewMetadataBag(values)
	if _, err := bag.Put("key_a", false); err != nil {
		t.Fatalf("replace at key limit: %v", err)
	}
	if _, err := bag.Put("overflow", true); err == nil {
		t.Fatal("expected 51st key to fail")
	}
	large := NewMetadataBag(nil)
	if _, err := large.Put("payload", strings.Repeat("x", 8192)); err == nil {
		t.Fatal("expected compact JSON above 8 KiB to fail")
	}
}
