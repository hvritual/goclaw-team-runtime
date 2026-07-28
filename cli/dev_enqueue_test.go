package cli

import (
	"reflect"
	"testing"
)

func TestDevelopmentEnqueueParamsUsesGatewayCapabilityContract(t *testing.T) {
	previousPriority := devQueuePriority
	previousCapabilities := devQueueCapabilities
	previousMaxAttempts := devQueueMaxAttempts
	t.Cleanup(func() {
		devQueuePriority = previousPriority
		devQueueCapabilities = previousCapabilities
		devQueueMaxAttempts = previousMaxAttempts
	})

	devQueuePriority = 12
	devQueueCapabilities = []string{"codex", "go"}
	devQueueMaxAttempts = 4
	params := developmentEnqueueParams("task-alpha")

	if _, legacy := params["required_capabilities"]; legacy {
		t.Fatal("CLI emitted unsupported required_capabilities field")
	}
	if _, clientKey := params["idempotency_key"]; clientKey {
		t.Fatal("CLI emitted a client-controlled idempotency key")
	}
	if got := params["capabilities"]; !reflect.DeepEqual(got, []string{"codex", "go"}) {
		t.Fatalf("capabilities = %#v", got)
	}
	if params["task_id"] != "task-alpha" ||
		params["priority"] != 12 ||
		params["max_attempts"] != 4 {
		t.Fatalf("unexpected enqueue params: %#v", params)
	}
}
