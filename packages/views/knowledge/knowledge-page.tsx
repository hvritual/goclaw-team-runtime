"use client";

import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { featureFlagEnabled, useConfigStore } from "@multica/core/config";
import {
  knowledgeCandidateListOptions,
  knowledgeDetailOptions,
  knowledgeListOptions,
  useProposeKnowledge,
  useReviewKnowledge,
} from "@multica/core/knowledge";
import { useWorkspaceId } from "@multica/core/hooks";
import { useCurrentMember } from "@multica/core/permissions";
import type {
  KnowledgeCandidate,
  KnowledgeKind,
  KnowledgeQueryFilters,
  KnowledgeReviewAction,
} from "@multica/core/types";
import { Badge } from "@multica/ui/components/ui/badge";
import { Button } from "@multica/ui/components/ui/button";
import { Input } from "@multica/ui/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@multica/ui/components/ui/select";
import { Textarea } from "@multica/ui/components/ui/textarea";
import {
  BookOpenText,
  ChevronRight,
  Eye,
  Loader2,
  Plus,
  Search,
  ShieldAlert,
  X,
} from "lucide-react";
import { useT } from "../i18n";
import { useNavigation } from "../navigation";

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
  const navigation = useNavigation();
  const workspaceId = useWorkspaceId();
  const { role, userId } = useCurrentMember(workspaceId);
  const configLoaded = useConfigStore((state) => state.configLoaded);
  const featureFlags = useConfigStore((state) => state.featureFlags);
  const installed =
    configLoaded && featureFlagEnabled(featureFlags, "knowledge_query", false);
  const reviewInstalled =
    configLoaded && featureFlagEnabled(featureFlags, "knowledge_review", false);
  const canSeeQuarantine = role === "owner" || role === "admin";
  const statusOptions = [
    { value: "published", label: "Published" },
    { value: "superseded", label: "Superseded" },
    ...(canSeeQuarantine
      ? [{ value: "quarantined", label: "Quarantined" }]
      : []),
  ] as const;
  const kindOptions = [
    { value: "all", label: "All kinds" },
    ...KNOWLEDGE_KINDS.map((value) => ({
      value,
      label: t(($) => $.kinds[value]),
    })),
  ];
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<
    "published" | "superseded" | "quarantined"
  >("published");
  const [kind, setKind] = useState<KnowledgeKind | "all">("all");
  const [cursor, setCursor] = useState<string | undefined>();
  const [selectedKnowledgeId, setSelectedKnowledgeId] = useState("");
  const [showProposal, setShowProposal] = useState(false);
  const [proposalKnowledgeId, setProposalKnowledgeId] = useState<string>();
  const sourceType =
    navigation.searchParams.get("source_type")?.trim() || undefined;
  const sourceId =
    navigation.searchParams.get("source_id")?.trim() || undefined;
  const sourceRevision =
    navigation.searchParams.get("source_revision")?.trim() || undefined;
  const projectId =
    navigation.searchParams.get("project_id")?.trim() || undefined;
  const rawRevision = navigation.searchParams.get("revision")?.trim();
  const revision =
    rawRevision && /^\d+$/.test(rawRevision) ? Number(rawRevision) : undefined;

  useEffect(() => {
    setCursor(undefined);
  }, [
    query,
    status,
    kind,
    sourceType,
    sourceId,
    sourceRevision,
    projectId,
    revision,
  ]);
  const filters = useMemo<KnowledgeQueryFilters>(
    () => ({
      query: query.trim() || undefined,
      statuses: [status],
      kinds: kind === "all" ? undefined : [kind],
      sourceType,
      sourceId,
      sourceRevision,
      applicability: projectId ? "project" : "workspace",
      projectId,
      revision,
      limit: 20,
      cursor,
    }),
    [
      query,
      status,
      kind,
      sourceType,
      sourceId,
      sourceRevision,
      projectId,
      revision,
      cursor,
    ]
  );
  const listQuery = useQuery(
    knowledgeListOptions(workspaceId, filters, installed)
  );

  if (!configLoaded) return <LoadingState label={t(($) => $.states.loading)} />;
  if (!installed) return <ErrorState label={t(($) => $.states.load_failed)} />;

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <header className="flex h-12 shrink-0 items-center gap-2 border-b px-4">
        <BookOpenText className="size-4 text-muted-foreground" />
        <h1 className="text-sm font-semibold">{t(($) => $.header.title)}</h1>
        <div className="ml-auto">
          {reviewInstalled ? (
            <Button
              size="sm"
              onClick={() => {
                setProposalKnowledgeId(undefined);
                setShowProposal((value) => !value);
              }}
            >
              <Plus className="size-4" />
              {t(($) => $.header.propose)}
            </Button>
          ) : null}
        </div>
      </header>
      <main className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-4 overflow-y-auto p-4 sm:p-6">
        {showProposal ? (
          <KnowledgeProposal
            workspaceId={workspaceId}
            knowledgeId={proposalKnowledgeId}
            onClose={() => setShowProposal(false)}
          />
        ) : null}
        {reviewInstalled && (role === "owner" || role === "admin") ? (
          <KnowledgeReviewQueue
            workspaceId={workspaceId}
            userId={userId}
            role={role}
          />
        ) : null}
        <div className="grid gap-2 sm:grid-cols-[1fr_170px_170px]">
          <div className="relative">
            <Search className="pointer-events-none absolute top-2.5 left-3 size-4 text-muted-foreground" />
            <Input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder={t(($) => $.search.placeholder)}
              className="pl-9"
            />
          </div>
          <Select
            items={statusOptions}
            value={status}
            onValueChange={(value) =>
              value && setStatus(value as typeof status)
            }
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {statusOptions.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select
            items={kindOptions}
            value={kind}
            onValueChange={(value) =>
              value && setKind(value as KnowledgeKind | "all")
            }
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {kindOptions.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        {sourceType && sourceId ? (
          <p className="text-xs text-muted-foreground">
            {t(($) => $.filters.source, { type: sourceType, id: sourceId })}
            {sourceRevision ? ` @ ${sourceRevision}` : ""}
          </p>
        ) : null}
        {selectedKnowledgeId ? (
          <KnowledgeDetails
            workspaceId={workspaceId}
            knowledgeId={selectedKnowledgeId}
            installed={installed}
            onClose={() => setSelectedKnowledgeId("")}
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
              const current =
                entry.revisions.find(
                  (item) => item.number === (revision ?? entry.currentRevision)
                ) ?? entry.revisions[0];
              return (
                <article
                  key={entry.id}
                  className="space-y-3 rounded-xl border bg-card p-4"
                >
                  <div className="flex flex-wrap items-center gap-2">
                    <Badge variant="secondary">
                      {t(($) => $.kinds[entry.kind])}
                    </Badge>
                    <Badge
                      variant={
                        entry.status === "quarantined"
                          ? "destructive"
                          : "outline"
                      }
                    >
                      {entry.status}
                    </Badge>
                    <span className="text-xs text-muted-foreground">
                      {t(($) => $.detail.revision, { number: current?.number ?? 0 })}
                    </span>
                  </div>
                  <div>
                    <h2 className="font-medium">{current?.title}</h2>
                    <p className="mt-1 whitespace-pre-wrap text-sm text-muted-foreground">
                      {current?.content}
                    </p>
                  </div>
                  <div className="rounded-lg bg-muted/40 p-3 text-xs text-muted-foreground">
                    <p className="font-medium text-foreground">
                      {entry.citation || "No citation"}
                    </p>
                    <p>
                      {entry.matchedBy} · {t(($) => $.source.count_other, { count: current?.sourceRefs.length ?? 0 })}
                    </p>
                  </div>
                  <div className="flex justify-end">
                    {reviewInstalled ? (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => {
                          setProposalKnowledgeId(entry.id);
                          setShowProposal(true);
                        }}
                      >
                        {t(($) => $.detail.propose_revision)}
                      </Button>
                    ) : null}
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => setSelectedKnowledgeId(entry.id)}
                    >
                      <Eye className="size-4" />
                      {t(($) => $.detail.open)}
                    </Button>
                  </div>
                </article>
              );
            })}
          </div>
        )}
        {listQuery.data?.nextCursor ? (
          <div className="flex justify-end">
            <Button
              variant="outline"
              onClick={() => setCursor(listQuery.data?.nextCursor ?? undefined)}
            >
              {t(($) => $.pagination.next)}
              <ChevronRight className="size-4" />
            </Button>
          </div>
        ) : null}
      </main>
    </div>
  );
}

