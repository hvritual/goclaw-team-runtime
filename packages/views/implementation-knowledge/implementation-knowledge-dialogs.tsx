"use client";

import { useEffect, useState } from "react";
import type {
  AcceptanceConclusionInput,
  AcceptanceResult,
  ProjectRetrospectiveInput,
} from "@multica/core/types";
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
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (input: ProjectRetrospectiveInput) => void;
  pending?: boolean;
}) {
  const { t } = useT("projects");
  const [summary, setSummary] = useState("");
  const [successes, setSuccesses] = useState("");
  const [problems, setProblems] = useState("");
  const [lessons, setLessons] = useState("");
  const [followUps, setFollowUps] = useState("");

  useEffect(() => {
    if (!open) {
      setSummary(""); setSuccesses(""); setProblems(""); setLessons(""); setFollowUps("");
    }
  }, [open]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{t(($) => $.implementation_knowledge.dialog_title)}</DialogTitle>
          <DialogDescription>{t(($) => $.implementation_knowledge.dialog_description)}</DialogDescription>
        </DialogHeader>
        <div className="space-y-3">
          <Field id="retro-summary" label={t(($) => $.implementation_knowledge.summary)} placeholder={t(($) => $.implementation_knowledge.line_placeholder)} value={summary} onChange={setSummary} />
          <Field id="retro-successes" label={t(($) => $.implementation_knowledge.successes)} placeholder={t(($) => $.implementation_knowledge.line_placeholder)} value={successes} onChange={setSuccesses} />
          <Field id="retro-problems" label={t(($) => $.implementation_knowledge.problems)} placeholder={t(($) => $.implementation_knowledge.line_placeholder)} value={problems} onChange={setProblems} />
          <Field id="retro-lessons" label={t(($) => $.implementation_knowledge.lessons)} placeholder={t(($) => $.implementation_knowledge.line_placeholder)} value={lessons} onChange={setLessons} />
          <Field id="retro-follow-ups" label={t(($) => $.implementation_knowledge.follow_ups)} placeholder={t(($) => $.implementation_knowledge.line_placeholder)} value={followUps} onChange={setFollowUps} />
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>{t(($) => $.implementation_knowledge.cancel)}</Button>
          <Button
            disabled={pending || !summary.trim() || lines(lessons).length === 0}
            onClick={() => onSubmit({
              summary: summary.trim(), successes: lines(successes), problems: lines(problems),
              lessons: lines(lessons), followUpRefs: lines(followUps),
            })}
          >
            {t(($) => $.implementation_knowledge.submit)}
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
