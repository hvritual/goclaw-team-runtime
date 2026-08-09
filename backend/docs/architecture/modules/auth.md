# Auth module

- Public seam: internal/modules/auth/contract
- Implementation: internal/modules/auth/internal
- Proto contracts: api/auth/v1/auth.proto, member.proto, and agent.proto

## Member foundation

The first migrated Auth slice implements `ProvisionWorkspaceOwner`,
`ListMembers`, and `UpdateMemberRole` behind an explicit
`NewWithSqliteMemberServices` constructor. The default `auth.New()` composition
still selects generated not-implemented Member and Agent services, so no
running-process behavior changes until a later cutover is authorized.

Auth owns `auth_users`, `auth_members`, and
`auth_workspace_membership_roots`. The root row records the one-time initial
owner decision for a Workspace. The SQLite unit of work begins with
`BEGIN IMMEDIATE`, so concurrent initial-owner calls and last-owner role changes
serialize before reading policy state. Repeating the same completed
Workspace/User provision returns the same Member; a different initializer is
rejected. No foreign key or cascade is used.

Member role policy belongs to the Auth domain:

- valid roles remain lowercase `owner`, `admin`, and `member`;
- Owner and Admin may manage ordinary roles;
- only Owner may promote to or demote from Owner;
- the final Owner cannot be demoted;
- member lists and role updates first require the authenticated User to belong
  to the request Workspace.

Transport authentication attaches the Auth User ID through the public
`WithMemberActor` context seam before invoking the application service. The
application and domain do not import gRPC, Kratos, SQLite, or Workspace
implementations.

`ProvisionWorkspaceOwner` is the Auth half of future Workspace creation. It
does not validate or create a Workspace and Auth never reads Workspace tables.
A later composition slice must coordinate Workspace persistence and this Auth
capability with an explicitly accepted atomic or compensating protocol before
the installed CreateWorkspace path can be cut over.

Login/session/profile mutation, invitations, removal/leave, Agent identity,
compatibility HTTP handlers, and production bootstrap wiring remain deferred.