function KnowledgeDetails({
  workspaceId,
  knowledgeId,
  installed,
  onClose,
}: {
  workspaceId: string;
  knowledgeId: string;
  installed: boolean;
  onClose: () => void;
}) {
  const { t } = useT("knowledge");
  const query = useQuery(
    knowledgeDetailOptions(workspaceId, knowledgeId, installed)
  );
  return (
    <section className="space-y-3 rounded-xl border bg-card p-4">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold">{t(($) => $.detail.title)}</h2>
        <Button
          size="icon-sm"
          variant="ghost"
          onClick={onClose}
          aria-label={t(($) => $.detail.close)}
        >
          <X className="size-4" />
        </Button>
      </div>
      {query.isLoading ? (
        <LoadingState label={t(($) => $.detail.loading)} />
      ) : query.isError ? (
        <ErrorState label={t(($) => $.detail.load_failed)} />
      ) : query.data ? (
        <div className="space-y-3">
          {query.data.revisions.map((revision) => (
            <article key={revision.number} className="rounded-lg border p-3">
              <div className="flex items-center gap-2">
                <Badge variant="outline">
                  {t(($) => $.detail.revision, { number: revision.number })}
                </Badge>
                {revision.supersedesRevision ? (
                  <span className="text-xs text-muted-foreground">
                    {t(($) => $.detail.supersedes, { number: revision.supersedesRevision })}
                  </span>
                ) : null}
              </div>
              <h3 className="mt-2 font-medium">{revision.title}</h3>
              <p className="mt-1 whitespace-pre-wrap text-sm text-muted-foreground">
                {revision.content}
              </p>
              <div className="mt-3 space-y-1">
                {revision.sourceRefs.map((source) => (
                  <p
                    key={`${source.type}:${source.id}:${source.revision}`}
                    className="text-xs text-muted-foreground"
                  >
                    {source.citation} ({source.type}:{source.id}@
                    {source.revision})
                  </p>
                ))}
              </div>
            </article>
          ))}
        </div>
      ) : null}
    </section>
  );
}

