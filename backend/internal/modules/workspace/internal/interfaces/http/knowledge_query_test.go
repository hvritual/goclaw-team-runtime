package http

import (
	"errors"
	"net/url"
	"testing"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

func TestStrictKnowledgeIntegerDistinguishesAbsentFromExplicitEmpty(t *testing.T) {
	values := url.Values{}
	if value, err := strictKnowledgeInteger(values, "revision"); err != nil || value != 0 {
		t.Fatalf("absent revision = %d, %v", value, err)
	}
	for _, raw := range []string{"", " ", "0", "-1", "abc"} {
		values.Set("revision", raw)
		if _, err := strictKnowledgeInteger(values, "revision"); !errors.Is(err, contract.ErrInvalidKnowledgeQuery) {
			t.Fatalf("revision %q error = %v", raw, err)
		}
	}
}
