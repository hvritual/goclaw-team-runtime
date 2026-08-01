"use client";

import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import type { ProjectRequirementContent, ProjectRequirementItem } from "@multica/core/types";
import {
  projectRequirementBaselineOptions,
  useApproveProjectRequirement,
  useSaveProjectRequirementDraft,
  useSubmitProjectRequirementReview,
  useWithdrawProjectRequirementReview,
} from "@multica/core/project-requirements";
import { Button } from "@multica/ui/components/ui/button";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { useWorkspaceId } from "@multica/core/hooks";
import { useActorName } from "@multica/core/workspace/hooks";
import { useT } from "../../i18n";

const EMPTY_CONTENT: ProjectRequirementContent = {
  problemStatement: "", goals: [], inScope: [], outOfScope: [], constraints: [], acceptanceCriteria: [], dependencies: [],
};

type ContentListKey = Exclude<keyof ProjectRequirementContent, "problemStatement">;

const sections: ContentListKey[] = [
  "goals", "inScope", "outOfScope", "constraints", "acceptanceCriteria", "dependencies",
];

function itemKey(): string {
  return globalThis.crypto?.randomUUID?.() ?? `item-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

export function updateRequirementItem(items: ProjectRequirementItem[], key: string, text: string): ProjectRequirementItem[] {
  return items.map((item) => item.key === key ? { ...item, text } : item);
}

export function removeRequirementItem(items: ProjectRequirementItem[], key: string): ProjectRequirementItem[] {
  return items.filter((item) => item.key !== key);
}

export function appendRequirementItem(items: ProjectRequirementItem[]): ProjectRequirementItem[] {
  let key = itemKey();
  while (items.some((item) => item.key === key)) key = itemKey();
  return [...items, { key, text: "" }];
}

export function formatRequirementAuditTime(value: string | null): string | null {
  if (!value) return null;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return null;
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(date);
}

export function ProjectRequirementBaseline({ projectId, canApprove }: { projectId: string; canApprove: boolean }) {
  const { t } = useT("projects");
  const wsId = useWorkspaceId();
  const { getActorName } = useActorName();
  const { data, isLoading, dataUpdatedAt } = useQuery(projectRequirementBaselineOptions(wsId, projectId));
  const baseline = data?.baseline;
  const [content, setContent] = useState<ProjectRequirementContent>(EMPTY_CONTENT);
  const [changeSummary, setChangeSummary] = useState("");
  const save = useSaveProjectRequirementDraft(wsId, projectId);
  const submit = useSubmitProjectRequirementReview(wsId, projectId);
  const approve = useApproveProjectRequirement(wsId, projectId);
  const withdraw = useWithdrawProjectRequirementReview(wsId, projectId);

  useEffect(() => { setContent(data?.currentContent ?? EMPTY_CONTENT); }, [dataUpdatedAt]);
  const revision = baseline?.currentRevision ?? 0;
  const isReview = baseline?.status === "in_review";
  const isApproved = baseline?.status === "approved";
  const isBusy = save.isPending || submit.isPending || approve.isPending || withdraw.isPending;
  const history = useMemo(() => data?.history ?? [], [data?.history]);
  const statusLabel = (status: string) => {
    switch (status) {
      case "in_review": return t(($) => $.requirements.status_in_review);
      case "approved": return t(($) => $.requirements.status_approved);
      case "superseded": return t(($) => $.requirements.status_superseded);
      default: return t(($) => $.requirements.status_draft);
    }
  };
  const sectionLabel = (key: ContentListKey) => {
    switch (key) {
      case "goals": return t(($) => $.requirements.goals);
      case "inScope": return t(($) => $.requirements.in_scope);
      case "outOfScope": return t(($) => $.requirements.out_of_scope);
      case "constraints": return t(($) => $.requirements.constraints);
      case "acceptanceCriteria": return t(($) => $.requirements.acceptance_criteria);
      case "dependencies": return t(($) => $.requirements.dependencies);
    }
  };
  const auditActorLabel = (actorId: string) => getActorName("member", actorId) || t(($) => $.requirements.unknown_member);
  const auditTimeLabel = (value: string | null) => formatRequirementAuditTime(value) ?? t(($) => $.requirements.unknown_time);
  const updateList = (key: ContentListKey, items: ProjectRequirementItem[]) => setContent((previous) => ({ ...previous, [key]: items }));

  if (isLoading) return <div className="space-y-4 p-6"><Skeleton className="h-8 w-48" /><Skeleton className="h-40 w-full" /></div>;

  const saveDraft = () => save.mutate({ expectedRevision: revision, content, changeSummary }, { onSuccess: () => toast.success(t(($) => $.requirements.saved)), onError: () => toast.error(t(($) => $.requirements.save_failed)) });
  const submitReview = () => submit.mutate({ expectedRevision: revision }, { onSuccess: () => toast.success(t(($) => $.requirements.submitted)), onError: () => toast.error(t(($) => $.requirements.transition_failed)) });
  const approveReview = () => approve.mutate({ expectedRevision: revision }, { onSuccess: () => toast.success(t(($) => $.requirements.approved)), onError: () => toast.error(t(($) => $.requirements.transition_failed)) });
  const withdrawReview = () => withdraw.mutate({ expectedRevision: revision }, { onSuccess: () => toast.success(t(($) => $.requirements.withdrawn)), onError: () => toast.error(t(($) => $.requirements.transition_failed)) });

  return (
    <div className="mx-auto w-full max-w-4xl overflow-y-auto px-6 py-6">
      <div className="mb-6 flex flex-wrap items-start justify-between gap-3">
        <div><h2 className="text-lg font-semibold">{t(($) => $.requirements.title)}</h2><p className="text-sm text-muted-foreground">{t(($) => $.requirements.subtitle)}</p></div>
        <div className="flex flex-wrap items-center gap-2 text-sm"><span className="rounded-full bg-muted px-2 py-1">{baseline ? statusLabel(baseline.status) : t(($) => $.requirements.status_draft)}</span>{baseline && <span className="text-muted-foreground">{t(($) => $.requirements.current_version, { revision: baseline.currentRevision })}</span>}{baseline?.approvedRevision && <span className="text-muted-foreground">{t(($) => $.requirements.effective_version, { revision: baseline.approvedRevision })}</span>}</div>
      </div>
      {isReview && <p className="mb-4 rounded-md border border-border bg-muted/40 px-3 py-2 text-sm">{t(($) => $.requirements.in_review_hint)}</p>}
      {isApproved && <p className="mb-4 rounded-md border border-border bg-muted/40 px-3 py-2 text-sm">{t(($) => $.requirements.approved_hint)}</p>}
      <div className="space-y-5">
        <label className="block space-y-2"><span className="text-sm font-medium">{t(($) => $.requirements.problem_statement)}</span><textarea disabled={isReview || isBusy} value={content.problemStatement} onChange={(event) => setContent({ ...content, problemStatement: event.target.value })} className="min-h-24 w-full rounded-md border bg-background p-3 text-sm" /></label>
        {sections.map((section) => <div key={section} className="space-y-2"><span className="text-sm font-medium">{sectionLabel(section)}</span><div className="space-y-2">{content[section].map((item) => <div key={item.key} className="flex gap-2"><input disabled={isReview || isBusy} aria-label={sectionLabel(section)} value={item.text} onChange={(event) => updateList(section, updateRequirementItem(content[section], item.key, event.target.value))} className="min-w-0 flex-1 rounded-md border bg-background px-3 py-2 text-sm" /><Button type="button" variant="ghost" size="sm" disabled={isReview || isBusy} onClick={() => updateList(section, removeRequirementItem(content[section], item.key))}>{t(($) => $.requirements.remove_item)}</Button></div>)}</div><Button type="button" variant="secondary" size="sm" disabled={isReview || isBusy} onClick={() => updateList(section, appendRequirementItem(content[section]))}>{t(($) => $.requirements.add_item)}</Button></div>)}
        <label className="block space-y-2"><span className="text-sm font-medium">{t(($) => $.requirements.change_summary)}</span><input disabled={isReview || isBusy} value={changeSummary} onChange={(event) => setChangeSummary(event.target.value)} className="w-full rounded-md border bg-background px-3 py-2 text-sm" /></label>
      </div>
      <div className="mt-5 flex flex-wrap gap-2">
        {!isReview && <Button disabled={isBusy} onClick={saveDraft}>{t(($) => $.requirements.save_draft)}</Button>}
        {!isReview && <Button variant="secondary" disabled={isBusy || !baseline} onClick={submitReview}>{t(($) => $.requirements.submit_review)}</Button>}
        {isReview && canApprove && <><Button disabled={isBusy} onClick={approveReview}>{t(($) => $.requirements.approve)}</Button><Button variant="secondary" disabled={isBusy} onClick={withdrawReview}>{t(($) => $.requirements.withdraw)}</Button></>}
      </div>
      <div className="mt-10 border-t pt-5"><h3 className="mb-3 text-sm font-medium">{t(($) => $.requirements.history)}</h3>{history.length === 0 ? <p className="text-sm text-muted-foreground">{t(($) => $.requirements.empty_history)}</p> : <ol className="space-y-2">{history.map((item) => <li key={item.revision} className="flex flex-wrap items-center justify-between gap-2 rounded-md border px-3 py-2 text-sm"><span>{t(($) => $.requirements.revision, { revision: item.revision })} · {statusLabel(item.state)}</span><span className="text-muted-foreground">{item.changeSummary || t(($) => $.requirements.no_change_summary)}</span>{item.submittedBy && <span className="w-full text-xs text-muted-foreground">{t(($) => $.requirements.submitted_audit, { actor: auditActorLabel(item.submittedBy), time: auditTimeLabel(item.submittedAt) })}</span>}{item.approvedBy && <span className="w-full text-xs text-muted-foreground">{t(($) => $.requirements.approved_audit, { actor: auditActorLabel(item.approvedBy), time: auditTimeLabel(item.approvedAt) })}</span>}</li>)}</ol>}</div>
    </div>
  );
}