function KnowledgeProposal({
  workspaceId,
  knowledgeId,
  onClose,
}: {
  workspaceId: string;
  knowledgeId?: string;
  onClose: () => void;
}) {
  const { t } = useT("knowledge");
  const mutation = useProposeKnowledge(workspaceId);
  const [idempotencyKey] = useState(() => crypto.randomUUID());
  const [kind, setKind] = useState<KnowledgeKind>("lesson");
  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");
  const [reason, setReason] = useState("");
  const [sourceId, setSourceId] = useState("");
  const [sourceRevision, setSourceRevision] = useState("");
  const [citation, setCitation] = useState("");
  const submit = async () => {
    await mutation.mutateAsync({
      idempotencyKey,
      knowledgeId,
      kind,
      title,
      content,
      reason,
      sourceRefs:
        sourceId.trim() && sourceRevision.trim() && citation.trim()
          ? [
              {
                type: "acceptance_conclusion",
                id: sourceId,
                revision: sourceRevision,
                citation,
              },
            ]
          : [],
    });
    onClose();
  };
  return (
    <section
      className="space-y-3 rounded-xl border bg-card p-4"
      data-testid="knowledge-proposal"
    >
      <div className="flex items-center justify-between">
        <h2 className="font-semibold">
          {knowledgeId
            ? t(($) => $.proposal.revision_title)
            : t(($) => $.proposal.title)}
        </h2>
        <Button size="icon-sm" variant="ghost" onClick={onClose}>
          <X className="size-4" />
        </Button>
      </div>
      <Select
        items={KNOWLEDGE_KINDS.map((value) => ({ value, label: t(($) => $.kinds[value]) }))}
        value={kind}
        onValueChange={(value) => value && setKind(value as KnowledgeKind)}
      >
        <SelectTrigger>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {KNOWLEDGE_KINDS.map((value) => (
            <SelectItem key={value} value={value}>
              {t(($) => $.kinds[value])}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Input
        value={title}
        onChange={(event) => setTitle(event.target.value)}
        placeholder={t(($) => $.proposal.title_placeholder)}
      />
      <Textarea
        value={content}
        onChange={(event) => setContent(event.target.value)}
        placeholder={t(($) => $.proposal.content_placeholder)}
      />
      <Textarea
        value={reason}
        onChange={(event) => setReason(event.target.value)}
        placeholder={t(($) => $.proposal.reason_placeholder)}
      />
      <div className="grid gap-2 sm:grid-cols-3">
        <Input
          value={sourceId}
          onChange={(event) => setSourceId(event.target.value)}
          placeholder="Source ID"
        />
        <Input
          value={sourceRevision}
          onChange={(event) => setSourceRevision(event.target.value)}
          placeholder="Source revision"
        />
        <Input
          value={citation}
          onChange={(event) => setCitation(event.target.value)}
          placeholder="Citation"
        />
      </div>
      {mutation.isError ? (
        <p className="text-sm text-destructive">{t(($) => $.proposal.failed)}</p>
      ) : null}
      <div className="flex justify-end">
        <Button
          onClick={submit}
          disabled={
            mutation.isPending ||
            !title.trim() ||
            !content.trim() ||
            !reason.trim()
          }
        >
          {t(($) => $.proposal.submit)}
        </Button>
      </div>
    </section>
  );
}

function KnowledgeReviewQueue({
  workspaceId,
  userId,
  role,
}: {
  workspaceId: string;
  userId: string | null;
  role: "owner" | "admin";
}) {
  const { t } = useT("knowledge");
  const query = useQuery(knowledgeCandidateListOptions(workspaceId, true));
  return (
    <section
      className="space-y-3 rounded-xl border bg-card p-4"
      data-testid="knowledge-review-queue"
    >
      <h2 className="font-semibold">{t(($) => $.tabs.review_queue)}</h2>
      {query.isLoading ? (
        <LoadingState label={t(($) => $.review.loading_candidates)} />
      ) : query.isError ? (
        <ErrorState label={t(($) => $.review.load_failed)} />
      ) : query.data?.candidates.length ? (
        <div className="space-y-3">
          {query.data.candidates.map((candidate) => (
            <KnowledgeCandidateReview
              key={candidate.id}
              workspaceId={workspaceId}
              candidate={candidate}
              userId={userId}
              role={role}
            />
          ))}
        </div>
      ) : (
        <EmptyState label={t(($) => $.states.empty_candidates)} />
      )}
    </section>
  );
}

function KnowledgeCandidateReview({
  workspaceId,
  candidate,
  userId,
  role,
}: {
  workspaceId: string;
  candidate: KnowledgeCandidate;
  userId: string | null;
  role: "owner" | "admin";
}) {
  const { t } = useT("knowledge");
  const mutation = useReviewKnowledge(workspaceId);
  const [rationale, setRationale] = useState("");
  const [emergency, setEmergency] = useState(false);
  const isSelf = candidate.proposedBy === userId;
  const actions: KnowledgeReviewAction[] =
    candidate.status === "candidate"
      ? ["approve"]
      : candidate.status === "quarantined"
      ? ["return"]
      : candidate.status === "in_review"
      ? candidate.knowledgeId
        ? ["reject", "quarantine", "supersede", "invalidate"]
        : ["reject", "quarantine", "publish"]
      : [];
  const run = (action: KnowledgeReviewAction) =>
    mutation.mutate({
      candidateId: candidate.id,
      action,
      expectedRevision: candidate.revision,
      rationale,
      emergency: isSelf && emergency,
    });
  return (
    <article className="space-y-3 rounded-lg border p-3">
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant="outline">{candidate.status}</Badge>
        <Badge variant="secondary">{candidate.kind}</Badge>
        <span className="text-xs text-muted-foreground">
          {t(($) => $.detail.revision, { number: candidate.revision })}
        </span>
        {isSelf ? <Badge variant="destructive">{t(($) => $.review.your_proposal)}</Badge> : null}
      </div>
      <div>
        <h3 className="font-medium">{candidate.title}</h3>
        <p className="text-sm text-muted-foreground">{candidate.content}</p>
      </div>
      <Input
        value={rationale}
        onChange={(event) => setRationale(event.target.value)}
        placeholder={t(($) => $.review.rationale_placeholder)}
      />
      {isSelf && role === "owner" ? (
        <label className="flex items-center gap-2 text-xs">
          <input
            type="checkbox"
            checked={emergency}
            onChange={(event) => setEmergency(event.target.checked)}
          />
          {t(($) => $.review.emergency_self_review)}
        </label>
      ) : null}
      <div className="flex flex-wrap gap-2">
        {actions.map((action) => (
          <Button
            key={action}
            size="sm"
            variant={
              action === "reject" || action === "invalidate"
                ? "destructive"
                : "outline"
            }
            disabled={
              mutation.isPending ||
              !rationale.trim() ||
              (isSelf &&
                (role !== "owner" ||
                  !emergency ||
                  rationale.trim().length < 12))
            }
            onClick={() => run(action)}
          >
            {action}
          </Button>
        ))}
      </div>
      {mutation.isError ? (
        <p className="text-sm text-destructive">
          {t(($) => $.review.failed)}
        </p>
      ) : null}
    </article>
  );
}

function LoadingState({ label }: { label: string }) {
  return (
    <div className="flex items-center justify-center gap-2 py-12 text-sm text-muted-foreground">
      <Loader2 className="size-4 animate-spin" />
      {label}
    </div>
  );
}
function ErrorState({ label }: { label: string }) {
  return (
    <div className="flex items-center justify-center gap-2 rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-10 text-sm text-destructive">
      <ShieldAlert className="size-4" />
      {label}
    </div>
  );
}
function EmptyState({ label }: { label: string }) {
  return (
    <div className="rounded-xl border border-dashed px-4 py-12 text-center text-sm text-muted-foreground">
      {label}
    </div>
  );
}
