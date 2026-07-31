"use client";

import { useQuery } from "@tanstack/react-query";
import {
  acceptanceConclusionListOptions,
  projectRetrospectiveListOptions,
} from "@multica/core/implementation-knowledge";
import { useWorkspaceId } from "@multica/core/hooks";
import { useWorkspacePaths } from "@multica/core/paths";
import { AppLink } from "../navigation";
import { useT } from "../i18n";

export function AcceptanceConclusionHistory({ issueId }: { issueId: string }) {
  const workspaceId = useWorkspaceId();
  const paths = useWorkspacePaths();
  const { t } = useT("issues");
  const { data } = useQuery(acceptanceConclusionListOptions(workspaceId, issueId));
  if (!data?.acceptanceConclusions.length) return null;
  const resultLabels = {
    accepted: t(($) => $.implementation_knowledge.result_accepted),
    conditional: t(($) => $.implementation_knowledge.result_conditional),
    rejected: t(($) => $.implementation_knowledge.result_rejected),
  };
  return (
    <section className="space-y-2">
      <div className="flex items-center justify-between">
        <h3 className="text-xs font-medium">{t(($) => $.implementation_knowledge.history_title)}</h3>
      </div>
      {data.acceptanceConclusions.map((item) => (
        <div key={item.id} className="rounded-lg border bg-muted/20 p-2.5 text-xs">
          <div className="font-medium">{resultLabels[item.result]}</div>
          <p className="mt-1 whitespace-pre-wrap text-muted-foreground">{item.rationale}</p>
          <AppLink
            href={`${paths.knowledge()}?source_type=acceptance_conclusion&source_id=${encodeURIComponent(issueId)}`}
            className="mt-2 inline-block text-muted-foreground hover:text-foreground"
          >
            {t(($) => $.implementation_knowledge.knowledge_link)}
          </AppLink>
        </div>
      ))}
    </section>
  );
}

export function ProjectRetrospectiveHistory({ projectId }: { projectId: string }) {
  const workspaceId = useWorkspaceId();
  const paths = useWorkspacePaths();
  const { t } = useT("projects");
  const { data } = useQuery(projectRetrospectiveListOptions(workspaceId, projectId));
  if (!data?.retrospectives.length) return null;
  return (
    <section className="space-y-2">
      <div className="flex items-center justify-between">
        <h3 className="text-xs font-medium">{t(($) => $.implementation_knowledge.history_title)}</h3>
      </div>
      {data.retrospectives.map((item) => (
        <div key={item.id} className="rounded-lg border bg-muted/20 p-2.5 text-xs">
          <p>{item.summary}</p>
          <p className="mt-1 text-muted-foreground">{t(($) => $.implementation_knowledge.lesson_prefix)} {item.lessons.join("；")}</p>
          <AppLink
            href={`${paths.knowledge()}?source_type=retrospective&source_id=${encodeURIComponent(item.id)}`}
            className="mt-2 inline-block text-muted-foreground hover:text-foreground"
          >
            {t(($) => $.implementation_knowledge.knowledge_link)}
          </AppLink>
        </div>
      ))}
    </section>
  );
}
