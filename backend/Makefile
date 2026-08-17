GO ?= go
BASE_REF ?= codex/multica-six-domain-baseline
RACE_PACKAGES ?= ./...

.PHONY: fmt-check test test-race vet policy-check generated-clean check run

fmt-check:
	$(GO) run ./cmd/backend-check fmt

test:
	$(GO) test ./...

ifeq ($(OS),Windows_NT)
test-race:
	pwsh -NoProfile -File ./ci/test-race.ps1 $(RACE_PACKAGES)
else
test-race:
	$(GO) test -race $(RACE_PACKAGES)
endif

vet:
	$(GO) vet ./...

policy-check:
	$(GO) run ./cmd/backend-check policy "$(BASE_REF)"

generated-clean:
	$(GO) run ./cmd/backend-check generated

check: fmt-check policy-check generated-clean vet test

run:
	$(GO) run ./cmd/controlplane
