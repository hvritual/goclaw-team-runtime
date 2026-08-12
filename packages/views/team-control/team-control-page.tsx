"use client";

import { useMemo, useState } from "react";
import { useQueries } from "@tanstack/react-query";
import {
  isTeamControlConflict,
  teamControlMembersOptions,
  teamControlProjectionOptions,
  teamControlWorkspaceOptions,
  useTeamControlCommand,
  useTeamControlEvents,
  type TeamControlCommandInput,
  type TeamControlConnectionState,
  type TeamControlWorkNode,
} from "@multica/core/team-control";
import { useWorkspaceId } from "@multica/core/hooks";
import { useWorkspacePaths } from "@multica/core/paths";
import { ApiError } from "@multica/core/api";
import { Alert, AlertDescription, AlertTitle } from "@multica/ui/components/ui/alert";
import { Badge } from "@multica/ui/components/ui/badge";
import { Button } from "@multica/ui/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@multica/ui/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@multica/ui/components/ui/dialog";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@multica/ui/components/ui/empty";
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@multica/ui/components/ui/field";
import { Input } from "@multica/ui/components/ui/input";
import { Separator } from "@multica/ui/components/ui/separator";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { Spinner } from "@multica/ui/components/ui/spinner";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@multica/ui/components/ui/tabs";
import { Textarea } from "@multica/ui/components/ui/textarea";
import {
  AlertTriangle,
  ArrowLeft,
  Bot,
  CheckCircle2,
  ClipboardCheck,
  FileCheck2,
  GitBranch,
  ListChecks,
  Plus,
  RefreshCw,
  ShieldCheck,
  Users,
} from "lucide-react";
import { toast } from "sonner";
import { useT } from "../i18n";
import { useNavigation } from "../navigation";

type ActionKind =
  | "requirement"
  | "defect"
  | "risk"
  | "finding"
  | "knowledge"
  | "run";

interface ActionForm {
  id: string;
  summary: string;
  detail: string;
  extra: string;
  references: string;
  valueA: string;
  valueB: string;
  dueAt: string;
}

const emptyForm = (): ActionForm => ({
  id: `item-${Date.now().toString(36)}`,
  summary: "",
  detail: "",
  extra: "",
  references: "",
  valueA: "",
  valueB: "",
  dueAt: "",
});

type ActionCopy = Record<ActionKind, { title: string; description: string }>;

function parseList(value: string): string[] {
  return value.split(",").map((item) => item.trim()).filter(Boolean);
}

function canSubmitAction(action: ActionKind, form: ActionForm): boolean {
  if (!form.id.trim()) return false;
  switch (action) {
    case "requirement":
      return Boolean(form.summary.trim());
    case "defect":
      return Boolean(form.summary.trim() && form.valueA.trim() && form.detail.trim());
    case "risk": {
      const probability = Number(form.valueA);
      const impact = Number(form.valueB);
      return Boolean(
        form.summary.trim()
        && probability >= 1 && probability <= 5
        && impact >= 1 && impact <= 5
        && form.detail.trim()
        && form.dueAt
        && !Number.isNaN(new Date(form.dueAt).getTime()),
      );
    }
    case "finding":
      return Boolean(form.summary.trim() && form.valueA.trim());
    case "knowledge":
      return Boolean(
        form.summary.trim()
        && form.detail.trim()
        && parseList(form.references).length
        && parseList(form.extra).length,
      );
    case "run":
      return Boolean(form.detail.trim() && Number(form.valueA) >= 1);
  }
}

