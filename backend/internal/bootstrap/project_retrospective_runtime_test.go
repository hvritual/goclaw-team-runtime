package bootstrap

import (
	"testing"

	workspacecontract "github.com/hvritual/workspace/internal/modules/workspace/contract"
)

func TestInstalledRuntimeAdvertisesCompleteProjectRetrospectiveCapability(t *testing.T) {
	installed := installedRuntimeCapabilities{next: runtimeCapabilityProviderStub{}}
	for _, permission := range []string{
		workspacecontract.PermissionRetrospectiveDraft,
		workspacecontract.PermissionRetrospectivePublish,
	} {
		if !installed.RoadmapCapabilityInstalled(permission) {
			t.Fatalf("installed Retrospective permission %q = false", permission)
		}
	}
	if !installed.RoadmapFeatureInstalled("project_retrospectives") {
		t.Fatal("installed project_retrospectives feature = false")
	}
}
