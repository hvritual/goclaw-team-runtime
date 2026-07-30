"use client";

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  parseTabSubject,
  resolveTabPresentation,
  useCurrentWorkspace,
  type TabSubject,
  type TabVisual,
  type TabTitleSpec,
  type TabEntityData,
  type TabLabelKey,
} from "@multica/core/paths";
import { issueDetailOptions } from "@multica/core/issues/queries";
import { projectDetailOptions } from "@multica/core/projects/queries";
import { taskDetailOptions } from "@multica/core/tasks/queries";
import {
  skillDetailOptions,
  memberListOptions,
} from "@multica/core/workspace/queries";
import { cn } from "@multica/ui/lib/utils";
import { StatusIcon } from "../issues/components";
import { ProjectIcon } from "../projects/components/project-icon";
import { ActorAvatar } from "../common/actor-avatar";
import { useT } from "../i18n";
import { ROUTE_ICON_COMPONENTS } from "./route-icon-components";

const NONE = "__tab_presentation_none__";
const PENDING_RESOURCE_KEYS: ReadonlySet<TabLabelKey> = new Set([
  "issue",
  "project",
  "task",
  "member",
  "skill",
]);

function useTabEntityData(subject: TabSubject, workspaceId: string): TabEntityData {
  const issue = useQuery({
    ...issueDetailOptions(
      workspaceId,
      subject.kind === "issue" ? subject.id : NONE,
    ),
    enabled: false,
  }).data;
  const project = useQuery({
    ...projectDetailOptions(
      workspaceId,
      subject.kind === "project" ? subject.id : NONE,
    ),
    enabled: false,
  }).data;
  const skill = useQuery({
    ...skillDetailOptions(
      workspaceId,
      subject.kind === "skill" ? subject.id : NONE,
    ),
    enabled: false,
  }).data;
  const task = useQuery({
    ...taskDetailOptions(
      workspaceId,
      subject.kind === "task" ? subject.id : NONE,
    ),
    enabled: false,
  }).data;
  const members = useQuery({
    ...memberListOptions(workspaceId),
    enabled: false,
  }).data;

  const data: TabEntityData = {};
  if (subject.kind === "issue" && issue) {
    data.issue = {
      identifier: issue.identifier,
      title: issue.title,
      status: issue.status,
    };
  } else if (subject.kind === "project" && project) {
    data.project = { icon: project.icon, title: project.title };
  } else if (subject.kind === "task" && task) {
    data.task = { title: task.title };
  } else if (subject.kind === "skill" && skill) {
    data.skill = { name: skill.name };
  } else if (subject.kind === "actor" && subject.actorType === "member") {
    const member = members?.find(
      (candidate) => candidate.user_id === subject.id,
    );
    if (member) data.actorName = member.name;
  }
  return data;
}

function useTabTitle(spec: TabTitleSpec, fallbackTitle?: string): string {
  const { t } = useT("layout");
  if (spec.kind === "text") return spec.text;
  if (spec.kind === "nav") return t(($) => $.nav[spec.navKey]);
  const fallback = fallbackTitle?.trim();
  if (fallback && PENDING_RESOURCE_KEYS.has(spec.tabKey)) return fallback;
  return t(($) => $.tab[spec.tabKey]);
}

export interface TabPresentationResult {
  visual: TabVisual;
  title: string;
}

export function useTabPresentation(
  url: string,
  fallbackTitle?: string,
): TabPresentationResult {
  const subject = useMemo(() => parseTabSubject(url), [url]);
  const workspaceId = useCurrentWorkspace()?.id ?? "";
  const data = useTabEntityData(subject, workspaceId);
  const presentation = resolveTabPresentation(subject, data);
  return {
    visual: presentation.visual,
    title: useTabTitle(presentation.title, fallbackTitle),
  };
}

export function ResourceLeadingVisual({
  visual,
  className,
}: {
  visual: TabVisual;
  className?: string;
}) {
  let inner: React.ReactNode;
  switch (visual.kind) {
    case "icon": {
      const Icon = ROUTE_ICON_COMPONENTS[visual.icon];
      inner = <Icon className="size-3.5" />;
      break;
    }
    case "issue-status":
      inner = <StatusIcon status={visual.status ?? ""} className="size-3.5" />;
      break;
    case "project-icon":
      inner = <ProjectIcon project={{ icon: visual.icon }} size="sm" />;
      break;
    case "actor":
      inner = (
        <ActorAvatar
          actorType={visual.actorType}
          actorId={visual.id}
          size="xs"
          profileLink={false}
        />
      );
      break;
  }

  return (
    <span
      className={cn(
        "flex size-4 shrink-0 items-center justify-center",
        className,
      )}
    >
      {inner}
    </span>
  );
}
