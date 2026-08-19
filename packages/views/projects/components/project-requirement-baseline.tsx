"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import { ApiError } from "@multica/core/api";
import { useWorkspaceId } from "@multica/core/hooks";
import {
  projectOutlineOptions,
  projectRequirementBaselineOptions,
  projectRequirementIssuesOptions,
  useApproveProjectRequirement,
  useCreateProjectOutlineNode,
  useFreezeProjectRequirement,
  useLinkProjectRequirementIssue,
  useLinkProjectRequirementOutline,
  useRetireProjectRequirement,
  useSaveProjectRequirementDraft,
  useSubmitProjectRequirementReview,
  useUnlinkProjectRequirementIssue,
  useUnlinkProjectRequirementOutline,
  useWithdrawProjectRequirementReview,
} from "@multica/core/project-requirements";
import type {
  Issue,
  ProjectOutlineNode,
  ProjectRequirementAction,
  ProjectRequirementContent,
  ProjectRequirementIssueLink,
  ProjectRequirementItem,
  ProjectRequirementOutlineLink,
  ProjectRequirementStatus,
} from "@multica/core/types";
import { useActorName } from "@multica/core/workspace/hooks";
import { Button } from "@multica/ui/components/ui/button";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { useT } from "../../i18n";

const EMPTY_CONTENT: ProjectRequirementContent = {
  problemStatement: "",
  goals: [],
  inScope: [],
  outOfScope: [],
  constraints: [],
  acceptanceCriteria: [],
  dependencies: [],
};

type ContentListKey = Exclude<keyof ProjectRequirementContent, "problemStatement">;

const SECTIONS: ContentListKey[] = [
  "goals",
  "inScope",
  "outOfScope",
  "constraints",
  "acceptanceCriteria",
  "dependencies",
];

const TRACEABLE_SECTIONS = new Set<ContentListKey>([
  "goals",
  "inScope",
  "constraints",
  "acceptanceCriteria",
]);

function itemKey(): string {
  return (
    globalThis.crypto?.randomUUID?.() ??
    `item-${Date.now()}-${Math.random().toString(36).slice(2)}`
  );
}

export function updateRequirementItem(
  items: ProjectRequirementItem[],
  key: string,
  text: string
): ProjectRequirementItem[] {
  return items.map((item) => (item.key === key ? { ...item, text } : item));
}

export function removeRequirementItem(
  items: ProjectRequirementItem[],
  key: string
): ProjectRequirementItem[] {
  return items.filter((item) => item.key !== key);
}

export function appendRequirementItem(
  items: ProjectRequirementItem[]
): ProjectRequirementItem[] {
  let key = itemKey();
  while (items.some((item) => item.key === key)) key = itemKey();
  return [...items, { key, text: "" }];
}

