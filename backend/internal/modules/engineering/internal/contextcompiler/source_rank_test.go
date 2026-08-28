package contextcompiler

import (
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/engineering/internal/scope"
)

func TestBetterSourcePrefersFreshPinnedObservationBeforeStaleAuthority(t *testing.T) {
	now := time.Date(2026, 8, 29, 4, 0, 0, 0, time.UTC)
	staleAuthoritative := scope.SourceRef{
		BindingID: "authoritative-old", EntityID: "service-1", SourceType: "github", Locator: "github://acme/service", Revision: "old",
		Authority: "authoritative", ObservedAt: now.Add(-72 * time.Hour), Stale: true,
	}
	freshObserved := scope.SourceRef{
		BindingID: "observed-fresh", EntityID: "service-1", SourceType: "github", Locator: "github://acme/service", Revision: "new",
		Authority: "observed", ObservedAt: now.Add(-time.Hour), Stale: false,
	}
	if !betterSource(freshObserved, staleAuthoritative) {
		t.Fatal("fresh pinned observed source must outrank stale authoritative source")
	}
	if betterSource(staleAuthoritative, freshObserved) {
		t.Fatal("stale authoritative source must not outrank fresh pinned observed source")
	}
}
