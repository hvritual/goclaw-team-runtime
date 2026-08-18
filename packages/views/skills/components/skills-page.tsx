"use client";

import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { AlertCircle, BookOpen, Plus, Search } from "lucide-react";
import type { SkillSummary } from "@multica/core/types";
import { useWorkspaceId } from "@multica/core/hooks";
import { useWorkspacePaths } from "@multica/core/paths";
import { useFeatureEnabled } from "@multica/core/config";
import { useCurrentMember } from "@multica/core/permissions";
import {
  memberListOptions,
  skillListOptions,
} from "@multica/core/workspace/queries";
import { Button } from "@multica/ui/components/ui/button";
import { Input } from "@multica/ui/components/ui/input";
import { useNavigation } from "../../navigation";
import {
  CollectionPageHeader,
  CollectionPageHeaderAction,
  CollectionPageState,
} from "../../layout/collection-page";
import { useT, useTimeAgo } from "../../i18n";
import { CreateSkillDialog } from "./create-skill-dialog";

export default function SkillsPage() {
  const { t } = useT("skills");
  const timeAgo = useTimeAgo();
  const workspaceId = useWorkspaceId();
  const paths = useWorkspacePaths();
  const navigation = useNavigation();
  const administrationInstalled = useFeatureEnabled(
    "skill_administration",
    false
  );
  const importInstalled = useFeatureEnabled("skill_import", false);
  const currentMember = useCurrentMember(workspaceId);
  const canAdminister =
    administrationInstalled &&
    (currentMember.role === "owner" || currentMember.role === "admin");
  const [search, setSearch] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const {
    data: skills = [],
    error,
    refetch,
  } = useQuery(skillListOptions(workspaceId));
  const { data: members = [] } = useQuery(memberListOptions(workspaceId));
  const memberNames = useMemo(
    () => new Map(members.map((member) => [member.user_id, member.name])),
    [members]
  );
  const visibleSkills = useMemo(() => {
    const query = search.trim().toLocaleLowerCase();
    if (!query) return skills;
    return skills.filter(
      (skill) =>
        skill.name.toLocaleLowerCase().includes(query) ||
        skill.description.toLocaleLowerCase().includes(query)
    );
  }, [search, skills]);

  const openSkill = (skill: SkillSummary) => {
    navigation.push(paths.skillDetail(skill.id));
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <CollectionPageHeader
        icon={BookOpen}
        title={t(($) => $.page.title)}
        count={skills.length}
        description={t(($) => $.page.tagline)}
        actions={
          canAdminister ? (
            <CollectionPageHeaderAction
              icon={Plus}
              label={t(($) => $.page.new_skill)}
              onClick={() => setCreateOpen(true)}
            />
          ) : undefined
        }
      />

      <main className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-4 overflow-y-auto p-4 sm:p-6">
        <div className="relative max-w-md">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder={t(($) => $.page.search_placeholder)}
            className="pl-9"
          />
        </div>

        {error ? (
          <CollectionPageState
            role="alert"
            tone="destructive"
            icon={AlertCircle}
            title={t(($) => $.page.list_error.title)}
            description={
              error instanceof Error
                ? error.message
                : t(($) => $.page.list_error.fallback)
            }
            actions={
              <Button variant="outline" onClick={() => void refetch()}>
                {t(($) => $.page.list_error.retry)}
              </Button>
            }
          />
        ) : visibleSkills.length === 0 ? (
          <CollectionPageState
            icon={BookOpen}
            title={
              search
                ? t(($) => $.page.no_matches.title)
                : t(($) => $.page.empty.title)
            }
            description={
              search
                ? t(($) => $.page.no_matches.with_query, {
                    query: search,
                    filterSuffix: "",
                  })
                : t(($) => $.page.empty.description)
            }
          />
        ) : (
          <div className="overflow-hidden rounded-lg border bg-card">
            <div className="grid grid-cols-[minmax(0,1fr)_10rem_8rem] gap-4 border-b bg-muted/40 px-4 py-2 text-xs font-medium text-muted-foreground">
              <span>{t(($) => $.table.name)}</span>
              <span className="hidden sm:block">
                {t(($) => $.table.created_by)}
              </span>
              <span className="text-right">{t(($) => $.table.updated)}</span>
            </div>
            <ul className="divide-y">
              {visibleSkills.map((skill) => (
                <li key={skill.id}>
                  <button
                    type="button"
                    onClick={() => openSkill(skill)}
                    className="grid w-full grid-cols-[minmax(0,1fr)_10rem_8rem] gap-4 px-4 py-3 text-left transition-colors hover:bg-muted/40"
                  >
                    <span className="min-w-0">
                      <span className="block truncate text-sm font-medium">
                        {skill.name}
                      </span>
                      {skill.description ? (
                        <span className="mt-0.5 block truncate text-xs text-muted-foreground">
                          {skill.description}
                        </span>
                      ) : null}
                    </span>
                    <span className="hidden truncate text-sm text-muted-foreground sm:block">
                      {skill.created_by
                        ? memberNames.get(skill.created_by) ?? "—"
                        : "—"}
                    </span>
                    <span className="text-right text-xs text-muted-foreground">
                      {timeAgo(skill.updated_at)}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          </div>
        )}
      </main>

      {createOpen && canAdminister ? (
        <CreateSkillDialog
          onClose={() => setCreateOpen(false)}
          onCreated={openSkill}
          allowImport={importInstalled}
        />
      ) : null}
    </div>
  );
}
