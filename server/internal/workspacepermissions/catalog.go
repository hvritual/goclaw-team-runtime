// Package workspacepermissions defines the fixed workspace role capability
// catalog exposed to enterprise administrators. It is deliberately data-only:
// request authorization remains in the HTTP middleware and handlers.
package workspacepermissions

type AccessLevel string

const (
	AccessAllowed     AccessLevel = "allowed"
	AccessConditional AccessLevel = "conditional"
	AccessDenied      AccessLevel = "denied"
)

type RoleKey string

const (
	RoleOwner  RoleKey = "owner"
	RoleAdmin  RoleKey = "admin"
	RoleMember RoleKey = "member"
)

type Domain string

const (
	DomainWorkspace Domain = "workspace"
	DomainMember    Domain = "member"
	DomainProject   Domain = "project"
	DomainIssue     Domain = "issue"
	DomainTask      Domain = "task"
	DomainSkill     Domain = "skill"
)

type Role struct {
	Key RoleKey `json:"key"`
}

type Access struct {
	Owner  AccessLevel `json:"owner"`
	Admin  AccessLevel `json:"admin"`
	Member AccessLevel `json:"member"`
}

type Capability struct {
	Key    string `json:"key"`
	Domain Domain `json:"domain"`
	Access Access `json:"access"`
}

type Catalog struct {
	Roles        []Role       `json:"roles"`
	Capabilities []Capability `json:"capabilities"`
}

var roles = []Role{
	{Key: RoleOwner},
	{Key: RoleAdmin},
	{Key: RoleMember},
}

var capabilities = []Capability{
	capability("workspace.view", DomainWorkspace, AccessAllowed, AccessAllowed, AccessAllowed),
	capability("workspace.update", DomainWorkspace, AccessAllowed, AccessAllowed, AccessDenied),
	capability("workspace.delete", DomainWorkspace, AccessAllowed, AccessDenied, AccessDenied),

	capability("member.view", DomainMember, AccessAllowed, AccessAllowed, AccessAllowed),
	capability("member.invite", DomainMember, AccessAllowed, AccessAllowed, AccessDenied),
	capability("member.change_role", DomainMember, AccessAllowed, AccessConditional, AccessDenied),
	capability("member.manage_owner", DomainMember, AccessAllowed, AccessDenied, AccessDenied),
	capability("member.remove", DomainMember, AccessAllowed, AccessConditional, AccessDenied),

	capability("project.view", DomainProject, AccessAllowed, AccessAllowed, AccessAllowed),
	capability("project.create", DomainProject, AccessAllowed, AccessAllowed, AccessAllowed),
	capability("project.update", DomainProject, AccessAllowed, AccessAllowed, AccessAllowed),
	capability("project.delete", DomainProject, AccessAllowed, AccessAllowed, AccessDenied),

	capability("issue.view", DomainIssue, AccessAllowed, AccessAllowed, AccessAllowed),
	capability("issue.create", DomainIssue, AccessAllowed, AccessAllowed, AccessAllowed),
	capability("issue.update", DomainIssue, AccessAllowed, AccessAllowed, AccessAllowed),
	capability("issue.delete", DomainIssue, AccessAllowed, AccessAllowed, AccessAllowed),

	capability("task.view", DomainTask, AccessAllowed, AccessAllowed, AccessAllowed),
	capability("task.run", DomainTask, AccessConditional, AccessConditional, AccessConditional),
	capability("task.cancel", DomainTask, AccessConditional, AccessConditional, AccessConditional),

	capability("skill.view", DomainSkill, AccessAllowed, AccessAllowed, AccessAllowed),
	capability("skill.create", DomainSkill, AccessAllowed, AccessAllowed, AccessAllowed),
	capability("skill.update", DomainSkill, AccessAllowed, AccessAllowed, AccessConditional),
	capability("skill.delete", DomainSkill, AccessAllowed, AccessAllowed, AccessConditional),
}

func capability(key string, domain Domain, owner, admin, member AccessLevel) Capability {
	return Capability{
		Key:    key,
		Domain: domain,
		Access: Access{Owner: owner, Admin: admin, Member: member},
	}
}

func FixedCatalog() Catalog {
	return Catalog{
		Roles:        append([]Role(nil), roles...),
		Capabilities: append([]Capability(nil), capabilities...),
	}
}
