"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  knowledgeCandidateListOptions,
  knowledgeDetailOptions,
  knowledgeListOptions,
  useProposeKnowledge,
  useReviewKnowledge,
} from "@multica/core/knowledge";
import { useWorkspaceId } from "@multica/core/hooks";
import { useCurrentMember } from "@multica/core/permissions";
import { projectListOptions } from "@multica/core/projects/queries";
import type {
  KnowledgeCandidate,
  KnowledgeEntry,
  KnowledgeKind,
  KnowledgeReviewAction,
} from "@multica/core/types";
import { Badge } from "@multica/ui/components/ui/badge";
import { Button } from "@multica/ui/components/ui/button";
import { Input } from "@multica/ui/components/ui/input";
import { Textarea } from "@multica/ui/components/ui/textarea";
import {
  BookOpenText,
  Check,
  Eye,
  FileText,
  History,
  Loader2,
  Plus,
  Search,
  ShieldAlert,
  X,
} from "lucide-react";
import { useT } from "../i18n";

const KNOWLEDGE_KINDS: KnowledgeKind[] = [
  "goal",
  "decision",
  "constraint",
  "requirement",
  "procedure",
  "lesson",
  "reference",
];

export function KnowledgePage() {
  const { t } = useT("knowledge");
  const workspaceId = useWorkspaceId();
  const { role, userId } = useCurrentMember(workspaceId);
  const projectsQuery = useQuery(projectListOptions(workspaceId));
  const canReview =
    role === "owner" ||
    role === "admin" ||
    projectsQuery.data?.some(
      (project) =>
        project.lead_type === "member" && project.lead_id === userId,
    ) === true;
  const [query, setQuery] = useState("");
  const [section, setSection] = useState<"published" | "review">("published");
  const [showProposal, setShowProposal] = useState(false);
  const [proposalTarget, setProposalTarget] = useState<KnowledgeEntry | null>(
    null,
  );
  const [selectedKnowledgeId, setSelectedKnowledgeId] = useState("");
  const [kind, setKind] = useState<KnowledgeKind>("lesson");
  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");
  const [reason, setReason] = useState("");
  const [reviewNotes, setReviewNotes] = useState<Record<string, string>>({});
  const listQuery = useQuery(knowledgeListOptions(workspaceId, query));
  const candidateQuery = useQuery(
    knowledgeCandidateListOptions(workspaceId, canReview),
  );
  const propose = useProposeKnowledge(workspaceId);
  const review = useReviewKnowledge(workspaceId);

  const openNewProposal = () => {
    setProposalTarget(null);
    setTitle("");
    setContent("");
    setReason("");
    setShowProposal(true);
  };

  const openRevisionProposal = (entry: KnowledgeEntry) => {
    const current =
      entry.revisions.find(
        (revision) => revision.number === entry.currentRevision,
      ) ?? entry.revisions.at(-1);
    setProposalTarget(entry);
    setKind(entry.kind);
    setTitle(current?.title ?? "");
    setContent(current?.content ?? "");
    setReason("");
    setShowProposal(true);
  };

  const closeProposal = () => {
    setShowProposal(false);
    setProposalTarget(null);
  };

  const submitProposal = () => {
    if (!title.trim() || !content.trim() || !reason.trim()) return;
    propose.mutate(
      {
        ...(proposalTarget
          ? {
              knowledgeId: proposalTarget.id,
              projectId: proposalTarget.projectId ?? undefined,
            }
          : {}),
        kind,
        title: title.trim(),
        content: content.trim(),
        reason: reason.trim(),
      },
      {
        onSuccess: () => {
          setTitle("");
          setContent("");
          setReason("");
          closeProposal();
        },
      },
    );
  };

  const reviewCandidate = (
    candidate: KnowledgeCandidate,
    action: KnowledgeReviewAction,
  ) => {
    const rationale = reviewNotes[candidate.id]?.trim();
    if (!rationale) return;
    review.mutate({
      candidateId: candidate.id,
      action,
      expectedRevision: candidate.revision,
      rationale,
    });
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <header className="flex h-12 shrink-0 items-center gap-2 border-b px-4">
        <BookOpenText className="size-4 text-muted-foreground" />
        <h1 className="text-sm font-semibold">
          {t(($) => $.header.title)}
        </h1>
        <div className="ml-auto">
          <Button size="sm" onClick={openNewProposal}>
            <Plus className="size-4" />
            {t(($) => $.header.propose)}
          </Button>
        </div>
      </header>

      <main className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-4 overflow-y-auto p-4 sm:p-6">
        {showProposal ? (
          <section className="space-y-3 rounded-xl border bg-card p-4">
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-semibold">
                {proposalTarget
                  ? t(($) => $.proposal.revision_title)
                  : t(($) => $.proposal.title)}
              </h2>
              <Button
                size="icon-sm"
                variant="ghost"
                onClick={closeProposal}
                aria-label={t(($) => $.proposal.cancel)}
              >
                <X className="size-4" />
              </Button>
            </div>
            <div className="grid gap-3 sm:grid-cols-[180px_1fr]">
              <select
                value={kind}
                onChange={(event) => setKind(event.target.value as KnowledgeKind)}
                className="h-9 rounded-md border bg-background px-3 text-sm"
              >
                {KNOWLEDGE_KINDS.map((value) => (
                  <option key={value} value={value}>
                    {t(($) => $.kinds[value])}
                  </option>
                ))}
              </select>
              <Input
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                placeholder={t(($) => $.proposal.title_placeholder)}
              />
            </div>
            <Textarea
              value={content}
              onChange={(event) => setContent(event.target.value)}
              placeholder={t(($) => $.proposal.content_placeholder)}
              className="min-h-28"
            />
            <Textarea
              value={reason}
              onChange={(event) => setReason(event.target.value)}
              placeholder={t(($) => $.proposal.reason_placeholder)}
              className="min-h-20"
            />
            <div className="flex justify-end">
              <Button
                onClick={submitProposal}
                disabled={
                  propose.isPending ||
                  !title.trim() ||
                  !content.trim() ||
                  !reason.trim()
                }
              >
                {propose.isPending ? (
                  <Loader2 className="size-4 animate-spin" />
                ) : (
                  <Plus className="size-4" />
                )}
                {t(($) => $.proposal.submit)}
              </Button>
            </div>
          </section>
        ) : null}

        <div className="flex flex-wrap items-center gap-2">
          <Button
            size="sm"
            variant={section === "published" ? "secondary" : "ghost"}
            onClick={() => setSection("published")}
          >
            {t(($) => $.tabs.published)}
          </Button>
          {canReview ? (
            <Button
              size="sm"
              variant={section === "review" ? "secondary" : "ghost"}
              onClick={() => setSection("review")}
            >
              {t(($) => $.tabs.review_queue)}
              {(candidateQuery.data?.total ?? 0) > 0 ? (
                <Badge variant="secondary">
                  {candidateQuery.data?.total ?? 0}
                </Badge>
              ) : null}
            </Button>
          ) : null}
        </div>

        {section === "published" ? (
          <>
            <div className="relative">
              <Search className="pointer-events-none absolute top-2.5 left-3 size-4 text-muted-foreground" />
              <Input
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder={t(($) => $.search.placeholder)}
                className="pl-9"
              />
            </div>
            {selectedKnowledgeId ? (
              <KnowledgeDetails
                workspaceId={workspaceId}
                knowledgeId={selectedKnowledgeId}
                onClose={() => setSelectedKnowledgeId("")}
                onProposeRevision={openRevisionProposal}
              />
            ) : null}
            {listQuery.isLoading ? (
              <LoadingState label={t(($) => $.states.loading)} />
            ) : listQuery.isError ? (
              <ErrorState label={t(($) => $.states.load_failed)} />
            ) : (listQuery.data?.entries.length ?? 0) === 0 ? (
              <EmptyState label={t(($) => $.states.empty)} />
            ) : (
              <div className="grid gap-3">
                {listQuery.data?.entries.map((entry) => {
                  const revision =
                    entry.revisions.find(
                      (candidate) =>
                        candidate.number === entry.currentRevision,
                    ) ?? entry.revisions.at(-1);
                  return (
                    <article
                      key={entry.id}
                      className="space-y-3 rounded-xl border bg-card p-4"
                    >
                      <div className="flex items-start gap-3">
                        <Badge variant="secondary">
                          {t(($) => $.kinds[entry.kind])}
                        </Badge>
                        <div className="min-w-0 flex-1">
                          <h2 className="font-medium">
                            {revision?.title ?? entry.id}
                          </h2>
                          <p className="mt-1 whitespace-pre-wrap text-sm text-muted-foreground">
                            {revision?.content ?? ""}
                          </p>
                        </div>
                      </div>
                      <p className="text-xs text-muted-foreground">
                        {t(($) => $.source.count, {
                          count: revision?.sourceRefs.length ?? 0,
                        })}
                      </p>
                      <div className="flex flex-wrap justify-end gap-2">
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => setSelectedKnowledgeId(entry.id)}
                        >
                          <Eye className="size-4" />
                          {t(($) => $.detail.open)}
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => openRevisionProposal(entry)}
                        >
                          <History className="size-4" />
                          {t(($) => $.detail.propose_revision)}
                        </Button>
                      </div>
                    </article>
                  );
                })}
              </div>
            )}
          </>
        ) : canReview ? (
          candidateQuery.isLoading ? (
            <LoadingState label={t(($) => $.states.loading)} />
          ) : candidateQuery.isError ? (
            <ErrorState label={t(($) => $.states.load_failed)} />
          ) : (candidateQuery.data?.candidates.length ?? 0) === 0 ? (
            <EmptyState label={t(($) => $.states.empty_candidates)} />
          ) : (
            <div className="grid gap-3">
              {candidateQuery.data?.candidates.map((candidate) => (
                <article
                  key={candidate.id}
                  className="space-y-3 rounded-xl border bg-card p-4"
                >
                  <div className="flex items-center gap-2">
                    <Badge variant="secondary">
                      {t(($) => $.kinds[candidate.kind])}
                    </Badge>
                    <h2 className="font-medium">{candidate.title}</h2>
                  </div>
                  <p className="whitespace-pre-wrap text-sm text-muted-foreground">
                    {candidate.content}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {candidate.reason}
                  </p>
                  <Textarea
                    value={reviewNotes[candidate.id] ?? ""}
                    onChange={(event) =>
                      setReviewNotes((current) => ({
                        ...current,
                        [candidate.id]: event.target.value,
                      }))
                    }
                    placeholder={t(($) => $.review.rationale_placeholder)}
                    className="min-h-20"
                  />
                  <div className="flex flex-wrap justify-end gap-2">
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={
                        review.isPending ||
                        !reviewNotes[candidate.id]?.trim()
                      }
                      onClick={() =>
                        reviewCandidate(candidate, "quarantine")
                      }
                    >
                      <ShieldAlert className="size-4" />
                      {t(($) => $.review.quarantine)}
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={
                        review.isPending ||
                        !reviewNotes[candidate.id]?.trim()
                      }
                      onClick={() => reviewCandidate(candidate, "reject")}
                    >
                      <X className="size-4" />
                      {t(($) => $.review.reject)}
                    </Button>
                    <Button
                      size="sm"
                      disabled={
                        review.isPending ||
                        !reviewNotes[candidate.id]?.trim()
                      }
                      onClick={() => reviewCandidate(candidate, "approve")}
                    >
                      <Check className="size-4" />
                      {t(($) => $.review.approve)}
                    </Button>
                  </div>
                </article>
              ))}
            </div>
          )
        ) : null}
      </main>
    </div>
  );
}

