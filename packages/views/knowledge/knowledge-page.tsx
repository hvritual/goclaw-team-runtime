"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  knowledgeCandidateListOptions,
  knowledgeListOptions,
  useProposeKnowledge,
  useReviewKnowledge,
} from "@multica/core/knowledge";
import { useWorkspaceId } from "@multica/core/hooks";
import { useCurrentMember } from "@multica/core/permissions";
import type {
  KnowledgeCandidate,
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
  const { role } = useCurrentMember(workspaceId);
  const canReview = role === "owner" || role === "admin";
  const [query, setQuery] = useState("");
  const [section, setSection] = useState<"published" | "review">("published");
  const [showProposal, setShowProposal] = useState(false);
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

  const submitProposal = () => {
    if (!title.trim() || !content.trim() || !reason.trim()) return;
    propose.mutate(
      {
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
          setShowProposal(false);
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
          <Button size="sm" onClick={() => setShowProposal(true)}>
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
                {t(($) => $.proposal.title)}
              </h2>
              <Button
                size="icon-sm"
                variant="ghost"
                onClick={() => setShowProposal(false)}
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