function buildCommand(
  action: ActionKind,
  form: ActionForm,
  expectedHead: number,
): TeamControlCommandInput {
  switch (action) {
    case "requirement":
      return { type: "requirement.start", expectedHead, payload: { id: form.id, text: form.summary } };
    case "defect":
      return {
        type: "defect.create",
        expectedHead,
        payload: { id: form.id, data: { summary: form.summary, severity: form.valueA, reproduction: form.detail } },
      };
    case "risk":
      return {
        type: "risk.create",
        expectedHead,
        payload: {
          id: form.id,
          data: {
            summary: form.summary,
            probability: Number(form.valueA),
            impact: Number(form.valueB),
            response_plan: form.detail,
            review_due_at: new Date(form.dueAt).toISOString(),
          },
        },
      };
    case "finding":
      return {
        type: "finding.create",
        expectedHead,
        payload: { id: form.id, data: { rule_id: form.valueA, summary: form.summary, model_finding: false } },
      };
    case "knowledge":
      return {
        type: "knowledge.create",
        expectedHead,
        payload: {
          id: form.id,
          data: {
            title: form.summary,
            source_ids: parseList(form.references),
            evidence_ids: parseList(form.extra),
            dedup_key: form.detail,
          },
        },
      };
    case "run":
      return {
        type: "run.queue",
        expectedHead,
        payload: {
          id: form.id,
          workspace_ref: form.detail,
          secret_refs: parseList(form.references),
          max_attempts: Number(form.valueA),
        },
      };
  }
}

function nodeData(node: TeamControlWorkNode): Record<string, unknown> {
  return node.data && typeof node.data === "object" && !Array.isArray(node.data)
    ? node.data as Record<string, unknown>
    : {};
}

function nodeLabel(node: TeamControlWorkNode): string {
  const data = nodeData(node);
  for (const key of ["request", "summary", "title", "workspace_ref", "rule_id"]) {
    const value = data[key];
    if (typeof value === "string" && value.trim()) return value;
  }
  return node.id;
}

function friendlyError(error: unknown, requestFailed: (status: number) => string, unknownError: string): string {
  if (error instanceof ApiError) {
    const detail = error.body && typeof error.body === "object"
      ? (error.body as { detail?: unknown }).detail
      : undefined;
    if (typeof detail === "string") return detail;
    return requestFailed(error.status);
  }
  return error instanceof Error ? error.message : unknownError;
}

function MetricCard({ label, value, description }: { label: string; value: number | string; description: string }) {
  return (
    <Card size="sm">
      <CardHeader>
        <CardDescription>{label}</CardDescription>
        <CardTitle className="text-title-lg tabular-nums">{value}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-caption text-muted-foreground">{description}</p>
      </CardContent>
    </Card>
  );
}

