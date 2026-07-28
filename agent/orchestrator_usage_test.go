package agent

import (
	"testing"

	"github.com/smallnest/goclaw/providers"
)

func TestConvertFromProviderResponsePreservesUsage(t *testing.T) {
	message := convertFromProviderResponse(&providers.Response{
		Content:      "done",
		FinishReason: "stop",
		Usage: providers.Usage{
			PromptTokens:     11,
			CompletionTokens: 7,
			TotalTokens:      18,
		},
	})
	usage, ok := message.Metadata["usage"].(map[string]any)
	if !ok {
		t.Fatalf("usage metadata missing: %+v", message.Metadata)
	}
	if numericInt(usage["input"]) != 11 || numericInt(usage["output"]) != 7 || numericInt(usage["total"]) != 18 {
		t.Fatalf("unexpected usage metadata: %+v", usage)
	}
}
