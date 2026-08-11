GO ?= go
BASE_REF ?= codex/multica-six-domain-baseline

.PHONY: fmt-check test test-race vet policy-check generated-clean check run

fmt-check:
	@test -z "$$($(GO)fmt -l $$(find . -name '*.go' -not -path './rpc/pb/*'))" || \
		( $(GO)fmt -l $$(find . -name '*.go' -not -path './rpc/pb/*'); exit 1 )

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

policy-check:
	BASE_REF="$(BASE_REF)" ./ci/check-policy.sh

generated-clean:
	@git diff --exit-code -- rpc/pb rpc/openapi >/dev/null

check: fmt-check policy-check generated-clean vet test

run:
	$(GO) run ./cmd/controlplane
