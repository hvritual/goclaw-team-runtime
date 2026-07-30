"use client";

import { useQuery } from "@tanstack/react-query";
import { Check, CircleDashed, Minus } from "lucide-react";
import { Button } from "@multica/ui/components/ui/button";
import { Card, CardContent } from "@multica/ui/components/ui/card";
import { Badge } from "@multica/ui/components/ui/badge";
import { useCurrentWorkspace } from "@multica/core/paths";
import {
  canViewPermissionManagement,
  useCurrentMember,
} from "@multica/core/permissions";
import { workspacePermissionOptions } from "@multica/core/workspace/queries";
import type { MemberRole } from "@multica/core/types";
import { SettingsSection, SettingsTab } from "./settings-layout";
import { useT } from "../../i18n";
import { useNavigation } from "../../navigation";

export function RolesTab() {
  const { t } = useT("settings");
  const workspace = useCurrentWorkspace();
  const currentMember = useCurrentMember(workspace?.id ?? "");
  const navigation = useNavigation();
  const canView = canViewPermissionManagement({
    userId: currentMember.userId,
    role: currentMember.role,
  }).allowed;
  const catalog = useQuery(
    workspacePermissionOptions(workspace?.id ?? "", canView),
  );

  if (!canView && !currentMember.isLoading) return null;

  const openMembers = () => {
    const params = new URLSearchParams(navigation.searchParams);
    params.set("tab", "members");
    navigation.replace(`${navigation.pathname}?${params.toString()}`);
  };
  const visibleRoles = catalog.data
    ? fixedPermissionRoles.flatMap((key) => {
        const role = catalog.data.roles.find(
          (candidate) => candidate.key === key,
        );
        return role ? [role] : [];
      })
    : [];
  const hasCompleteCatalog =
    visibleRoles.length === fixedPermissionRoles.length &&
    permissionDomains.every((domain) =>
      catalog.data?.capabilities.some(
        (capability) => capability.domain === domain,
      ),
    );

  return (
    <SettingsTab
      title={t(($) => $.page.tabs.roles)}
      description={t(($) => $.permissions.description)}
    >
      <SettingsSection>
        <Card className="shadow-none">
          <CardContent className="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <div className="text-sm font-medium">
                {t(($) => $.permissions.current_role, {
                  role: roleLabel(currentMember.role, t),
                })}
              </div>
              <p className="mt-1 text-xs leading-5 text-muted-foreground">
                {t(($) => $.permissions.assignment_hint)}
              </p>
            </div>
            <Button variant="outline" onClick={openMembers}>
              {t(($) => $.permissions.manage_members)}
            </Button>
          </CardContent>
        </Card>
      </SettingsSection>

      <SettingsSection
        title={t(($) => $.permissions.matrix_title)}
        description={t(($) => $.permissions.matrix_description)}
      >
        {catalog.isLoading || currentMember.isLoading ? (
          <Card className="shadow-none">
            <CardContent className="p-6 text-sm text-muted-foreground">
              {t(($) => $.permissions.loading)}
            </CardContent>
          </Card>
        ) : catalog.isError || !catalog.data || !hasCompleteCatalog ? (
          <Card className="border-destructive/30 shadow-none">
            <CardContent className="p-6 text-sm text-destructive">
              {t(($) => $.permissions.load_failed)}
            </CardContent>
          </Card>
        ) : (
          <div className="overflow-x-auto rounded-xl border border-surface-border">
            <table className="w-full min-w-[640px] border-collapse text-sm">
              <thead className="bg-muted/40">
                <tr>
                  <th className="px-4 py-3 text-left font-medium">
                    {t(($) => $.permissions.capability)}
                  </th>
                  {visibleRoles.map((role) => (
                    <th
                      key={role.key}
                      className="w-36 px-4 py-3 text-left font-medium"
                    >
                      <span className="inline-flex items-center gap-2">
                        {roleLabel(role.key, t)}
                        {currentMember.role === role.key ? (
                          <Badge variant="secondary">
                            {t(($) => $.permissions.you)}
                          </Badge>
                        ) : null}
                      </span>
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {permissionDomains.map((domain) => {
                  const capabilities = catalog.data.capabilities.filter(
                    (capability) => capability.domain === domain,
                  );
                  if (capabilities.length === 0) return null;
                  return [
                    <tr key={`${domain}-heading`} className="border-t border-surface-border bg-muted/20">
                      <th
                        colSpan={4}
                        className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-muted-foreground"
                      >
                        {domainLabel(domain, t)}
                      </th>
                    </tr>,
                    ...capabilities.map((capability) => (
                      <tr
                        key={capability.key}
                        className="border-t border-surface-border"
                      >
                        <th className="px-4 py-3 text-left font-medium">
                          {capabilityLabel(capability.key, t)}
                        </th>
                        {visibleRoles.map((role) => (
                          <td key={role.key} className="px-4 py-3">
                            <AccessCell
                              access={capability.access[role.key]}
                              label={accessLabel(
                                capability.access[role.key],
                                t,
                              )}
                            />
                          </td>
                        ))}
                      </tr>
                    )),
                  ];
                })}
              </tbody>
            </table>
          </div>
        )}
      </SettingsSection>
    </SettingsTab>
  );
}

const permissionDomains = [
  "workspace",
  "member",
  "project",
  "issue",
  "task",
  "skill",
] as const;
const fixedPermissionRoles = ["owner", "admin", "member"] as const;

type Translator = ReturnType<typeof useT<"settings">>["t"];

function roleLabel(role: MemberRole | null, t: Translator): string {
  if (role === "owner" || role === "admin" || role === "member") return role;
  return t(($) => $.permissions.unknown_role);
}

function domainLabel(
  domain: (typeof permissionDomains)[number],
  t: Translator,
): string {
  switch (domain) {
    case "workspace":
      return t(($) => $.permissions.domains.workspace);
    case "member":
      return t(($) => $.permissions.domains.member);
    case "project":
      return t(($) => $.permissions.domains.project);
    case "issue":
      return t(($) => $.permissions.domains.issue);
    case "task":
      return t(($) => $.permissions.domains.task);
    case "skill":
      return t(($) => $.permissions.domains.skill);
    default:
      return domain;
  }
}

function accessLabel(access: string, t: Translator): string {
  switch (access) {
    case "allowed":
      return t(($) => $.permissions.access.allowed);
    case "conditional":
      return t(($) => $.permissions.access.conditional);
    case "denied":
      return t(($) => $.permissions.access.denied);
    default:
      return access;
  }
}

function AccessCell({
  access,
  label,
}: {
  access: string;
  label: string;
}) {
  const Icon =
    access === "allowed"
      ? Check
      : access === "conditional"
        ? CircleDashed
        : Minus;
  return (
    <span className="inline-flex items-center gap-1.5 text-xs text-muted-foreground">
      <Icon
        className={
          access === "allowed"
            ? "size-3.5 text-success"
            : access === "conditional"
              ? "size-3.5 text-warning"
              : "size-3.5"
        }
      />
      {label}
    </span>
  );
}

function capabilityLabel(key: string, t: Translator): string {
  switch (key) {
    case "workspace.view":
      return t(($) => $.permissions.capabilities.workspace_view);
    case "workspace.update":
      return t(($) => $.permissions.capabilities.workspace_update);
    case "workspace.delete":
      return t(($) => $.permissions.capabilities.workspace_delete);
    case "member.view":
      return t(($) => $.permissions.capabilities.member_view);
    case "member.invite":
      return t(($) => $.permissions.capabilities.member_invite);
    case "member.change_role":
      return t(($) => $.permissions.capabilities.member_change_role);
    case "member.manage_owner":
      return t(($) => $.permissions.capabilities.member_manage_owner);
    case "member.remove":
      return t(($) => $.permissions.capabilities.member_remove);
    case "project.view":
      return t(($) => $.permissions.capabilities.project_view);
    case "project.create":
      return t(($) => $.permissions.capabilities.project_create);
    case "project.update":
      return t(($) => $.permissions.capabilities.project_update);
    case "project.delete":
      return t(($) => $.permissions.capabilities.project_delete);
    case "issue.view":
      return t(($) => $.permissions.capabilities.issue_view);
    case "issue.create":
      return t(($) => $.permissions.capabilities.issue_create);
    case "issue.update":
      return t(($) => $.permissions.capabilities.issue_update);
    case "issue.delete":
      return t(($) => $.permissions.capabilities.issue_delete);
    case "task.view":
      return t(($) => $.permissions.capabilities.task_view);
    case "task.create":
      return t(($) => $.permissions.capabilities.task_create);
    case "task.update":
      return t(($) => $.permissions.capabilities.task_update);
    case "task.delete":
      return t(($) => $.permissions.capabilities.task_delete);
    case "skill.view":
      return t(($) => $.permissions.capabilities.skill_view);
    case "skill.create":
      return t(($) => $.permissions.capabilities.skill_create);
    case "skill.update":
      return t(($) => $.permissions.capabilities.skill_update);
    case "skill.delete":
      return t(($) => $.permissions.capabilities.skill_delete);
    default:
      return key;
  }
}
