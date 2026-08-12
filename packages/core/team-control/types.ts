import type { z } from "zod";
import type {
  TeamControlAcceptanceSchema,
  TeamControlAppendResultSchema,
  TeamControlCheckSchema,
  TeamControlEvidenceSchema,
  TeamControlMemberSchema,
  TeamControlMembersResponseSchema,
  TeamControlProblemSchema,
  TeamControlProjectionSchema,
  TeamControlSessionEventSchema,
  TeamControlWorkspaceResponseSchema,
  TeamControlWorkspaceSchema,
  TeamControlWorkEdgeSchema,
  TeamControlWorkNodeSchema,
} from "./schemas";

export type TeamControlWorkspace = z.infer<typeof TeamControlWorkspaceSchema>;
export type TeamControlWorkspaceResponse = z.infer<typeof TeamControlWorkspaceResponseSchema>;
export type TeamControlMember = z.infer<typeof TeamControlMemberSchema>;
export type TeamControlMembersResponse = z.infer<typeof TeamControlMembersResponseSchema>;
export type TeamControlWorkNode = z.infer<typeof TeamControlWorkNodeSchema>;
export type TeamControlWorkEdge = z.infer<typeof TeamControlWorkEdgeSchema>;
export type TeamControlEvidence = z.infer<typeof TeamControlEvidenceSchema>;
export type TeamControlCheck = z.infer<typeof TeamControlCheckSchema>;
export type TeamControlAcceptance = z.infer<typeof TeamControlAcceptanceSchema>;
export type TeamControlProjection = z.infer<typeof TeamControlProjectionSchema>;
export type TeamControlSessionEvent = z.infer<typeof TeamControlSessionEventSchema>;
export type TeamControlAppendResult = z.infer<typeof TeamControlAppendResultSchema>;
export type TeamControlProblem = z.infer<typeof TeamControlProblemSchema>;

type IDTextPayload = { id: string; text: string };
type DonePayload = { subject_id: string; revision: number; policies?: string[] };

export type TeamControlCommand =
  | { type: "requirement.start" | "requirement.intent" | "requirement.solution" | "requirement.change"; payload: IDTextPayload }
  | { type: "requirement.freeze" | "quality.close" | "finding.resolve" | "knowledge.publish"; payload: DonePayload }
  | { type: "requirement.task"; payload: { requirement_id: string; task_id: string; assignee_id: string; edge_command_id: string } }
  | { type: "defect.create"; payload: { id: string; data: { summary: string; severity: string; reproduction: string } } }
  | { type: "risk.create"; payload: { id: string; data: { summary: string; probability: number; impact: number; response_plan: string; review_due_at: string } } }
  | { type: "finding.create"; payload: { id: string; data: { rule_id: string; summary: string; model_finding: boolean } } }
  | { type: "knowledge.create"; payload: { id: string; data: { title: string; source_ids: string[]; evidence_ids: string[]; dedup_key: string } } }
  | { type: "knowledge.invalidate" | "run.complete" | "run.cancel" | "run.retry"; payload: { id: string } }
  | { type: "run.queue"; payload: { id: string; workspace_ref: string; secret_refs: string[]; max_attempts: number } }
  | { type: "run.claim" | "run.heartbeat"; payload: { id: string; lease_seconds: number } }
  | { type: "evidence.attach"; payload: TeamControlEvidence }
  | { type: "check.record"; payload: TeamControlCheck }
  | { type: "done.accept"; payload: Required<DonePayload> };

export type TeamControlCommandInput = TeamControlCommand & {
  commandId?: string;
  expectedHead: number;
};

export type TeamControlConnectionState =
  | "connecting"
  | "connected"
  | "reconnecting"
  | "offline";
