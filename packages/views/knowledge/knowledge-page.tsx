"use client";

import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { featureFlagEnabled, useConfigStore } from "@multica/core/config";
import { knowledgeDetailOptions, knowledgeListOptions } from "@multica/core/knowledge";
import { useWorkspaceId } from "@multica/core/hooks";
import { useCurrentMember } from "@multica/core/permissions";
import type { KnowledgeKind, KnowledgeQueryFilters } from "@multica/core/types";
import { Badge } from "@multica/ui/components/ui/badge";
import { Button } from "@multica/ui/components/ui/button";
import { Input } from "@multica/ui/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@multica/ui/components/ui/select";
import { BookOpenText, ChevronRight, Eye, Loader2, Search, ShieldAlert, X } from "lucide-react";
import { useT } from "../i18n";
import { useNavigation } from "../navigation";

const KNOWLEDGE_KINDS: KnowledgeKind[] = ["goal", "decision", "constraint", "requirement", "procedure", "lesson", "reference"];

export function KnowledgePage() {
  const { t } = useT("knowledge");
  const navigation = useNavigation();
  const workspaceId = useWorkspaceId();
  const { role } = useCurrentMember(workspaceId);
  const configLoaded = useConfigStore((state) => state.configLoaded);
  const featureFlags = useConfigStore((state) => state.featureFlags);
  const installed = configLoaded && featureFlagEnabled(featureFlags, "knowledge_query", false);
  const canSeeQuarantine = role === "owner" || role === "admin";
  const statusOptions = [
    { value: "published", label: "Published" },
    { value: "superseded", label: "Superseded" },
    ...(canSeeQuarantine ? [{ value: "quarantined", label: "Quarantined" }] : []),
  ] as const;
  const kindOptions = [
    { value: "all", label: "All kinds" },
    ...KNOWLEDGE_KINDS.map((value) => ({ value, label: t(($) => $.kinds[value]) })),
  ];
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<"published" | "superseded" | "quarantined">("published");
  const [kind, setKind] = useState<KnowledgeKind | "all">("all");
  const [cursor, setCursor] = useState<string | undefined>();
  const [selectedKnowledgeId, setSelectedKnowledgeId] = useState("");
  const sourceType = navigation.searchParams.get("source_type")?.trim() || undefined;
  const sourceId = navigation.searchParams.get("source_id")?.trim() || undefined;
  const sourceRevision = navigation.searchParams.get("source_revision")?.trim() || undefined;
  const projectId = navigation.searchParams.get("project_id")?.trim() || undefined;
  const rawRevision = navigation.searchParams.get("revision")?.trim();
  const revision = rawRevision && /^\d+$/.test(rawRevision) ? Number(rawRevision) : undefined;

  useEffect(() => { setCursor(undefined); }, [query, status, kind, sourceType, sourceId, sourceRevision, projectId, revision]);
  const filters = useMemo<KnowledgeQueryFilters>(() => ({
    query: query.trim() || undefined,
    statuses: [status], kinds: kind === "all" ? undefined : [kind],
    sourceType, sourceId, sourceRevision,
    applicability: projectId ? "project" : "workspace", projectId, revision,
    limit: 20, cursor,
  }), [query, status, kind, sourceType, sourceId, sourceRevision, projectId, revision, cursor]);
  const listQuery = useQuery(knowledgeListOptions(workspaceId, filters, installed));

  if (!configLoaded) return <LoadingState label={t(($) => $.states.loading)} />;
  if (!installed) return <ErrorState label={t(($) => $.states.load_failed)} />;

  return <div className="flex min-h-0 flex-1 flex-col">
    <header className="flex h-12 shrink-0 items-center gap-2 border-b px-4"><BookOpenText className="size-4 text-muted-foreground" /><h1 className="text-sm font-semibold">{t(($) => $.header.title)}</h1></header>
    <main className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-4 overflow-y-auto p-4 sm:p-6">
      <div className="grid gap-2 sm:grid-cols-[1fr_170px_170px]">
        <div className="relative"><Search className="pointer-events-none absolute top-2.5 left-3 size-4 text-muted-foreground" /><Input value={query} onChange={(event) => setQuery(event.target.value)} placeholder={t(($) => $.search.placeholder)} className="pl-9" /></div>
        <Select items={statusOptions} value={status} onValueChange={(value) => value && setStatus(value as typeof status)}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent>{statusOptions.map((option) => <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>)}</SelectContent></Select>
        <Select items={kindOptions} value={kind} onValueChange={(value) => value && setKind(value as KnowledgeKind | "all")}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent>{kindOptions.map((option) => <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>)}</SelectContent></Select>
      </div>
      {sourceType && sourceId ? <p className="text-xs text-muted-foreground">Source: {sourceType} / {sourceId}{sourceRevision ? ` @ ${sourceRevision}` : ""}</p> : null}
      {selectedKnowledgeId ? <KnowledgeDetails workspaceId={workspaceId} knowledgeId={selectedKnowledgeId} installed={installed} onClose={() => setSelectedKnowledgeId("")} /> : null}
      {listQuery.isLoading ? <LoadingState label={t(($) => $.states.loading)} /> : listQuery.isError ? <ErrorState label={t(($) => $.states.load_failed)} /> : (listQuery.data?.entries.length ?? 0) === 0 ? <EmptyState label={t(($) => $.states.empty)} /> : <div className="grid gap-3">{listQuery.data?.entries.map((entry) => {
        const current = entry.revisions.find((item) => item.number === (revision ?? entry.currentRevision)) ?? entry.revisions[0];
        return <article key={entry.id} className="space-y-3 rounded-xl border bg-card p-4"><div className="flex flex-wrap items-center gap-2"><Badge variant="secondary">{t(($) => $.kinds[entry.kind])}</Badge><Badge variant={entry.status === "quarantined" ? "destructive" : "outline"}>{entry.status}</Badge><span className="text-xs text-muted-foreground">Revision {current?.number}</span></div><div><h2 className="font-medium">{current?.title}</h2><p className="mt-1 whitespace-pre-wrap text-sm text-muted-foreground">{current?.content}</p></div><div className="rounded-lg bg-muted/40 p-3 text-xs text-muted-foreground"><p className="font-medium text-foreground">{entry.citation || "No citation"}</p><p>{entry.matchedBy} · {current?.sourceRefs.length ?? 0} source(s)</p></div><div className="flex justify-end"><Button size="sm" variant="ghost" onClick={() => setSelectedKnowledgeId(entry.id)}><Eye className="size-4" />{t(($) => $.detail.open)}</Button></div></article>;
      })}</div>}
      {listQuery.data?.nextCursor ? <div className="flex justify-end"><Button variant="outline" onClick={() => setCursor(listQuery.data?.nextCursor ?? undefined)}>Next page<ChevronRight className="size-4" /></Button></div> : null}
    </main>
  </div>;
}