export function formatRequirementAuditTime(value: string | null): string | null {
  if (!value) return null;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return null;
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

type ProjectRequirementProblem =
  | { code: "revision_conflict"; currentRevision?: number }
  | { code: "independent_approval_required" }
  | { code: "permission_denied" }
  | { code: "invalid_transition" }
  | { code: "unknown" };

export function getProjectRequirementError(error: unknown): ProjectRequirementProblem {
  if (!(error instanceof ApiError) || !error.body || typeof error.body !== "object") {
    return { code: "unknown" };
  }
  const body = error.body as { code?: unknown; current_revision?: unknown };
  if (body.code === "revision_conflict") {
    return {
      code: "revision_conflict",
      ...(typeof body.current_revision === "number" &&
      Number.isSafeInteger(body.current_revision) &&
      body.current_revision >= 0
        ? { currentRevision: body.current_revision }
        : {}),
    };
  }
  if (
    body.code === "independent_approval_required" ||
    body.code === "permission_denied" ||
    body.code === "invalid_transition"
  ) {
    return { code: body.code };
  }
  return { code: "unknown" };
}

function EffectiveBaseline({
  revision,
  content,
}: {
  revision: number;
  content: ProjectRequirementContent;
}) {
  const { t } = useT("projects");
  return (
    <section className="mb-5 rounded-md border bg-muted/20 p-3">
      <div className="flex items-center justify-between gap-3">
        <h3 className="text-sm font-medium">
          {t(($) => $.requirements.effective_baseline)}
        </h3>
        <span className="text-xs text-muted-foreground">
          {t(($) => $.requirements.effective_version, { revision })}
        </span>
      </div>
      <p className="mt-2 whitespace-pre-wrap text-sm text-muted-foreground">
        {content.problemStatement}
      </p>
    </section>
  );
}

function MinimalOutlineRoots({
  revision,
  canCreate,
  busy,
  title,
  onTitleChange,
  onCreate,
}: {
  revision: number;
  canCreate: boolean;
  busy: boolean;
  title: string;
  onTitleChange: (value: string) => void;
  onCreate: () => void;
}) {
  const { t } = useT("projects");
  if (!canCreate) return null;
  return (
    <section className="mb-5 rounded-md border p-3">
      <div className="flex flex-wrap items-end gap-2">
        <label className="min-w-56 flex-1 space-y-1">
          <span className="text-sm font-medium">
            {t(($) => $.requirements.new_outline_root)}
          </span>
          <input
            aria-label={t(($) => $.requirements.new_outline_root)}
            value={title}
            disabled={busy}
            onChange={(event) => onTitleChange(event.target.value)}
            className="w-full rounded-md border bg-background px-3 py-2 text-sm"
          />
        </label>
        <Button
          type="button"
          variant="secondary"
          disabled={busy || !title.trim()}
          onClick={onCreate}
        >
          {t(($) => $.requirements.create_outline_root)}
        </Button>
      </div>
      <p className="mt-2 text-xs text-muted-foreground">
        {t(($) => $.requirements.outline_revision, { revision })}
      </p>
    </section>
  );
}

function RequirementTrackingControls({
  requirementKey,
  issueLinks,
  outlineLinks,
  availableIssues,
  outlineNodes,
  canLinkIssues,
  canLinkOutline,
  busy,
  onLinkIssue,
  onUnlinkIssue,
  onLinkOutline,
  onUnlinkOutline,
}: {
  requirementKey: string;
  issueLinks: ProjectRequirementIssueLink[];
  outlineLinks: ProjectRequirementOutlineLink[];
  availableIssues: Issue[];
  outlineNodes: ProjectOutlineNode[];
  canLinkIssues: boolean;
  canLinkOutline: boolean;
  busy: boolean;
  onLinkIssue: (requirementKey: string, issueId: string) => void;
  onUnlinkIssue: (requirementKey: string, issueId: string) => void;
  onLinkOutline: (requirementKey: string, nodeId: string) => void;
  onUnlinkOutline: (requirementKey: string, nodeId: string) => void;
}) {
  const { t } = useT("projects");
  const linkedIssueIds = new Set(issueLinks.map((link) => link.issueId));
  const linkedNodeIds = new Set(outlineLinks.map((link) => link.nodeId));
  return (
    <div className="mt-2 space-y-2 text-sm">
      <div className="flex flex-wrap gap-2">
        {issueLinks.map((link) => (
          <span key={link.issueId} className="rounded bg-muted px-2 py-1">
            <span>{link.identifier} · {link.title} · {link.status}</span>
            {link.reviewRequired && (
              <span className="ml-2 text-amber-700 dark:text-amber-400">
                {t(($) => $.requirements.review_required)}
              </span>
            )}
            {canLinkIssues && (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                aria-label={t(($) => $.requirements.unlink_issue_label, {
                  identifier: link.identifier,
                })}
                disabled={busy}
                onClick={() => onUnlinkIssue(requirementKey, link.issueId)}
              >
                {t(($) => $.requirements.unlink_issue)}
              </Button>
            )}
          </span>
        ))}
      </div>
      <div className="flex flex-wrap gap-2">
        {outlineLinks.map((link) => (
          <span key={link.nodeId} className="rounded border px-2 py-1">
            <span>{link.nodeTitle}</span>
            {canLinkOutline && (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                aria-label={t(($) => $.requirements.unlink_outline_label, {
                  title: link.nodeTitle,
                })}
                disabled={busy}
                onClick={() => onUnlinkOutline(requirementKey, link.nodeId)}
              >
                {t(($) => $.requirements.unlink_outline)}
              </Button>
            )}
          </span>
        ))}
      </div>
      <div className="flex flex-wrap gap-2">
        {canLinkIssues && (
          <select
            aria-label={t(($) => $.requirements.link_existing)}
            defaultValue=""
            disabled={busy}
            onChange={(event) => {
              if (event.target.value) {
                onLinkIssue(requirementKey, event.target.value);
                event.target.value = "";
              }
            }}
            className="rounded-md border bg-background px-2 py-1"
          >
            <option value="">{t(($) => $.requirements.link_existing)}</option>
            {availableIssues
              .filter((issue) => !linkedIssueIds.has(issue.id))
              .map((issue) => (
                <option key={issue.id} value={issue.id}>
                  {issue.identifier} · {issue.title}
                </option>
              ))}
          </select>
        )}
        {canLinkOutline && (
          <select
            aria-label={t(($) => $.requirements.link_outline_root)}
            defaultValue=""
            disabled={busy}
            onChange={(event) => {
              if (event.target.value) {
                onLinkOutline(requirementKey, event.target.value);
                event.target.value = "";
              }
            }}
            className="rounded-md border bg-background px-2 py-1"
          >
            <option value="">{t(($) => $.requirements.link_outline_root)}</option>
            {outlineNodes
              .filter((node) => !linkedNodeIds.has(node.id))
              .map((node) => (
                <option key={node.id} value={node.id}>
                  {t(($) => $.requirements.outline_root_option, { title: node.title })}
                </option>
              ))}
          </select>
        )}
      </div>
    </div>
  );
}

export function ProjectRequirementBaseline({ projectId }: { projectId: string }) {
  const { t } = useT("projects");
  const wsId = useWorkspaceId();
  const { getActorName } = useActorName();
  const { data, isLoading, dataUpdatedAt } = useQuery(
    projectRequirementBaselineOptions(wsId, projectId)
  );
  const { data: projectIssues } = useQuery(
    projectRequirementIssuesOptions(wsId, projectId)
  );
  const { data: outline } = useQuery(projectOutlineOptions(wsId, projectId));
  const baseline = data?.baseline;
  const currentContent = data?.currentContent;
  const currentRevision = baseline?.currentRevision ?? 0;
  const status = baseline?.status;
  const access = data?.access ?? {
    canEdit: false,
    canApprove: false,
    canManageAccess: false,
    canManageOutline: false,
  };

  const [content, setContent] = useState<ProjectRequirementContent>(EMPTY_CONTENT);
  const [changeSummary, setChangeSummary] = useState("");
  const [materialChange, setMaterialChange] = useState(false);
  const [outlineTitle, setOutlineTitle] = useState("");
  const currentContentRef = useRef(currentContent);
  currentContentRef.current = currentContent;

  const save = useSaveProjectRequirementDraft(wsId, projectId);
  const submit = useSubmitProjectRequirementReview(wsId, projectId);
  const approve = useApproveProjectRequirement(wsId, projectId);
  const withdraw = useWithdrawProjectRequirementReview(wsId, projectId);
  const freeze = useFreezeProjectRequirement(wsId, projectId);
  const retire = useRetireProjectRequirement(wsId, projectId);
  const linkIssue = useLinkProjectRequirementIssue(wsId, projectId);
  const unlinkIssue = useUnlinkProjectRequirementIssue(wsId, projectId);
  const linkOutline = useLinkProjectRequirementOutline(wsId, projectId);
  const unlinkOutline = useUnlinkProjectRequirementOutline(wsId, projectId);
  const createOutline = useCreateProjectOutlineNode(wsId, projectId);

  useEffect(() => {
    setContent(currentContentRef.current ?? EMPTY_CONTENT);
    setChangeSummary("");
    setMaterialChange(false);
  }, [dataUpdatedAt]);

  const isBusy =
    save.isPending ||
    submit.isPending ||
    approve.isPending ||
    withdraw.isPending ||
    freeze.isPending ||
    retire.isPending ||
    linkIssue.isPending ||
    unlinkIssue.isPending ||
    linkOutline.isPending ||
    unlinkOutline.isPending ||
    createOutline.isPending;

  const history = useMemo(() => data?.history ?? [], [data?.history]);
  const savedItems = useMemo(() => {
    const result = new Map<string, { section: ContentListKey; text: string }>();
    if (!currentContent) return result;
    for (const section of SECTIONS) {
      for (const item of currentContent[section]) {
        result.set(item.key, { section, text: item.text });
      }
    }
    return result;
  }, [currentContent]);
  const issueLinksByKey = useMemo(() => {
    const result = new Map<string, ProjectRequirementIssueLink[]>();
    for (const link of data?.issueLinks ?? []) {
      const links = result.get(link.requirementKey) ?? [];
      links.push(link);
      result.set(link.requirementKey, links);
    }
    return result;
  }, [data?.issueLinks]);
  const outlineLinksByKey = useMemo(() => {
    const result = new Map<string, ProjectRequirementOutlineLink[]>();
    for (const link of data?.outlineLinks ?? []) {
      const links = result.get(link.requirementKey) ?? [];
      links.push(link);
      result.set(link.requirementKey, links);
    }
    return result;
  }, [data?.outlineLinks]);

  const statusLabel = (value: ProjectRequirementStatus) => {
    switch (value) {
      case "in_review":
        return t(($) => $.requirements.status_in_review);
      case "approved":
        return t(($) => $.requirements.status_approved);
      case "frozen":
        return t(($) => $.requirements.status_frozen);
      case "changed":
        return t(($) => $.requirements.status_changed);
      case "retired":
        return t(($) => $.requirements.status_retired);
      case "draft":
        return t(($) => $.requirements.status_draft);
    }
  };

  const actionLabel = (action: ProjectRequirementAction) => {
    switch (action) {
      case "create":
        return t(($) => $.requirements.action_create);
      case "save_draft":
        return t(($) => $.requirements.action_save_draft);
      case "submit_review":
        return t(($) => $.requirements.action_submit_review);
      case "withdraw_review":
        return t(($) => $.requirements.action_withdraw_review);
      case "approve":
        return t(($) => $.requirements.action_approve);
      case "freeze":
        return t(($) => $.requirements.action_freeze);
      case "material_change":
        return t(($) => $.requirements.action_material_change);
      case "retire":
        return t(($) => $.requirements.action_retire);
      case "link_issue":
        return t(($) => $.requirements.action_link_issue);
      case "unlink_issue":
        return t(($) => $.requirements.action_unlink_issue);
      case "link_outline":
        return t(($) => $.requirements.action_link_outline);
      case "unlink_outline":
        return t(($) => $.requirements.action_unlink_outline);
      case "issue_deleted":
        return t(($) => $.requirements.action_issue_deleted);
      case "legacy_import":
        return t(($) => $.requirements.action_legacy_import);
    }
  };

  const sectionLabel = (key: ContentListKey) => {
    switch (key) {
      case "goals":
        return t(($) => $.requirements.goals);
      case "inScope":
        return t(($) => $.requirements.in_scope);
      case "outOfScope":
        return t(($) => $.requirements.out_of_scope);
      case "constraints":
        return t(($) => $.requirements.constraints);
      case "acceptanceCriteria":
        return t(($) => $.requirements.acceptance_criteria);
      case "dependencies":
        return t(($) => $.requirements.dependencies);
    }
  };

  const showMutationError = (error: unknown) => {
    const problem = getProjectRequirementError(error);
    switch (problem.code) {
      case "revision_conflict":
        toast.error(
          t(($) => $.requirements.stale_revision, {
            revision: problem.currentRevision ?? "?",
          })
        );
        return;
      case "independent_approval_required":
        toast.error(t(($) => $.requirements.independent_approval_required));
        return;
      case "permission_denied":
        toast.error(t(($) => $.requirements.permission_denied));
        return;
      case "invalid_transition":
        toast.error(t(($) => $.requirements.invalid_transition));
        return;
      case "unknown":
        toast.error(t(($) => $.requirements.mutation_failed));
    }
  };

  const auditActorLabel = (actorId: string) =>
    getActorName("member", actorId) || t(($) => $.requirements.unknown_member);
  const auditTimeLabel = (value: string | null) =>
    formatRequirementAuditTime(value) ?? t(($) => $.requirements.unknown_time);
  const updateList = (key: ContentListKey, items: ProjectRequirementItem[]) =>
    setContent((previous) => ({ ...previous, [key]: items }));

  if (isLoading) {
    return (
      <div className="space-y-4 p-6">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-40 w-full" />
      </div>
    );
  }

  const saveAllowed =
    !baseline || status === "draft" || status === "changed" || status === "frozen";
  const contentDisabled =
    isBusy ||
    !access.canEdit ||
    !saveAllowed ||
    (status === "frozen" && !materialChange);
  const terminal = status === "retired";
  const issueLinkAllowed = !!baseline && access.canEdit && !terminal;
  const outlineLinkAllowed = !!baseline && access.canManageOutline && !terminal;

  const saveDraft = () =>
    save.mutate(
      {
        expectedRevision: currentRevision,
        content,
        changeSummary,
        materialChange: status === "frozen" && materialChange,
      },
      {
        onSuccess: () => toast.success(t(($) => $.requirements.saved)),
        onError: showMutationError,
      }
    );
  const submitReview = () =>
    submit.mutate(
      { expectedRevision: currentRevision },
      {
        onSuccess: () => toast.success(t(($) => $.requirements.submitted)),
        onError: showMutationError,
      }
    );
  const approveReview = () =>
    approve.mutate(
      { expectedRevision: currentRevision },
      {
        onSuccess: () => toast.success(t(($) => $.requirements.approved)),
        onError: showMutationError,
      }
    );
  const withdrawReview = () =>
    withdraw.mutate(
      { expectedRevision: currentRevision },
      {
        onSuccess: () => toast.success(t(($) => $.requirements.withdrawn)),
        onError: showMutationError,
      }
    );
  const freezeBaseline = () =>
    freeze.mutate(
      { expectedRevision: currentRevision },
      {
        onSuccess: () => toast.success(t(($) => $.requirements.frozen)),
        onError: showMutationError,
      }
    );
  const retireBaseline = () =>
    retire.mutate(
      { expectedRevision: currentRevision },
      {
        onSuccess: () => toast.success(t(($) => $.requirements.retired)),
        onError: showMutationError,
      }
    );
  const handleLinkIssue = (requirementKey: string, issueId: string) =>
    linkIssue.mutate(
      { requirementKey, issueId, expectedRevision: currentRevision },
      { onError: showMutationError }
    );
  const handleUnlinkIssue = (requirementKey: string, issueId: string) =>
    unlinkIssue.mutate(
      { requirementKey, issueId, expectedRevision: currentRevision },
      { onError: showMutationError }
    );
  const handleLinkOutline = (requirementKey: string, nodeId: string) =>
    linkOutline.mutate(
      { requirementKey, nodeId, expectedRevision: currentRevision },
      { onError: showMutationError }
    );
  const handleUnlinkOutline = (requirementKey: string, nodeId: string) =>
    unlinkOutline.mutate(
      { requirementKey, nodeId, expectedRevision: currentRevision },
      { onError: showMutationError }
    );
  const handleCreateOutline = () =>
    createOutline.mutate(
      { expectedRevision: outline?.revision ?? 0, title: outlineTitle.trim() },
      {
        onSuccess: () => setOutlineTitle(""),
        onError: showMutationError,
      }
    );

  return (
    <div className="mx-auto w-full max-w-4xl overflow-y-auto px-6 py-6">
      <div className="mb-6 flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-lg font-semibold">{t(($) => $.requirements.title)}</h2>
          <p className="text-sm text-muted-foreground">
            {t(($) => $.requirements.subtitle)}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2 text-sm">
          <span className="rounded-full bg-muted px-2 py-1">
            {status ? statusLabel(status) : t(($) => $.requirements.status_draft)}
          </span>
          {baseline && (
            <span className="text-muted-foreground">
              {t(($) => $.requirements.current_version, {
                revision: baseline.currentRevision,
              })}
            </span>
          )}
        </div>
      </div>

      {status === "in_review" && (
        <p className="mb-4 rounded-md border bg-muted/40 px-3 py-2 text-sm">
          {t(($) => $.requirements.in_review_hint)}
        </p>
      )}
      {status === "approved" && (
        <p className="mb-4 rounded-md border bg-muted/40 px-3 py-2 text-sm">
          {t(($) => $.requirements.approved_hint)}
        </p>
      )}
      {status === "retired" && (
        <p className="mb-4 rounded-md border bg-muted/40 px-3 py-2 text-sm">
          {t(($) => $.requirements.retired_hint)}
        </p>
      )}

      {baseline?.effectiveRevision && data?.effectiveContent && (
        <EffectiveBaseline
          revision={baseline.effectiveRevision}
          content={data.effectiveContent}
        />
      )}

      <MinimalOutlineRoots
        revision={outline?.revision ?? 0}
        canCreate={access.canManageOutline && !terminal}
        busy={isBusy}
        title={outlineTitle}
        onTitleChange={setOutlineTitle}
        onCreate={handleCreateOutline}
      />

      <div className="space-y-5">
        <label className="block space-y-2">
          <span className="text-sm font-medium">
            {t(($) => $.requirements.problem_statement)}
          </span>
          <textarea
            disabled={contentDisabled}
            value={content.problemStatement}
            onChange={(event) =>
              setContent((previous) => ({
                ...previous,
                problemStatement: event.target.value,
              }))
            }
            className="min-h-24 w-full rounded-md border bg-background p-3 text-sm"
          />
        </label>

        {SECTIONS.map((section) => (
          <div key={section} className="space-y-2">
            <span className="text-sm font-medium">{sectionLabel(section)}</span>
            <div className="space-y-2">
              {content[section].map((item) => {
                const savedItem = savedItems.get(item.key);
                const canTrack =
                  savedItem?.section === section && savedItem.text === item.text;
                return (
                  <div key={item.key} className="rounded-md border p-2">
                    <div className="flex gap-2">
                      <input
                        disabled={contentDisabled}
                        aria-label={sectionLabel(section)}
                        value={item.text}
                        onChange={(event) =>
                          updateList(
                            section,
                            updateRequirementItem(
                              content[section],
                              item.key,
                              event.target.value
                            )
                          )
                        }
                        className="min-w-0 flex-1 rounded-md border bg-background px-3 py-2 text-sm"
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        disabled={contentDisabled}
                        onClick={() =>
                          updateList(
                            section,
                            removeRequirementItem(content[section], item.key)
                          )
                        }
                      >
                        {t(($) => $.requirements.remove_item)}
                      </Button>
                    </div>
                    {TRACEABLE_SECTIONS.has(section) &&
                      (canTrack ? (
                        <RequirementTrackingControls
                          requirementKey={item.key}
                          issueLinks={issueLinksByKey.get(item.key) ?? []}
                          outlineLinks={outlineLinksByKey.get(item.key) ?? []}
                          availableIssues={projectIssues?.issues ?? []}
                          outlineNodes={outline?.nodes ?? []}
                          canLinkIssues={issueLinkAllowed}
                          canLinkOutline={outlineLinkAllowed}
                          busy={isBusy}
                          onLinkIssue={handleLinkIssue}
                          onUnlinkIssue={handleUnlinkIssue}
                          onLinkOutline={handleLinkOutline}
                          onUnlinkOutline={handleUnlinkOutline}
                        />
                      ) : (
                        <p className="mt-2 text-sm text-muted-foreground">
                          {t(($) => $.requirements.save_before_tracking)}
                        </p>
                      ))}
                  </div>
                );
              })}
            </div>
            <Button
              type="button"
              variant="secondary"
              size="sm"
              disabled={contentDisabled}
              onClick={() => updateList(section, appendRequirementItem(content[section]))}
            >
              {t(($) => $.requirements.add_item)}
            </Button>
          </div>
        ))}

        {status === "frozen" && access.canEdit && (
          <label className="flex items-center gap-2 text-sm font-medium">
            <input
              type="checkbox"
              checked={materialChange}
              disabled={isBusy}
              onChange={(event) => setMaterialChange(event.target.checked)}
            />
            {t(($) => $.requirements.material_change)}
          </label>
        )}
        {saveAllowed && (
          <label className="block space-y-2">
            <span className="text-sm font-medium">
              {t(($) => $.requirements.change_summary)}
            </span>
            <input
              aria-label={t(($) => $.requirements.change_summary)}
              disabled={isBusy || !access.canEdit || (status === "frozen" && !materialChange)}
              value={changeSummary}
              onChange={(event) => setChangeSummary(event.target.value)}
              className="w-full rounded-md border bg-background px-3 py-2 text-sm"
            />
          </label>
        )}
      </div>

      <div className="mt-5 flex flex-wrap gap-2">
        {saveAllowed && access.canEdit && (
          <Button
            disabled={
              isBusy ||
              !changeSummary.trim() ||
              (status === "frozen" && !materialChange)
            }
            onClick={saveDraft}
          >
            {status === "frozen"
              ? t(($) => $.requirements.save_material_change)
              : t(($) => $.requirements.save_draft)}
          </Button>
        )}
        {(status === "draft" || status === "changed") && access.canEdit && baseline && (
          <Button variant="secondary" disabled={isBusy} onClick={submitReview}>
            {t(($) => $.requirements.submit_review)}
          </Button>
        )}
        {status === "in_review" && access.canEdit && (
          <Button variant="secondary" disabled={isBusy} onClick={withdrawReview}>
            {t(($) => $.requirements.withdraw)}
          </Button>
        )}
        {status === "in_review" && access.canApprove && (
          <Button disabled={isBusy} onClick={approveReview}>
            {t(($) => $.requirements.approve)}
          </Button>
        )}
        {status === "approved" && access.canApprove && (
          <Button disabled={isBusy} onClick={freezeBaseline}>
            {t(($) => $.requirements.freeze)}
          </Button>
        )}
        {baseline && !terminal && access.canApprove && (
          <Button variant="destructive" disabled={isBusy} onClick={retireBaseline}>
            {t(($) => $.requirements.retire)}
          </Button>
        )}
      </div>

      <div className="mt-10 border-t pt-5">
        <h3 className="mb-3 text-sm font-medium">
          {t(($) => $.requirements.history)}
        </h3>
        {history.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            {t(($) => $.requirements.empty_history)}
          </p>
        ) : (
          <ol className="space-y-2">
            {history.map((item) => (
              <li key={item.revision} className="rounded-md border px-3 py-2 text-sm">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span>
                    {t(($) => $.requirements.revision, { revision: item.revision })} ·{" "}
                    {statusLabel(item.state)} · {actionLabel(item.action)}
                  </span>
                  <span className="text-muted-foreground">
                    {item.changeSummary || t(($) => $.requirements.no_change_summary)}
                  </span>
                </div>
                {item.submittedBy && (
                  <p className="mt-1 text-xs text-muted-foreground">
                    {t(($) => $.requirements.submitted_audit, {
                      actor: auditActorLabel(item.submittedBy),
                      time: auditTimeLabel(item.submittedAt),
                    })}
                  </p>
                )}
                {item.approvedBy && (
                  <p className="mt-1 text-xs text-muted-foreground">
                    {t(($) => $.requirements.approved_audit, {
                      actor: auditActorLabel(item.approvedBy),
                      time: auditTimeLabel(item.approvedAt),
                    })}
                  </p>
                )}
                {item.frozenBy && (
                  <p className="mt-1 text-xs text-muted-foreground">
                    {t(($) => $.requirements.frozen_audit, {
                      actor: auditActorLabel(item.frozenBy),
                      time: auditTimeLabel(item.frozenAt),
                    })}
                  </p>
                )}
              </li>
            ))}
          </ol>
        )}
      </div>
    </div>
  );
}
