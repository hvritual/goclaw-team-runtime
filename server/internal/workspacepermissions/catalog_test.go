package workspacepermissions

import "testing"

func TestFixedCatalogCoversSixDomainsAndFixedRoles(t *testing.T) {
	catalog := FixedCatalog()

	if len(catalog.Roles) != 3 {
		t.Fatalf("got %d roles, want 3", len(catalog.Roles))
	}
	for index, want := range []RoleKey{RoleOwner, RoleAdmin, RoleMember} {
		if got := catalog.Roles[index].Key; got != want {
			t.Fatalf("role %d = %q, want %q", index, got, want)
		}
	}

	domains := make(map[Domain]bool)
	for _, capability := range catalog.Capabilities {
		domains[capability.Domain] = true
	}
	for _, domain := range []Domain{
		DomainWorkspace,
		DomainMember,
		DomainProject,
		DomainIssue,
		DomainTask,
		DomainSkill,
	} {
		if !domains[domain] {
			t.Errorf("permission catalog does not include the %s domain", domain)
		}
	}
	if len(domains) != 6 {
		t.Fatalf("permission catalog has unexpected domains: %#v", domains)
	}
}

func TestFixedCatalogReturnsIndependentSlices(t *testing.T) {
	first := FixedCatalog()
	first.Roles[0].Key = RoleKey("changed")
	first.Capabilities[0].Key = "changed"

	second := FixedCatalog()
	if second.Roles[0].Key != "owner" {
		t.Fatalf("roles leaked mutation: %#v", second.Roles)
	}
	if second.Capabilities[0].Key != "workspace.view" {
		t.Fatalf("capabilities leaked mutation: %#v", second.Capabilities[0])
	}
}
