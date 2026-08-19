"use client";

import { useEffect, useMemo, useState } from "react";
import type {
  AcceptanceConclusionInput,
  AcceptanceResult,
  MemberWithUser,
  ProjectRetrospectiveInput,
} from "@multica/core/types";
import type {
  ProjectRetrospectiveActionItemInput,
  ProjectRetrospectiveContentInput,
  ProjectRetrospectiveParticipantInput,
} from "@multica/core/types/implementation-knowledge";
import { Button } from "@multica/ui/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@multica/ui/components/ui/dialog";
import { Label } from "@multica/ui/components/ui/label";
import { Input } from "@multica/ui/components/ui/input";
import {
  NativeSelect,
  NativeSelectOption,
} from "@multica/ui/components/ui/native-select";
import { Textarea } from "@multica/ui/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@multica/ui/components/ui/select";
import { useT } from "../i18n";

function lines(value: string): string[] {
  return value.split("\n").map((item) => item.trim()).filter(Boolean);
}

const EMPTY_RETROSPECTIVE_MEMBERS: MemberWithUser[] = [];
const EMPTY_RETROSPECTIVE_PARTICIPANTS: ProjectRetrospectiveParticipantInput[] = [];

export function AcceptanceConclusionDialog({
  open,
  onOpenChange,
  onSubmit,
  pending = false,
  mode = "complete",
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (input: AcceptanceConclusionInput | null) => void;
  pending?: boolean;
  mode?: "complete" | "capture";
}) {
  const { t } = useT("issues");
  const [result, setResult] = useState<AcceptanceResult>("accepted");
  const [rationale, setRationale] = useState("");
  const [evidenceRefs, setEvidenceRefs] = useState("");
  const resultOptions = [
    { value: "accepted" as const, label: t(($) => $.implementation_knowledge.result_accepted) },
    { value: "conditional" as const, label: t(($) => $.implementation_knowledge.result_conditional) },
    { value: "rejected" as const, label: t(($) => $.implementation_knowledge.result_rejected) },
  ];

  useEffect(() => {
    if (!open) {
      setResult("accepted");
      setRationale("");
      setEvidenceRefs("");
    }
  }, [open]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{mode === "capture"
            ? t(($) => $.implementation_knowledge.capture_title)
            : t(($) => $.implementation_knowledge.complete_title)}</DialogTitle>
          <DialogDescription>{mode === "capture"
            ? t(($) => $.implementation_knowledge.capture_description)
            : t(($) => $.implementation_knowledge.complete_description)}</DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="acceptance-result">{t(($) => $.implementation_knowledge.result_label)}</Label>
            <Select
              items={resultOptions}
              value={result}
              onValueChange={(value) => value && setResult(value as AcceptanceResult)}
            >
              <SelectTrigger id="acceptance-result" className="w-full" aria-label={t(($) => $.implementation_knowledge.result_label)}>
                <SelectValue>{resultOptions.find((option) => option.value === result)?.label}</SelectValue>
              </SelectTrigger>
              <SelectContent>
                {resultOptions.map((option) => (
                  <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="acceptance-rationale">{t(($) => $.implementation_knowledge.rationale_label)}</Label>
            <Textarea id="acceptance-rationale" value={rationale} onChange={(event) => setRationale(event.target.value)} />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="acceptance-evidence">{t(($) => $.implementation_knowledge.evidence_label)}</Label>
            <Textarea
              id="acceptance-evidence"
              value={evidenceRefs}
              onChange={(event) => setEvidenceRefs(event.target.value)}
              placeholder={t(($) => $.implementation_knowledge.evidence_placeholder)}
            />
          </div>
        </div>
        <DialogFooter>
          {mode === "complete" ? (
            <Button variant="outline" disabled={pending} onClick={() => onSubmit(null)}>{t(($) => $.implementation_knowledge.complete_directly)}</Button>
          ) : null}
          <Button
            disabled={pending || !rationale.trim()}
            onClick={() => onSubmit({ result, rationale: rationale.trim(), evidenceRefs: lines(evidenceRefs) })}
          >
            {mode === "capture"
              ? t(($) => $.implementation_knowledge.capture_submit)
              : t(($) => $.implementation_knowledge.complete_and_capture)}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function ProjectRetrospectiveDialog({
  open,
  onOpenChange,
  onSubmit,
  pending = false,
  mode = "create",
  members = EMPTY_RETROSPECTIVE_MEMBERS,
  initialContent,
  initialParticipants = EMPTY_RETROSPECTIVE_PARTICIPANTS,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (input: ProjectRetrospectiveInput) => void;
  pending?: boolean;
  mode?: "create" | "save_draft" | "publish_revision";
  members?: MemberWithUser[];
  initialContent?: ProjectRetrospectiveContentInput;
  initialParticipants?: ProjectRetrospectiveParticipantInput[];
}) {
  const { t } = useT("projects");
  const [summary, setSummary] = useState("");
  const [successes, setSuccesses] = useState("");
  const [problems, setProblems] = useState("");
  const [lessons, setLessons] = useState("");
  const [actionItems, setActionItems] = useState<ProjectRetrospectiveActionItemInput[]>([]);
  const [participants, setParticipants] = useState<ProjectRetrospectiveParticipantInput[]>([]);

  const memberOptions = useMemo(() => {
    const options = new Map(members.map((member) => [member.id, member.name]));
    for (const participant of initialParticipants) {
      if (!options.has(participant.memberId)) options.set(participant.memberId, participant.memberId);
    }
    for (const actionItem of initialContent?.actionItems ?? []) {
      if (actionItem.assigneeId && !options.has(actionItem.assigneeId)) {
        options.set(actionItem.assigneeId, actionItem.assigneeId);
      }
    }
    return [...options].map(([id, name]) => ({ id, name }));
  }, [initialContent, initialParticipants, members]);

  useEffect(() => {
    setSummary(open ? initialContent?.summary ?? "" : "");
    setSuccesses(open ? initialContent?.successes.join("\n") ?? "" : "");
    setProblems(open ? initialContent?.problems.join("\n") ?? "" : "");
    setLessons(open ? initialContent?.lessons.join("\n") ?? "" : "");
    setActionItems(open ? initialContent?.actionItems.map((item) => ({ ...item })) ?? [] : []);
    setParticipants(open ? initialParticipants.map((participant) => ({ ...participant })) : []);
  }, [initialContent, initialParticipants, open]);

  const updateActionItem = (
    index: number,
    update: (item: ProjectRetrospectiveActionItemInput) => ProjectRetrospectiveActionItemInput,
  ) => setActionItems((items) => items.map((item, itemIndex) => (
    itemIndex === index ? update(item) : item
  )));

  const submitLabel = mode === "create"
    ? t(($) => $.implementation_knowledge.create_draft)
    : mode === "save_draft"
      ? t(($) => $.implementation_knowledge.save_draft)
      : t(($) => $.implementation_knowledge.save_revision);
  const valid = Boolean(
    summary.trim() &&
    lines(lessons).length > 0 &&
    actionItems.every((item) => item.title.trim()),
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{mode === "create"
            ? t(($) => $.implementation_knowledge.dialog_title)
            : mode === "save_draft"
              ? t(($) => $.implementation_knowledge.dialog_edit_title)
              : t(($) => $.implementation_knowledge.dialog_revision_title)}</DialogTitle>
          <DialogDescription>{t(($) => $.implementation_knowledge.dialog_description)}</DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <Field id="retro-summary" label={t(($) => $.implementation_knowledge.summary)} placeholder={t(($) => $.implementation_knowledge.line_placeholder)} value={summary} onChange={setSummary} />
          <Field id="retro-successes" label={t(($) => $.implementation_knowledge.successes)} placeholder={t(($) => $.implementation_knowledge.line_placeholder)} value={successes} onChange={setSuccesses} />
          <Field id="retro-problems" label={t(($) => $.implementation_knowledge.problems)} placeholder={t(($) => $.implementation_knowledge.line_placeholder)} value={problems} onChange={setProblems} />
          <Field id="retro-lessons" label={t(($) => $.implementation_knowledge.lessons)} placeholder={t(($) => $.implementation_knowledge.line_placeholder)} value={lessons} onChange={setLessons} />

          <fieldset className="space-y-2 rounded-lg border p-3">
            <div className="flex items-center justify-between gap-2">
              <legend className="text-sm font-medium">{t(($) => $.implementation_knowledge.action_items)}</legend>
              <Button
                type="button"
                size="sm"
                variant="outline"
                disabled={actionItems.length >= 100}
                onClick={() => setActionItems((items) => [...items, { title: "" }])}
              >
                {t(($) => $.implementation_knowledge.add_action_item)}
              </Button>
            </div>
            {actionItems.length === 0 ? (
              <p className="text-xs text-muted-foreground">{t(($) => $.implementation_knowledge.action_items_empty)}</p>
            ) : actionItems.map((item, index) => (
              <div key={item.id ?? `new-${index}`} className="space-y-2 rounded-md bg-muted/30 p-2">
                <div className="grid gap-2 sm:grid-cols-2">
                  <div className="space-y-1">
                    <Label htmlFor={`retro-action-title-${index}`}>
                      {t(($) => $.implementation_knowledge.action_title)} {index + 1}
                    </Label>
                    <Input
                      id={`retro-action-title-${index}`}
                      value={item.title}
                      onChange={(event) => updateActionItem(index, (current) => ({
                        ...current,
                        title: event.target.value,
                      }))}
                    />
                  </div>
                  <div className="space-y-1">
                    <Label htmlFor={`retro-action-assignee-${index}`}>
                      {t(($) => $.implementation_knowledge.action_assignee)} {index + 1}
                    </Label>
                    <NativeSelect
                      id={`retro-action-assignee-${index}`}
                      className="w-full"
                      value={item.assigneeId ?? ""}
                      onChange={(event) => updateActionItem(index, (current) => ({
                        ...current,
                        ...(event.target.value
                          ? { assigneeId: event.target.value }
                          : { assigneeId: undefined }),
                      }))}
                    >
                      <NativeSelectOption value="">{t(($) => $.implementation_knowledge.unassigned)}</NativeSelectOption>
                      {memberOptions.map((member) => (
                        <NativeSelectOption key={member.id} value={member.id}>{member.name}</NativeSelectOption>
                      ))}
                    </NativeSelect>
                  </div>
                </div>
                <div className="grid gap-2 sm:grid-cols-[1fr_11rem]">
                  <div className="space-y-1">
                    <Label htmlFor={`retro-action-description-${index}`}>
                      {t(($) => $.implementation_knowledge.action_description)} {index + 1}
                    </Label>
                    <Textarea
                      id={`retro-action-description-${index}`}
                      value={item.description ?? ""}
                      onChange={(event) => updateActionItem(index, (current) => ({
                        ...current,
                        ...(event.target.value
                          ? { description: event.target.value }
                          : { description: undefined }),
                      }))}
                    />
                  </div>
                  <div className="space-y-1">
                    <Label htmlFor={`retro-action-due-${index}`}>
                      {t(($) => $.implementation_knowledge.action_due_date)} {index + 1}
                    </Label>
                    <Input
                      id={`retro-action-due-${index}`}
                      type="date"
                      value={item.dueDate ?? ""}
                      onChange={(event) => updateActionItem(index, (current) => ({
                        ...current,
                        ...(event.target.value
                          ? { dueDate: event.target.value }
                          : { dueDate: undefined }),
                      }))}
                    />
                  </div>
                </div>
                <Button
                  type="button"
                  size="sm"
                  variant="ghost"
                  onClick={() => setActionItems((items) => items.filter((_, itemIndex) => itemIndex !== index))}
                >
                  {t(($) => $.implementation_knowledge.remove_action_item)} {index + 1}
                </Button>
              </div>
            ))}
          </fieldset>

          <fieldset className="space-y-2 rounded-lg border p-3">
            <legend className="text-sm font-medium">{t(($) => $.implementation_knowledge.participants)}</legend>
            {memberOptions.length === 0 ? (
              <p className="text-xs text-muted-foreground">{t(($) => $.implementation_knowledge.participants_empty)}</p>
            ) : memberOptions.map((member) => {
              const participant = participants.find((item) => item.memberId === member.id);
              return (
                <div key={member.id} className="flex flex-wrap items-center justify-between gap-2 rounded-md bg-muted/30 p-2">
                  <label className="flex items-center gap-2 text-sm">
                    <input
                      type="checkbox"
                      aria-label={`${t(($) => $.implementation_knowledge.include_participant)}：${member.name}`}
                      checked={Boolean(participant)}
                      onChange={(event) => setParticipants((items) => event.target.checked
                        ? [...items, { memberId: member.id, role: "participant" }]
                        : items.filter((item) => item.memberId !== member.id))}
                    />
                    {member.name}
                  </label>
                  <NativeSelect
                    aria-label={`${t(($) => $.implementation_knowledge.participant_role)}：${member.name}`}
                    disabled={!participant}
                    value={participant?.role ?? "participant"}
                    onChange={(event) => setParticipants((items) => items.map((item) => (
                      item.memberId === member.id
                        ? { ...item, role: event.target.value as ProjectRetrospectiveParticipantInput["role"] }
                        : item
                    )))}
                  >
                    <NativeSelectOption value="participant">{t(($) => $.implementation_knowledge.role_participant)}</NativeSelectOption>
                    <NativeSelectOption value="facilitator">{t(($) => $.implementation_knowledge.role_facilitator)}</NativeSelectOption>
                  </NativeSelect>
                </div>
              );
            })}
          </fieldset>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>{t(($) => $.implementation_knowledge.cancel)}</Button>
          <Button
            disabled={pending || !valid}
            onClick={() => onSubmit({
              content: {
                summary: summary.trim(),
                successes: lines(successes),
                problems: lines(problems),
                lessons: lines(lessons),
                actionItems: actionItems.map((item) => ({
                  ...(item.id === undefined ? {} : { id: item.id }),
                  title: item.title.trim(),
                  ...(item.description?.trim() ? { description: item.description.trim() } : {}),
                  ...(item.assigneeId ? { assigneeId: item.assigneeId } : {}),
                  ...(item.dueDate ? { dueDate: item.dueDate } : {}),
                })),
              },
              participants,
            })}
          >
            {submitLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function Field({ id, label, placeholder, value, onChange }: {
  id: string;
  label: string;
  placeholder: string;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>{label}</Label>
      <Textarea id={id} value={value} onChange={(event) => onChange(event.target.value)} placeholder={placeholder} />
    </div>
  );
}
