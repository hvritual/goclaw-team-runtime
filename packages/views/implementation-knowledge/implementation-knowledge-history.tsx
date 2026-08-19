"use client";

import { useState } from "react";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { ApiError } from "@multica/core/api";
import {
  acceptanceConclusionListOptions,
  projectRetrospectiveInfiniteListOptions,
  useArchiveProjectRetrospective,
  useCreateProjectRetrospective,
  useCreateProjectRetrospectiveTarget,
  useUpdateProjectRetrospective,
} from "@multica/core/implementation-knowledge";
import { useWorkspaceId } from "@multica/core/hooks";
import { useWorkspacePaths } from "@multica/core/paths";
import type { MemberWithUser, ProjectRetrospectiveInput } from "@multica/core/types";
import type {
  ProjectRetrospective,
  ProjectRetrospectiveRevisionStatus,
} from "@multica/core/types/implementation-knowledge";
import { Button } from "@multica/ui/components/ui/button";
import { toast } from "sonner";
import { AppLink } from "../navigation";
import { useT } from "../i18n";
import { ProjectRetrospectiveDialog } from "./implementation-knowledge-dialogs";

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

type EditorState =
  | { mode: "create"; retrospective?: undefined }
  | { mode: "save_draft" | "publish_revision"; retrospective: ProjectRetrospective };