function KnowledgeDetails({ workspaceId, knowledgeId, installed, onClose }: { workspaceId: string; knowledgeId: string; installed: boolean; onClose: () => void }) {
  const { t } = useT("knowledge");
  const query = useQuery(knowledgeDetailOptions(workspaceId, knowledgeId, installed));
  return <section className="space-y-3 rounded-xl border bg-card p-4"><div className="flex items-center justify-between"><h2 className="text-sm font-semibold">{t(($) => $.detail.title)}</h2><Button size="icon-sm" variant="ghost" onClick={onClose} aria-label={t(($) => $.detail.close)}><X className="size-4" /></Button></div>{query.isLoading ? <LoadingState label={t(($) => $.detail.loading)} /> : query.isError ? <ErrorState label={t(($) => $.detail.load_failed)} /> : query.data ? <div className="space-y-3">{query.data.revisions.map((revision) => <article key={revision.number} className="rounded-lg border p-3"><div className="flex items-center gap-2"><Badge variant="outline">Revision {revision.number}</Badge>{revision.supersedesRevision ? <span className="text-xs text-muted-foreground">Supersedes {revision.supersedesRevision}</span> : null}</div><h3 className="mt-2 font-medium">{revision.title}</h3><p className="mt-1 whitespace-pre-wrap text-sm text-muted-foreground">{revision.content}</p><div className="mt-3 space-y-1">{revision.sourceRefs.map((source) => <p key={`${source.type}:${source.id}:${source.revision}`} className="text-xs text-muted-foreground">{source.citation} ({source.type}:{source.id}@{source.revision})</p>)}</div></article>)}</div> : null}</section>;
}

function LoadingState({ label }: { label: string }) { return <div className="flex items-center justify-center gap-2 py-12 text-sm text-muted-foreground"><Loader2 className="size-4 animate-spin" />{label}</div>; }
function ErrorState({ label }: { label: string }) { return <div className="flex items-center justify-center gap-2 rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-10 text-sm text-destructive"><ShieldAlert className="size-4" />{label}</div>; }
function EmptyState({ label }: { label: string }) { return <div className="rounded-xl border border-dashed px-4 py-12 text-center text-sm text-muted-foreground">{label}</div>; }