function NodeList({
  nodes,
  emptyTitle,
  emptyDescription,
  revisionLabel,
  assigneeLabel,
}: {
  nodes: TeamControlWorkNode[];
  emptyTitle: string;
  emptyDescription: string;
  revisionLabel: (count: number) => string;
  assigneeLabel: (count: number) => string;
}) {
  if (nodes.length === 0) {
    return (
      <Empty>
        <EmptyHeader>
          <EmptyMedia variant="icon"><ListChecks /></EmptyMedia>
          <EmptyTitle>{emptyTitle}</EmptyTitle>
          <EmptyDescription>{emptyDescription}</EmptyDescription>
        </EmptyHeader>
      </Empty>
    );
  }
  return (
    <div className="grid gap-3 md:grid-cols-2">
      {nodes.map((node) => (
        <Card key={node.id} size="sm">
          <CardHeader>
            <CardTitle className="truncate text-body">{nodeLabel(node)}</CardTitle>
            <CardDescription className="truncate">{node.id}</CardDescription>
            <CardAction><Badge variant="secondary">{node.state}</Badge></CardAction>
          </CardHeader>
          <CardContent className="flex flex-wrap items-center gap-2 text-caption text-muted-foreground">
            <span>{node.kind}</span>
            <span>{revisionLabel(node.revision)}</span>
            {node.assignee_ids.length > 0 ? <span>{assigneeLabel(node.assignee_ids.length)}</span> : null}
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

function ActionButtons({ actions, copy, onOpen }: { actions: ActionKind[]; copy: ActionCopy; onOpen: (action: ActionKind) => void }) {
  return (
    <div className="flex flex-wrap gap-2">
      {actions.map((action) => (
        <Button key={action} variant="outline" size="sm" onClick={() => onOpen(action)}>
          <Plus data-icon="inline-start" />
          {copy[action].title}
        </Button>
      ))}
    </div>
  );
}

function ProjectionSkeleton() {
  const { t } = useT("projects");
  return (
    <div className="flex flex-col gap-4 p-4 md:p-6" aria-label={t(($) => $.team_control.loading)}>
      <Skeleton className="h-8 w-64" />
      <Skeleton className="h-4 w-96 max-w-full" />
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }, (_, index) => <Skeleton key={index} className="h-28 w-full" />)}
      </div>
      <Skeleton className="h-72 w-full" />
    </div>
  );
}

export function TeamControlPage({ projectId }: { projectId: string }) {
  const workspaceId = useWorkspaceId();
  return <TeamControlView workspaceId={workspaceId} projectId={projectId} />;
}

export function TeamControlView({ workspaceId, projectId }: { workspaceId: string; projectId: string }) {
  const { t } = useT("projects");
  const navigation = useNavigation();
  const workspacePaths = useWorkspacePaths();
  const [workspaceQuery, membersQuery, projectionQuery] = useQueries({
    queries: [
      teamControlWorkspaceOptions(workspaceId),
      teamControlMembersOptions(workspaceId),
      teamControlProjectionOptions(workspaceId, projectId),
    ],
  });
  const connection = useTeamControlEvents(workspaceId, projectId);
  const command = useTeamControlCommand(workspaceId, projectId);
  const [action, setAction] = useState<ActionKind | null>(null);
  const [form, setForm] = useState<ActionForm>(emptyForm);
  const actionCopy: ActionCopy = {
    requirement: {
      title: t(($) => $.team_control.actions.requirement_title),
      description: t(($) => $.team_control.actions.requirement_description),
    },
    defect: {
      title: t(($) => $.team_control.actions.defect_title),
      description: t(($) => $.team_control.actions.defect_description),
    },
    risk: {
      title: t(($) => $.team_control.actions.risk_title),
      description: t(($) => $.team_control.actions.risk_description),
    },
    finding: {
      title: t(($) => $.team_control.actions.finding_title),
      description: t(($) => $.team_control.actions.finding_description),
    },
    knowledge: {
      title: t(($) => $.team_control.actions.knowledge_title),
      description: t(($) => $.team_control.actions.knowledge_description),
    },
    run: {
      title: t(($) => $.team_control.actions.run_title),
      description: t(($) => $.team_control.actions.run_description),
    },
  };
  const connectionCopy: Record<TeamControlConnectionState, string> = {
    connecting: t(($) => $.team_control.connection.connecting),
    connected: t(($) => $.team_control.connection.connected),
    reconnecting: t(($) => $.team_control.connection.reconnecting),
    offline: t(($) => $.team_control.connection.offline),
  };

  const projection = projectionQuery.data;
  const nodes = useMemo(
    () => projection ? Object.values(projection.nodes).toSorted((left, right) => left.id.localeCompare(right.id)) : [],
    [projection],
  );
  const nodesByKind = (kinds: string[]) => nodes.filter((node) => kinds.includes(node.kind));
  const checkCount = projection
    ? Object.values(projection.checks).reduce((total, checks) => total + checks.length, 0)
    : 0;

  const openAction = (nextAction: ActionKind) => {
    command.reset();
    setForm(emptyForm());
    setAction(nextAction);
  };

  const submitAction = async () => {
    if (!action || !projection) return;
    try {
      await command.mutateAsync(buildCommand(action, form, projection.head));
      toast.success(t(($) => $.team_control.actions.submitted, { action: actionCopy[action].title }));
      setAction(null);
    } catch {
      // The dialog renders the structured mutation error and keeps user input.
    }
  };

  if (workspaceQuery.isLoading || membersQuery.isLoading || projectionQuery.isLoading) {
    return <ProjectionSkeleton />;
  }

  if (projectionQuery.error) {
    const denied = projectionQuery.error instanceof ApiError && projectionQuery.error.status === 403;
    return (
      <div className="flex flex-1 items-center justify-center p-4 md:p-6">
        <Alert variant="destructive" className="max-w-xl">
          <ShieldCheck />
          <AlertTitle>{denied
            ? t(($) => $.team_control.errors.denied_title)
            : t(($) => $.team_control.errors.unavailable_title)}</AlertTitle>
          <AlertDescription>
            {denied
              ? t(($) => $.team_control.errors.denied_description)
              : friendlyError(
                  projectionQuery.error,
                  (status) => t(($) => $.team_control.errors.request_failed, { status }),
                  t(($) => $.team_control.errors.unknown),
                )}
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  if (!projection) return <ProjectionSkeleton />;

  return (
    <main className="flex min-h-0 flex-1 flex-col overflow-auto" aria-labelledby="team-control-title">
      <header className="sticky top-0 z-10 flex flex-wrap items-center gap-3 border-b bg-background/95 px-4 py-3 backdrop-blur md:px-6">
        <Button
          variant="ghost"
          size="icon-sm"
          aria-label={t(($) => $.team_control.back)}
          onClick={() => navigation.push(workspacePaths.projectDetail(projectId))}
        >
          <ArrowLeft />
        </Button>
        <div className="min-w-0 flex-1">
          <h1 id="team-control-title" className="truncate text-title-sm font-semibold">{t(($) => $.team_control.title)}</h1>
          <p className="truncate text-caption text-muted-foreground">
            {workspaceQuery.data?.workspace.name ?? workspaceId} · {projectId}
          </p>
        </div>
        <Badge variant={connection === "connected" ? "secondary" : "outline"}>
          {connection === "reconnecting" ? <RefreshCw data-icon="inline-start" /> : null}
          {connectionCopy[connection]}
        </Badge>
        <Badge variant="outline">{t(($) => $.team_control.head, { count: projection.head })}</Badge>
      </header>

      <div className="flex flex-col gap-5 p-4 md:p-6">
        {connection !== "connected" ? (
          <Alert>
            <AlertTriangle />
            <AlertTitle>{connectionCopy[connection]}</AlertTitle>
            <AlertDescription>
              {t(($) => $.team_control.connection.stale_hint)}
            </AlertDescription>
          </Alert>
        ) : null}

        <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4" aria-label={t(($) => $.team_control.summary.aria)}>
          <MetricCard label={t(($) => $.team_control.summary.work_nodes)} value={nodes.length} description={t(($) => $.team_control.summary.work_nodes_description)} />
          <MetricCard label={t(($) => $.team_control.summary.evidence)} value={Object.keys(projection.evidence).length} description={t(($) => $.team_control.summary.evidence_description)} />
          <MetricCard label={t(($) => $.team_control.summary.checks)} value={checkCount} description={t(($) => $.team_control.summary.checks_description)} />
          <MetricCard label={t(($) => $.team_control.summary.accepted)} value={Object.keys(projection.acceptances).length} description={t(($) => $.team_control.summary.accepted_description)} />
        </section>

        <Tabs defaultValue="overview" className="min-w-0">
          <div className="overflow-x-auto">
            <TabsList variant="line" className="w-max min-w-full justify-start">
              <TabsTrigger value="overview">{t(($) => $.team_control.tabs.overview)}</TabsTrigger>
              <TabsTrigger value="requirements">{t(($) => $.team_control.tabs.requirements)}</TabsTrigger>
              <TabsTrigger value="quality">{t(($) => $.team_control.tabs.quality)}</TabsTrigger>
              <TabsTrigger value="review">{t(($) => $.team_control.tabs.review)}</TabsTrigger>
              <TabsTrigger value="runtime">{t(($) => $.team_control.tabs.runner)}</TabsTrigger>
              <TabsTrigger value="governance">{t(($) => $.team_control.tabs.governance)}</TabsTrigger>
              <TabsTrigger value="members">{t(($) => $.team_control.tabs.members)}</TabsTrigger>
            </TabsList>
          </div>

          <TabsContent value="overview" className="flex flex-col gap-4 pt-4">
            <Card>
              <CardHeader>
                <CardTitle>{t(($) => $.team_control.overview.title)}</CardTitle>
                <CardDescription>{t(($) => $.team_control.overview.description)}</CardDescription>
                <CardAction><GitBranch /></CardAction>
              </CardHeader>
              <CardContent className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                <MetricCard label={t(($) => $.team_control.overview.edges)} value={Object.keys(projection.edges).length} description={t(($) => $.team_control.overview.edges_description)} />
                <MetricCard label={t(($) => $.team_control.overview.head_hash)} value={projection.head_hash ? projection.head_hash.slice(0, 10) : t(($) => $.team_control.overview.initial)} description={t(($) => $.team_control.overview.head_hash_description)} />
                <MetricCard label={t(($) => $.team_control.overview.schema)} value={`v${projection.schema_version}`} description={t(($) => $.team_control.overview.schema_description)} />
              </CardContent>
            </Card>
            <NodeList
              nodes={nodes.slice(0, 6)}
              emptyTitle={t(($) => $.team_control.empty.all)}
              emptyDescription={t(($) => $.team_control.empty.description)}
              revisionLabel={(count) => t(($) => $.team_control.node.revision, { count })}
              assigneeLabel={(count) => t(($) => $.team_control.node.assignees, { count })}
            />
          </TabsContent>

          <TabsContent value="requirements" className="flex flex-col gap-4 pt-4">
            <ActionButtons actions={["requirement"]} copy={actionCopy} onOpen={openAction} />
            <NodeList nodes={nodesByKind(["requirement", "task"])} emptyTitle={t(($) => $.team_control.empty.requirements)} emptyDescription={t(($) => $.team_control.empty.description)} revisionLabel={(count) => t(($) => $.team_control.node.revision, { count })} assigneeLabel={(count) => t(($) => $.team_control.node.assignees, { count })} />
          </TabsContent>

          <TabsContent value="quality" className="flex flex-col gap-4 pt-4">
            <ActionButtons actions={["defect", "risk"]} copy={actionCopy} onOpen={openAction} />
            <NodeList nodes={nodesByKind(["defect", "risk"])} emptyTitle={t(($) => $.team_control.empty.quality)} emptyDescription={t(($) => $.team_control.empty.description)} revisionLabel={(count) => t(($) => $.team_control.node.revision, { count })} assigneeLabel={(count) => t(($) => $.team_control.node.assignees, { count })} />
          </TabsContent>

          <TabsContent value="review" className="flex flex-col gap-4 pt-4">
            <ActionButtons actions={["finding", "knowledge"]} copy={actionCopy} onOpen={openAction} />
            <NodeList nodes={nodesByKind(["review_finding", "knowledge_candidate"])} emptyTitle={t(($) => $.team_control.empty.review)} emptyDescription={t(($) => $.team_control.empty.description)} revisionLabel={(count) => t(($) => $.team_control.node.revision, { count })} assigneeLabel={(count) => t(($) => $.team_control.node.assignees, { count })} />
          </TabsContent>

          <TabsContent value="runtime" className="flex flex-col gap-4 pt-4">
            <ActionButtons actions={["run"]} copy={actionCopy} onOpen={openAction} />
            <NodeList nodes={nodesByKind(["run"])} emptyTitle={t(($) => $.team_control.empty.runner)} emptyDescription={t(($) => $.team_control.empty.description)} revisionLabel={(count) => t(($) => $.team_control.node.revision, { count })} assigneeLabel={(count) => t(($) => $.team_control.node.assignees, { count })} />
          </TabsContent>

          <TabsContent value="governance" className="grid gap-4 pt-4 lg:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>{t(($) => $.team_control.governance.evidence_title)}</CardTitle>
                <CardDescription>{t(($) => $.team_control.governance.evidence_description)}</CardDescription>
                <CardAction><FileCheck2 /></CardAction>
              </CardHeader>
              <CardContent className="flex flex-col gap-3">
                {Object.values(projection.evidence).length === 0
                  ? <p className="text-body text-muted-foreground">{t(($) => $.team_control.governance.no_evidence)}</p>
                  : Object.values(projection.evidence).map((evidence) => (
                    <div key={evidence.id} className="flex items-center gap-3 rounded-lg border p-3">
                      <FileCheck2 className="shrink-0" />
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-body font-medium">{evidence.kind}</p>
                        <p className="truncate text-caption text-muted-foreground">{evidence.subject_id} · {evidence.media_type}</p>
                      </div>
                      <Badge variant={evidence.sanitized ? "secondary" : "destructive"}>
                        {evidence.sanitized
                          ? t(($) => $.team_control.governance.sanitized)
                          : t(($) => $.team_control.governance.unsafe)}
                      </Badge>
                    </div>
                  ))}
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle>{t(($) => $.team_control.governance.checks_title)}</CardTitle>
                <CardDescription>{t(($) => $.team_control.governance.checks_description)}</CardDescription>
                <CardAction><ClipboardCheck /></CardAction>
              </CardHeader>
              <CardContent className="flex flex-col gap-3">
                <div className="flex items-center justify-between gap-3">
                  <span className="text-body">{t(($) => $.team_control.governance.recorded_checks)}</span><Badge variant="outline">{checkCount}</Badge>
                </div>
                <Separator />
                <div className="flex items-center justify-between gap-3">
                  <span className="text-body">{t(($) => $.team_control.governance.independent_acceptances)}</span><Badge variant="outline">{Object.keys(projection.acceptances).length}</Badge>
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="members" className="pt-4">
            <Card>
              <CardHeader>
                <CardTitle>{t(($) => $.team_control.members.title)}</CardTitle>
                <CardDescription>{t(($) => $.team_control.members.description)}</CardDescription>
                <CardAction><Users /></CardAction>
              </CardHeader>
              <CardContent className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {(membersQuery.data?.members ?? []).map((member) => (
                  <div key={member.id} className="flex min-w-0 items-center gap-3 rounded-lg border p-3">
                    {member.kind === "agent" ? <Bot className="shrink-0" /> : <Users className="shrink-0" />}
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-body font-medium">{member.id}</p>
                      <p className="text-caption text-muted-foreground">{member.kind}</p>
                    </div>
                    <Badge variant="secondary">{member.role}</Badge>
                  </div>
                ))}
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>

      <ActionDialog
        action={action}
        copy={actionCopy}
        form={form}
        pending={command.isPending}
        error={command.error}
        onChange={setForm}
        onClose={() => setAction(null)}
        onSubmit={() => void submitAction()}
      />
    </main>
  );
}

function ActionDialog({
  action,
  copy,
  form,
  pending,
  error,
  onChange,
  onClose,
  onSubmit,
}: {
  action: ActionKind | null;
  copy: ActionCopy;
  form: ActionForm;
  pending: boolean;
  error: unknown;
  onChange: (form: ActionForm) => void;
  onClose: () => void;
  onSubmit: () => void;
}) {
  const { t } = useT("projects");
  if (!action) return null;
  const actionText = copy[action];
  const set = (key: keyof ActionForm, value: string) => onChange({ ...form, [key]: value });
  const conflict = isTeamControlConflict(error);
  const canSubmit = canSubmitAction(action, form);

  return (
    <Dialog open onOpenChange={(open) => { if (!open && !pending) onClose(); }}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{actionText.title}</DialogTitle>
          <DialogDescription>{actionText.description}</DialogDescription>
        </DialogHeader>
        {error ? (
          <Alert variant="destructive">
            <AlertTriangle />
            <AlertTitle>{conflict
              ? t(($) => $.team_control.errors.conflict_title)
              : t(($) => $.team_control.errors.rejected_title)}</AlertTitle>
            <AlertDescription>
              {friendlyError(
                error,
                (status) => t(($) => $.team_control.errors.request_failed, { status }),
                t(($) => $.team_control.errors.unknown),
              )}
              {conflict ? t(($) => $.team_control.errors.conflict_hint) : ""}
            </AlertDescription>
          </Alert>
        ) : null}
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="team-control-id">{t(($) => $.team_control.actions.id)}</FieldLabel>
            <Input id="team-control-id" value={form.id} onChange={(event) => set("id", event.target.value)} required />
            <FieldDescription>{t(($) => $.team_control.actions.id_description)}</FieldDescription>
          </Field>
          <Field>
            <FieldLabel htmlFor="team-control-summary">{action === "run"
              ? t(($) => $.team_control.actions.run_label)
              : t(($) => $.team_control.actions.summary)}</FieldLabel>
            <Input id="team-control-summary" value={form.summary} onChange={(event) => set("summary", event.target.value)} required={action !== "run"} />
          </Field>
          {action === "defect" ? (
            <>
              <Field>
                <FieldLabel htmlFor="team-control-severity">{t(($) => $.team_control.actions.severity)}</FieldLabel>
                <Input id="team-control-severity" placeholder={t(($) => $.team_control.actions.severity_placeholder)} value={form.valueA} onChange={(event) => set("valueA", event.target.value)} required />
              </Field>
              <Field>
                <FieldLabel htmlFor="team-control-reproduction">{t(($) => $.team_control.actions.reproduction)}</FieldLabel>
                <Textarea id="team-control-reproduction" value={form.detail} onChange={(event) => set("detail", event.target.value)} required />
              </Field>
            </>
          ) : null}
          {action === "risk" ? (
            <>
              <div className="grid gap-4 sm:grid-cols-2">
                <Field>
                  <FieldLabel htmlFor="team-control-probability">{t(($) => $.team_control.actions.probability)}</FieldLabel>
                  <Input id="team-control-probability" type="number" min="1" max="5" value={form.valueA} onChange={(event) => set("valueA", event.target.value)} required />
                </Field>
                <Field>
                  <FieldLabel htmlFor="team-control-impact">{t(($) => $.team_control.actions.impact)}</FieldLabel>
                  <Input id="team-control-impact" type="number" min="1" max="5" value={form.valueB} onChange={(event) => set("valueB", event.target.value)} required />
                </Field>
              </div>
              <Field>
                <FieldLabel htmlFor="team-control-response">{t(($) => $.team_control.actions.response_plan)}</FieldLabel>
                <Textarea id="team-control-response" value={form.detail} onChange={(event) => set("detail", event.target.value)} required />
              </Field>
              <Field>
                <FieldLabel htmlFor="team-control-due">{t(($) => $.team_control.actions.review_due)}</FieldLabel>
                <Input id="team-control-due" type="datetime-local" value={form.dueAt} onChange={(event) => set("dueAt", event.target.value)} required />
              </Field>
            </>
          ) : null}
          {action === "finding" ? (
            <Field>
              <FieldLabel htmlFor="team-control-rule">{t(($) => $.team_control.actions.rule_id)}</FieldLabel>
              <Input id="team-control-rule" value={form.valueA} onChange={(event) => set("valueA", event.target.value)} required />
            </Field>
          ) : null}
          {action === "knowledge" ? (
            <>
              <Field>
                <FieldLabel htmlFor="team-control-dedup">{t(($) => $.team_control.actions.dedup_key)}</FieldLabel>
                <Input id="team-control-dedup" value={form.detail} onChange={(event) => set("detail", event.target.value)} required />
              </Field>
              <Field>
                <FieldLabel htmlFor="team-control-sources">{t(($) => $.team_control.actions.source_ids)}</FieldLabel>
                <Input id="team-control-sources" placeholder={t(($) => $.team_control.actions.source_ids_placeholder)} value={form.references} onChange={(event) => set("references", event.target.value)} required />
              </Field>
              <Field>
                <FieldLabel htmlFor="team-control-evidence">{t(($) => $.team_control.actions.evidence_ids)}</FieldLabel>
                <Input id="team-control-evidence" placeholder={t(($) => $.team_control.actions.evidence_ids_placeholder)} value={form.extra} onChange={(event) => set("extra", event.target.value)} required />
              </Field>
            </>
          ) : null}
          {action === "run" ? (
            <>
              <Field>
                <FieldLabel htmlFor="team-control-workspace-ref">{t(($) => $.team_control.actions.workspace_reference)}</FieldLabel>
                <Input id="team-control-workspace-ref" placeholder={t(($) => $.team_control.actions.workspace_reference_placeholder)} value={form.detail} onChange={(event) => set("detail", event.target.value)} required />
              </Field>
              <Field>
                <FieldLabel htmlFor="team-control-attempts">{t(($) => $.team_control.actions.maximum_attempts)}</FieldLabel>
                <Input id="team-control-attempts" type="number" min="1" value={form.valueA} onChange={(event) => set("valueA", event.target.value)} required />
              </Field>
              <Field>
                <FieldLabel htmlFor="team-control-secrets">{t(($) => $.team_control.actions.secret_references)}</FieldLabel>
                <Input id="team-control-secrets" placeholder={t(($) => $.team_control.actions.secret_references_placeholder)} value={form.references} onChange={(event) => set("references", event.target.value)} />
                <FieldDescription>{t(($) => $.team_control.actions.secret_references_description)}</FieldDescription>
              </Field>
            </>
          ) : null}
        </FieldGroup>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose} disabled={pending}>{t(($) => $.team_control.actions.cancel)}</Button>
          <Button type="button" onClick={onSubmit} disabled={pending || !canSubmit}>
            {pending ? <Spinner data-icon="inline-start" /> : <CheckCircle2 data-icon="inline-start" />}
            {t(($) => $.team_control.actions.submit)}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