export function ProjectRetrospectiveHistory({
  projectId,
  members = [],
}: {
  projectId: string;
  members?: MemberWithUser[];
}) {
  const workspaceId = useWorkspaceId();
  const paths = useWorkspacePaths();
  const { t } = useT("projects");
  const query = useInfiniteQuery(
    projectRetrospectiveInfiniteListOptions(workspaceId, projectId, true),
  );
  const create = useCreateProjectRetrospective(projectId);
  const update = useUpdateProjectRetrospective(projectId);
  const archive = useArchiveProjectRetrospective(projectId);
  const target = useCreateProjectRetrospectiveTarget(projectId);
  const [editor, setEditor] = useState<EditorState | null>(null);
  const retrospectives = query.data?.retrospectives ?? [];

  const statusLabels: Record<ProjectRetrospectiveRevisionStatus, string> = {
    draft: t(($) => $.implementation_knowledge.status_draft),
    published: t(($) => $.implementation_knowledge.status_published),
    superseded: t(($) => $.implementation_knowledge.status_superseded),
    archived: t(($) => $.implementation_knowledge.status_archived),
  };
  const memberNames = new Map<string, string>();
  for (const member of members) {
    memberNames.set(member.id, member.name);
    memberNames.set(member.user_id, member.name);
  }
  const memberName = (id: string) => memberNames.get(id) ?? id;
  const failure = (key: "failed" | "save_failed" | "publish_failed" | "archive_failed" | "target_failed") =>
    () => toast.error(t(($) => $.implementation_knowledge[key]));

  const submitEditor = (input: ProjectRetrospectiveInput) => {
    if (!editor) return;
    if (editor.mode === "create") {
      create.mutate(input, {
        onSuccess: () => {
          setEditor(null);
          toast.success(t(($) => $.implementation_knowledge.success));
        },
        onError: failure("failed"),
      });
      return;
    }
    update.mutate({
      retrospectiveId: editor.retrospective.id,
      expectedRevision: editor.retrospective.currentRevision,
      action: editor.mode,
      content: input.content,
      participants: input.participants,
    }, {
      onSuccess: () => {
        setEditor(null);
        toast.success(t(($) => $.implementation_knowledge.saved));
      },
      onError: failure("save_failed"),
    });
  };

  return (
    <section aria-labelledby="project-retrospectives-heading" className="space-y-2">
      <div className="flex items-center justify-between gap-2 px-2">
        <h3 id="project-retrospectives-heading" className="text-xs font-medium">
          {t(($) => $.implementation_knowledge.history_title)}
        </h3>
        {!query.error ? (
          <Button type="button" size="sm" variant="outline" onClick={() => setEditor({ mode: "create" })}>
            {t(($) => $.implementation_knowledge.record_action)}
          </Button>
        ) : null}
      </div>

      {query.isLoading ? (
        <p className="px-2 text-xs text-muted-foreground" aria-live="polite">
          {t(($) => $.implementation_knowledge.loading)}
        </p>
      ) : query.error ? (
        <p className="px-2 text-xs text-destructive" role="alert">
          {query.error instanceof ApiError && query.error.status === 403
            ? t(($) => $.implementation_knowledge.denied)
            : t(($) => $.implementation_knowledge.load_failed)}
        </p>
      ) : retrospectives.length === 0 ? (
        <p className="px-2 text-xs text-muted-foreground">
          {t(($) => $.implementation_knowledge.empty)}
        </p>
      ) : (
        <div className="space-y-3 pl-2">
          {retrospectives.map((retrospective) => (
            <article key={retrospective.id} className="space-y-3 rounded-lg border bg-muted/20 p-3 text-xs">
              <div className="flex items-start justify-between gap-2">
                <p className="min-w-0 whitespace-pre-wrap font-medium">{retrospective.current.content.summary}</p>
                <span className="shrink-0 rounded-full border px-2 py-0.5 text-[10px]">
                  {statusLabels[retrospective.status]}
                </span>
              </div>

              <div className="space-y-2 text-muted-foreground">
                {retrospective.current.content.successes.length > 0 ? (
                  <ContentList label={t(($) => $.implementation_knowledge.successes)} values={retrospective.current.content.successes} />
                ) : null}
                {retrospective.current.content.problems.length > 0 ? (
                  <ContentList label={t(($) => $.implementation_knowledge.problems)} values={retrospective.current.content.problems} />
                ) : null}
                <ContentList label={t(($) => $.implementation_knowledge.lessons)} values={retrospective.current.content.lessons} />
                <p>
                  {t(($) => $.implementation_knowledge.participants)}: {retrospective.current.participants.map((participant) => (
                    `${memberName(participant.memberId)} (${participant.role === "facilitator"
                      ? t(($) => $.implementation_knowledge.role_facilitator)
                      : t(($) => $.implementation_knowledge.role_participant)})`
                  )).join(", ")}
                </p>
              </div>

              {retrospective.current.content.actionItems.length > 0 ? (
                <div className="space-y-2">
                  <h4 className="font-medium">{t(($) => $.implementation_knowledge.action_items)}</h4>
                  {retrospective.current.content.actionItems.map((item) => {
                    const link = retrospective.actionLinks.find((candidate) => candidate.actionItemId === item.id);
                    return (
                      <div key={item.id} className="space-y-1 rounded-md border bg-background/60 p-2">
                        <p className="font-medium">{item.title}</p>
                        {item.description ? <p className="whitespace-pre-wrap text-muted-foreground">{item.description}</p> : null}
                        {item.assigneeId ? <p className="text-muted-foreground">{memberName(item.assigneeId)}</p> : null}
                        {item.dueDate ? <time className="text-muted-foreground" dateTime={item.dueDate}>{item.dueDate}</time> : null}
                        {link?.state === "linked" && link.targetId ? (
                          <AppLink
                            href={link.targetKind === "task"
                              ? paths.taskDetail(link.targetId)
                              : paths.issueDetail(link.targetId)}
                            className="inline-block text-foreground underline underline-offset-2"
                          >
                            {link.targetKind === "task"
                              ? t(($) => $.implementation_knowledge.target_task)
                              : t(($) => $.implementation_knowledge.target_issue)} {link.targetId}
                          </AppLink>
                        ) : link?.state === "pending" ? (
                          <div className="flex flex-wrap items-center gap-2">
                            <span role="status">{t(($) => $.implementation_knowledge.target_pending)}</span>
                            {retrospective.access.canPublish ? (
                              <Button
                                type="button"
                                size="sm"
                                variant="outline"
                                disabled={target.isPending}
                                onClick={() => target.mutate({
                                  retrospectiveId: retrospective.id,
                                  actionItemId: item.id,
                                  targetKind: link.targetKind,
                                }, {
                                  onSuccess: () => toast.success(t(($) => $.implementation_knowledge.target_success)),
                                  onError: failure("target_failed"),
                                })}
                              >
                                {link.targetKind === "task"
                                  ? t(($) => $.implementation_knowledge.retry_task)
                                  : t(($) => $.implementation_knowledge.retry_issue)}
                              </Button>
                            ) : null}
                          </div>
                        ) : retrospective.status === "published" && retrospective.access.canPublish ? (
                          <div className="flex flex-wrap gap-2">
                            <Button
                              type="button"
                              size="sm"
                              variant="outline"
                              disabled={target.isPending}
                              onClick={() => target.mutate({
                                retrospectiveId: retrospective.id,
                                actionItemId: item.id,
                              }, {
                                onSuccess: () => toast.success(t(($) => $.implementation_knowledge.target_success)),
                                onError: failure("target_failed"),
                              })}
                            >
                              {t(($) => $.implementation_knowledge.create_task)}: {item.title}
                            </Button>
                            <Button
                              type="button"
                              size="sm"
                              variant="outline"
                              disabled={target.isPending}
                              onClick={() => target.mutate({
                                retrospectiveId: retrospective.id,
                                actionItemId: item.id,
                                targetKind: "issue",
                              }, {
                                onSuccess: () => toast.success(t(($) => $.implementation_knowledge.target_success)),
                                onError: failure("target_failed"),
                              })}
                            >
                              {t(($) => $.implementation_knowledge.create_issue)}: {item.title}
                            </Button>
                          </div>
                        ) : null}
                      </div>
                    );
                  })}
                </div>
              ) : null}

              <div className="flex flex-wrap gap-2">
                {retrospective.status === "draft" && retrospective.access.canEdit ? (
                  <Button type="button" size="sm" variant="outline" onClick={() => setEditor({ mode: "save_draft", retrospective })}>
                    {t(($) => $.implementation_knowledge.edit_draft)}
                  </Button>
                ) : null}
                {retrospective.status === "draft" && retrospective.access.canPublish ? (
                  <Button
                    type="button"
                    size="sm"
                    disabled={update.isPending}
                    onClick={() => update.mutate({
                      retrospectiveId: retrospective.id,
                      expectedRevision: retrospective.currentRevision,
                      action: "publish",
                    }, {
                      onSuccess: () => toast.success(t(($) => $.implementation_knowledge.published_success)),
                      onError: failure("publish_failed"),
                    })}
                  >
                    {t(($) => $.implementation_knowledge.publish)}
                  </Button>
                ) : null}
                {retrospective.status === "published" && retrospective.access.canPublish ? (
                  <Button type="button" size="sm" variant="outline" onClick={() => setEditor({ mode: "publish_revision", retrospective })}>
                    {t(($) => $.implementation_knowledge.publish_revision)}
                  </Button>
                ) : null}
                {retrospective.access.canArchive ? (
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    disabled={archive.isPending}
                    onClick={() => archive.mutate({
                      retrospectiveId: retrospective.id,
                      expectedRevision: retrospective.currentRevision,
                    }, {
                      onSuccess: () => toast.success(t(($) => $.implementation_knowledge.archived_success)),
                      onError: failure("archive_failed"),
                    })}
                  >
                    {t(($) => $.implementation_knowledge.archive)}
                  </Button>
                ) : null}
              </div>

              <details>
                <summary className="cursor-pointer font-medium">
                  {t(($) => $.implementation_knowledge.revision_history)} ({retrospective.history.length})
                </summary>
                <ol className="mt-2 space-y-2 border-l pl-3">
                  {retrospective.history.map((revision) => (
                    <li key={revision.revision} className="space-y-1">
                      <div className="flex items-center justify-between gap-2">
                        <span>{t(($) => $.implementation_knowledge.revision)} {revision.revision}</span>
                        <span>{statusLabels[revision.status]}</span>
                      </div>
                      <p className="whitespace-pre-wrap text-muted-foreground">{revision.content.summary}</p>
                      <ContentList label={t(($) => $.implementation_knowledge.lessons)} values={revision.content.lessons} />
                    </li>
                  ))}
                </ol>
              </details>
            </article>
          ))}
          {query.hasNextPage ? (
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={query.isFetchingNextPage}
              onClick={() => void query.fetchNextPage()}
            >
              {t(($) => $.implementation_knowledge.load_more)}
            </Button>
          ) : null}
        </div>
      )}

      <ProjectRetrospectiveDialog
        open={editor !== null}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) setEditor(null);
        }}
        mode={editor?.mode ?? "create"}
        members={members}
        initialContent={editor?.retrospective?.current.content}
        initialParticipants={editor?.retrospective?.current.participants}
        pending={create.isPending || update.isPending}
        onSubmit={submitEditor}
      />
    </section>
  );
}

function ContentList({ label, values }: { label: string; values: string[] }) {
  return (
    <div>
      <span>{label}:</span>
      <ul className="list-disc space-y-0.5 pl-4">
        {values.map((value) => <li key={value} className="whitespace-pre-wrap">{value}</li>)}
      </ul>
    </div>
  );
}