function KnowledgeDetails({
  workspaceId,
  knowledgeId,
  onClose,
  onProposeRevision,
}: {
  workspaceId: string;
  knowledgeId: string;
  onClose: () => void;
  onProposeRevision: (entry: KnowledgeEntry) => void;
}) {
  const { t } = useT("knowledge");
  const detailQuery = useQuery(
    knowledgeDetailOptions(workspaceId, knowledgeId),
  );

  if (detailQuery.isLoading) {
    return <LoadingState label={t(($) => $.detail.loading)} />;
  }
  if (detailQuery.isError || !detailQuery.data?.id) {
    return <ErrorState label={t(($) => $.detail.load_failed)} />;
  }

  const entry = detailQuery.data;
  const revisions = [...entry.revisions].sort(
    (left, right) => right.number - left.number,
  );
  return (
    <section className="space-y-4 rounded-xl border bg-card p-4">
      <div className="flex flex-wrap items-center gap-2">
        <History className="size-4 text-muted-foreground" />
        <h2 className="text-sm font-semibold">{t(($) => $.detail.title)}</h2>
        <Badge variant="secondary">
          {t(($) => $.detail.revision_count, { count: revisions.length })}
        </Badge>
        <div className="ml-auto flex gap-2">
          <Button
            size="sm"
            variant="outline"
            onClick={() => onProposeRevision(entry)}
          >
            <History className="size-4" />
            {t(($) => $.detail.propose_revision)}
          </Button>
          <Button
            size="icon-sm"
            variant="ghost"
            onClick={onClose}
            aria-label={t(($) => $.detail.close)}
          >
            <X className="size-4" />
          </Button>
        </div>
      </div>
      <div className="grid gap-3">
        {revisions.map((revision) => (
          <article
            key={revision.number}
            className="space-y-2 rounded-lg border bg-background p-3"
          >
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant={revision.number === entry.currentRevision ? "default" : "outline"}>
                {t(($) => $.detail.revision, { number: revision.number })}
              </Badge>
              {revision.supersedesRevision > 0 ? (
                <span className="text-xs text-muted-foreground">
                  {t(($) => $.detail.supersedes, {
                    number: revision.supersedesRevision,
                  })}
                </span>
              ) : null}
              <h3 className="font-medium">{revision.title}</h3>
            </div>
            <p className="whitespace-pre-wrap text-sm text-muted-foreground">
              {revision.content}
            </p>
            {revision.sourceRefs.length > 0 ? (
              <div className="space-y-1 border-t pt-2">
                <p className="flex items-center gap-1 text-xs font-medium">
                  <FileText className="size-3.5" />
                  {t(($) => $.detail.sources)}
                </p>
                {revision.sourceRefs.map((source) => (
                  <p
                    key={`${source.type}:${source.id}:${source.revision}`}
                    className="break-all text-xs text-muted-foreground"
                  >
                    {source.type} · {source.id}
                    {source.revision ? ` · ${source.revision}` : ""}
                    {source.uri ? ` · ${source.uri}` : ""}
                  </p>
                ))}
              </div>
            ) : null}
          </article>
        ))}
      </div>
    </section>
  );
}

function LoadingState({ label }: { label: string }) {
  return (
    <div className="flex flex-1 items-center justify-center gap-2 p-10 text-sm text-muted-foreground">
      <Loader2 className="size-5 animate-spin" />
      {label}
    </div>
  );
}

function EmptyState({ label }: { label: string }) {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-2 rounded-xl border border-dashed p-10 text-sm text-muted-foreground">
      <BookOpenText className="size-8 opacity-40" />
      {label}
    </div>
  );
}

function ErrorState({ label }: { label: string }) {
  return (
    <div className="flex flex-1 items-center justify-center rounded-xl border border-destructive/30 p-10 text-sm text-destructive">
      {label}
    </div>
  );
}
