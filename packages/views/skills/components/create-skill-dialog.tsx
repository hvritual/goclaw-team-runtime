"use client";

import { useRef, useState } from "react";
import {
  AlertCircle,
  ArrowLeft,
  ChevronRight,
  Download,
  Loader2,
  Pencil,
  Plus,
  X as XIcon,
} from "lucide-react";
import { toast } from "sonner";
import { useQueryClient } from "@tanstack/react-query";
import { api } from "@multica/core/api";
import type { Skill, SkillImportPreview, SkillSummary } from "@multica/core/types";
import { useWorkspaceId } from "@multica/core/hooks";
import { isImeComposing } from "@multica/core/utils";
import {
  skillDetailOptions,
  workspaceKeys,
} from "@multica/core/workspace/queries";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@multica/ui/components/ui/dialog";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@multica/ui/components/ui/tooltip";
import { Button } from "@multica/ui/components/ui/button";
import { Input } from "@multica/ui/components/ui/input";
import { Label } from "@multica/ui/components/ui/label";
import { Textarea } from "@multica/ui/components/ui/textarea";
import { useScrollFade } from "@multica/ui/hooks/use-scroll-fade";
import { cn } from "@multica/ui/lib/utils";
import { openExternal } from "../../platform";
import { useT } from "../../i18n";
import { isNameConflictError } from "../lib/utils";

type Method = "chooser" | "manual" | "archive" | "url";
type CreateMethod = Exclude<Method, "chooser">;

export function availableCreateMethods(allowImport: boolean): CreateMethod[] {
  return allowImport ? ["manual", "archive", "url"] : ["manual"];
}

function seedAfterCreate(
  qc: ReturnType<typeof useQueryClient>,
  wsId: string,
  skill: Skill,
) {
  qc.setQueryData(skillDetailOptions(wsId, skill.id).queryKey, skill);
  qc.invalidateQueries({ queryKey: workspaceKeys.skills(wsId) });
}

function MethodChooser({ onChoose, allowImport }: { onChoose: (m: Method) => void; allowImport: boolean }) {
  const { t } = useT("skills");
  const methods: {
    key: Method;
    icon: typeof Plus;
    titleKey: "manual" | "archive" | "url";
  }[] = availableCreateMethods(allowImport).map((key) => ({
    key,
    icon: key === "manual" ? Plus : Download,
    titleKey: key,
  }));
  return (
    <div className="grid gap-2 p-5">
      {methods.map(({ key, icon: Icon, titleKey }) => (
        <button
          key={key}
          type="button"
          onClick={() => onChoose(key)}
          className="group flex items-start gap-3 rounded-lg border bg-card p-4 text-left transition-colors hover:border-primary/40 hover:bg-accent/40"
        >
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground group-hover:text-foreground">
            <Icon className="h-4 w-4" />
          </div>
          <div className="min-w-0 flex-1">
            <div className="text-sm font-medium">
              {t(($) => $.create.method_card[`${titleKey}_title`])}
            </div>
            <div className="mt-0.5 text-xs text-muted-foreground">
              {t(($) => $.create.method_card[`${titleKey}_desc`])}
            </div>
          </div>
          <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground/40 transition-colors group-hover:text-muted-foreground" />
        </button>
      ))}
    </div>
  );
}

function ManualForm({
  onCreated,
  onCancel,
}: {
  onCreated: (skill: Skill) => void;
  onCancel: () => void;
}) {
  const { t } = useT("skills");
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const scrollRef = useRef<HTMLDivElement>(null);
  const fadeStyle = useScrollFade(scrollRef);

  const submit = async () => {
    const trimmed = name.trim();
    if (!trimmed) return;
    setLoading(true);
    setError("");
    try {
      const skill = await api.createSkill({
        name: trimmed,
        description: description.trim(),
      });
      seedAfterCreate(qc, wsId, skill);
      toast.success(t(($) => $.create.manual.toast_created));
      onCreated(skill);
    } catch (err) {
      setError(err instanceof Error ? err.message : t(($) => $.create.manual.fallback_error));
      setLoading(false);
    }
  };

  return (
    <>
      <div
        ref={scrollRef}
        style={fadeStyle}
        className="flex-1 min-h-0 space-y-4 overflow-y-auto px-5 py-4"
      >
        <div className="space-y-1.5">
          <Label
            htmlFor="create-skill-name"
            className="text-xs text-muted-foreground"
          >
            {t(($) => $.create.manual.name_label)}
          </Label>
          <Input
            id="create-skill-name"
            autoFocus
            value={name}
            onChange={(e) => {
              setName(e.target.value);
              setError("");
            }}
            placeholder={t(($) => $.create.manual.name_placeholder)}
            onKeyDown={(e) => {
              if (isImeComposing(e)) return;
              if (e.key === "Enter") submit();
            }}
          />
          <p className="text-xs text-muted-foreground">
            {t(($) => $.create.manual.name_hint)}
          </p>
        </div>

        <div className="space-y-1.5">
          <Label
            htmlFor="create-skill-desc"
            className="text-xs text-muted-foreground"
          >
            <Pencil className="h-3 w-3" />
            {t(($) => $.create.manual.description_label)}
          </Label>
          <Textarea
            id="create-skill-desc"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder={t(($) => $.create.manual.description_placeholder)}
            rows={3}
            className="resize-none"
          />
        </div>

        {error && (
          <div
            role="alert"
            className="flex items-start gap-2 rounded-md bg-destructive/10 px-3 py-2 text-xs text-destructive"
          >
            <AlertCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
            <span>
              {error}
              {isNameConflictError(error) && (
                <>{t(($) => $.create.manual.name_conflict_hint)}</>
              )}
            </span>
          </div>
        )}
      </div>

      <div className="flex shrink-0 items-center justify-end gap-2 border-t bg-muted/30 px-5 py-3">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={onCancel}
          disabled={loading}
        >
          {t(($) => $.create.manual.cancel)}
        </Button>
        <Button
          type="button"
          size="sm"
          onClick={submit}
          disabled={!name.trim() || loading}
        >
          {loading ? (
            <>
              <Loader2 className="h-3 w-3 animate-spin" />
              {t(($) => $.create.manual.submitting)}
            </>
          ) : (
            t(($) => $.create.manual.submit)
          )}
        </Button>
      </div>
    </>
  );
}

type DetectedSource = "clawhub" | "skills.sh" | "github" | null;
type ConflictMode = "new_version" | "replace";

function PreviewSummary({
  preview,
  existing,
  conflictMode,
  replaceConfirmed,
  onConflictMode,
  onReplaceConfirmed,
}: {
  preview: SkillImportPreview;
  existing?: SkillSummary;
  conflictMode: ConflictMode;
  replaceConfirmed: boolean;
  onConflictMode: (mode: ConflictMode) => void;
  onReplaceConfirmed: (confirmed: boolean) => void;
}) {
  const { t } = useT("skills");
  return (
    <div className="space-y-3 rounded-md border bg-muted/20 p-3" data-testid="skill-import-preview">
      <div>
        <div className="text-sm font-medium">{preview.name}</div>
        {preview.description && <p className="mt-0.5 text-xs text-muted-foreground">{preview.description}</p>}
      </div>
      <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
        <span>{t(($) => $.create.import_common.files, { count: preview.files.length })}</span>
        <span>{t(($) => $.create.import_common.bytes, { count: preview.total_bytes })}</span>
      </div>
      <div className="max-h-28 space-y-1 overflow-y-auto rounded border bg-background p-2 font-mono text-[11px]">
        {preview.files.map((file) => <div key={file.path} className="truncate">{file.path}</div>)}
      </div>
      {existing && (
        <div className="space-y-2 border-t pt-3">
          <p className="text-xs font-medium">{t(($) => $.create.import_common.conflict_label)}</p>
          <label className="flex items-start gap-2 text-xs">
            <input type="radio" checked={conflictMode === "new_version"} onChange={() => onConflictMode("new_version")} />
            <span>{t(($) => $.create.import_common.new_version)}</span>
          </label>
          <label className="flex items-start gap-2 text-xs">
            <input type="radio" checked={conflictMode === "replace"} onChange={() => onConflictMode("replace")} />
            <span>{t(($) => $.create.import_common.replace)}</span>
          </label>
          {conflictMode === "replace" && (
            <label className="flex items-start gap-2 rounded bg-destructive/10 p-2 text-xs text-destructive">
              <input type="checkbox" checked={replaceConfirmed} onChange={(event) => onReplaceConfirmed(event.target.checked)} />
              <span>{t(($) => $.create.import_common.replace_confirm)}</span>
            </label>
          )}
        </div>
      )}
    </div>
  );
}

function detectUrlSource(url: string): DetectedSource {
  const u = url.trim().toLowerCase();
  if (u.includes("clawhub.ai")) return "clawhub";
  if (u.includes("skills.sh")) return "skills.sh";
  if (u.includes("github.com")) return "github";
  return null;
}

function SourceCard({
  label,
  exampleHost,
  browseUrl,
  active,
}: {
  label: string;
  exampleHost: string;
  browseUrl: string;
  active: boolean;
}) {
  return (
    <div
      className={`rounded-md border px-3 py-2.5 transition-colors ${
        active ? "border-primary bg-primary/5" : ""
      }`}
    >
      <div className="text-xs font-medium">{label}</div>
      <button
        type="button"
        onClick={() => openExternal(browseUrl)}
        className="mt-0.5 block max-w-full truncate text-left font-mono text-xs text-brand underline decoration-brand/40 underline-offset-2 hover:decoration-brand"
      >
        {exampleHost}
      </button>
    </div>
  );
}

function UrlForm({
  onCreated,
  onCancel,
}: {
  onCreated: (skill: Skill) => void;
  onCancel: () => void;
}) {
  const { t } = useT("skills");
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  const [url, setUrl] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [preview, setPreview] = useState<SkillImportPreview>();
  const [existing, setExisting] = useState<SkillSummary>();
  const [conflictMode, setConflictMode] = useState<ConflictMode>("new_version");
  const [replaceConfirmed, setReplaceConfirmed] = useState(false);
  const abortRef = useRef<AbortController | undefined>(undefined);
  const source = detectUrlSource(url);
  const scrollRef = useRef<HTMLDivElement>(null);
  const fadeStyle = useScrollFade(scrollRef);

  const loadPreview = async () => {
    const trimmed = url.trim();
    if (!trimmed) return;
    setLoading(true);
    setError("");
    const controller = new AbortController();
    abortRef.current = controller;
    try {
      const [nextPreview, skills] = await Promise.all([
        api.previewSkillImportURL(trimmed, controller.signal),
        api.listSkills(),
      ]);
      setPreview(nextPreview);
      setExisting(skills.find((skill) => skill.name === nextPreview.name && !skill.archived));
      setConflictMode("new_version");
      setReplaceConfirmed(false);
    } catch (err) {
      if ((err as Error)?.name !== "AbortError") setError(err instanceof Error ? err.message : t(($) => $.create.url.fallback_error));
    } finally {
      setLoading(false);
      abortRef.current = undefined;
    }
  };

  const submit = async () => {
    const trimmed = url.trim();
    if (!trimmed || !preview) return;
    setLoading(true);
    setError("");
    const controller = new AbortController();
    abortRef.current = controller;
    try {
      const skill = await api.commitSkillImportURL(trimmed, preview.preview_token, conflictMode, conflictMode === "replace" ? existing?.revision ?? 0 : 0, controller.signal);
      seedAfterCreate(qc, wsId, skill);
      toast.success(t(($) => $.create.url.toast_imported));
      onCreated(skill);
    } catch (err) {
      if ((err as Error)?.name !== "AbortError") setError(err instanceof Error ? err.message : t(($) => $.create.url.fallback_error));
    } finally {
      setLoading(false);
      abortRef.current = undefined;
    }
  };

  const submittingLabel = (() => {
    if (!loading) return t(($) => $.create.url.import);
    if (source === "clawhub") return t(($) => $.create.url.importing_clawhub);
    if (source === "skills.sh") return t(($) => $.create.url.importing_skills_sh);
    if (source === "github") return t(($) => $.create.url.importing_github);
    return t(($) => $.create.url.importing);
  })();

  return (
    <>
      <div
        ref={scrollRef}
        style={fadeStyle}
        className="flex-1 min-h-0 space-y-4 overflow-y-auto px-5 py-4"
      >
        <div className="space-y-1.5">
          <Label htmlFor="import-url" className="text-xs text-muted-foreground">
            {t(($) => $.create.url.url_label)}
          </Label>
          <Input
            id="import-url"
            autoFocus
            value={url}
            onChange={(e) => {
              setUrl(e.target.value);
              setError("");
              setPreview(undefined);
              setExisting(undefined);
            }}
            placeholder="https://clawhub.ai/owner/skill"
            className="font-mono text-sm"
            onKeyDown={(e) => {
              if (e.key !== "Enter") return;
              if (preview) {
                void submit();
                return;
              }
              void loadPreview();
            }}
          />
        </div>

        {preview && (
          <PreviewSummary preview={preview} existing={existing} conflictMode={conflictMode} replaceConfirmed={replaceConfirmed} onConflictMode={(mode) => { setConflictMode(mode); setReplaceConfirmed(false); }} onReplaceConfirmed={setReplaceConfirmed} />
        )}

        <div>
          <p className="mb-2 text-xs text-muted-foreground">
            {t(($) => $.create.url.supported_sources)}
          </p>
          <div className="grid grid-cols-3 gap-2">
            <SourceCard
              label="ClawHub"
              exampleHost="clawhub.ai/owner/skill"
              browseUrl="https://clawhub.ai"
              active={source === "clawhub"}
            />
            <SourceCard
              label="Skills.sh"
              exampleHost="skills.sh/owner/repo/skill"
              browseUrl="https://skills.sh"
              active={source === "skills.sh"}
            />
            <SourceCard
              label="GitHub"
              exampleHost="github.com/owner/repo"
              browseUrl="https://github.com"
              active={source === "github"}
            />
          </div>
        </div>

        {error && (
          <div
            role="alert"
            className="flex items-start gap-2 rounded-md bg-destructive/10 px-3 py-2 text-xs text-destructive"
          >
            <AlertCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
            <span>
              {error}
              {isNameConflictError(error) && (
                <>{t(($) => $.create.url.name_conflict_hint)}</>
              )}
            </span>
          </div>
        )}
      </div>

      <div className="flex shrink-0 items-center justify-end gap-2 border-t bg-muted/30 px-5 py-3">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={onCancel}
          onMouseDown={() => {
            if (loading) abortRef.current?.abort();
          }}
        >
          {loading ? t(($) => $.create.import_common.cancel_operation) : t(($) => $.create.url.cancel)}
        </Button>
        <Button
          type="button"
          size="sm"
          onClick={preview ? submit : loadPreview}
          disabled={!url.trim() || loading || (conflictMode === "replace" && !replaceConfirmed)}
        >
          {loading ? (
            <>
              <Loader2 className="h-3 w-3 animate-spin" />
              {preview ? submittingLabel : t(($) => $.create.import_common.previewing)}
            </>
          ) : (
            <>
              <Download className="h-3 w-3" />
              {preview ? submittingLabel : t(($) => $.create.import_common.preview)}
            </>
          )}
        </Button>
      </div>
    </>
  );
}

function ArchiveForm({
  onCreated,
  onCancel,
}: {
  onCreated: (skill: Skill) => void;
  onCancel: () => void;
}) {
  const { t } = useT("skills");
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  const [file, setFile] = useState<File>();
  const [preview, setPreview] = useState<SkillImportPreview>();
  const [existing, setExisting] = useState<SkillSummary>();
  const [conflictMode, setConflictMode] = useState<ConflictMode>("new_version");
  const [replaceConfirmed, setReplaceConfirmed] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const abortRef = useRef<AbortController | undefined>(undefined);

  const loadPreview = async () => {
    if (!file) return;
    setLoading(true);
    setError("");
    const controller = new AbortController();
    abortRef.current = controller;
    try {
      const [nextPreview, skills] = await Promise.all([
        api.previewSkillImportArchive(file, controller.signal),
        api.listSkills(),
      ]);
      setPreview(nextPreview);
      setExisting(skills.find((skill) => skill.name === nextPreview.name && !skill.archived));
      setConflictMode("new_version");
      setReplaceConfirmed(false);
    } catch (err) {
      if ((err as Error)?.name !== "AbortError") setError(err instanceof Error ? err.message : t(($) => $.create.archive.fallback_error));
    } finally {
      setLoading(false);
      abortRef.current = undefined;
    }
  };

  const submit = async () => {
    if (!file || !preview) return;
    setLoading(true);
    setError("");
    const controller = new AbortController();
    abortRef.current = controller;
    try {
      const skill = await api.commitSkillImportArchive(file, preview.preview_token, conflictMode, conflictMode === "replace" ? existing?.revision ?? 0 : 0, controller.signal);
      seedAfterCreate(qc, wsId, skill);
      toast.success(t(($) => $.create.archive.toast_imported));
      onCreated(skill);
    } catch (err) {
      if ((err as Error)?.name !== "AbortError") setError(err instanceof Error ? err.message : t(($) => $.create.archive.fallback_error));
    } finally {
      setLoading(false);
      abortRef.current = undefined;
    }
  };

  return (
    <>
      <div className="flex-1 min-h-0 space-y-4 overflow-y-auto px-5 py-4">
        <div className="space-y-1.5">
          <Label htmlFor="import-archive" className="text-xs text-muted-foreground">{t(($) => $.create.archive.file_label)}</Label>
          <Input
            id="import-archive"
            type="file"
            accept=".zip,.skill,application/zip"
            onChange={(event) => {
              setFile(event.target.files?.[0]);
              setPreview(undefined);
              setExisting(undefined);
              setError("");
            }}
          />
          <p className="text-xs text-muted-foreground">{t(($) => $.create.archive.file_hint)}</p>
        </div>
        {preview && (
          <PreviewSummary preview={preview} existing={existing} conflictMode={conflictMode} replaceConfirmed={replaceConfirmed} onConflictMode={(mode) => { setConflictMode(mode); setReplaceConfirmed(false); }} onReplaceConfirmed={setReplaceConfirmed} />
        )}
        {error && (
          <div role="alert" className="flex items-start gap-2 rounded-md bg-destructive/10 px-3 py-2 text-xs text-destructive">
            <AlertCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
            <span>{error}</span>
          </div>
        )}
      </div>
      <div className="flex shrink-0 items-center justify-end gap-2 border-t bg-muted/30 px-5 py-3">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={onCancel}
          onMouseDown={() => {
            if (loading) abortRef.current?.abort();
          }}
        >
          {loading ? t(($) => $.create.import_common.cancel_operation) : t(($) => $.create.archive.cancel)}
        </Button>
        <Button type="button" size="sm" onClick={preview ? submit : loadPreview} disabled={!file || loading || (conflictMode === "replace" && !replaceConfirmed)}>
          {loading && <Loader2 className="h-3 w-3 animate-spin" />}
          {!loading && <Download className="h-3 w-3" />}
          {loading ? (preview ? t(($) => $.create.archive.importing) : t(($) => $.create.import_common.previewing)) : (preview ? t(($) => $.create.archive.import) : t(($) => $.create.import_common.preview))}
        </Button>
      </div>
    </>
  );
}

export function CreateSkillDialog({
  onClose,
  onCreated,
  allowImport = false,
}: {
  onClose: () => void;
  onCreated?: (skill: Skill) => void;
  allowImport?: boolean;
}) {
  const { t } = useT("skills");
  const [method, setMethod] = useState<Method>("chooser");

  const handleCreated = (skill: Skill) => {
    onCreated?.(skill);
    onClose();
  };

  return (
    <Dialog open onOpenChange={(v) => !v && onClose()}>
      <DialogContent
        showCloseButton={false}
        className={cn(
          "flex flex-col gap-0 overflow-hidden p-0",
          "!transition-all !duration-300 !ease-out",
          "!h-auto !max-h-[85vh] !max-w-md !w-full",
        )}
      >
        <div className="flex shrink-0 items-start justify-between gap-3 border-b px-5 pt-4 pb-3">
          <div className="flex items-center gap-2 min-w-0">
            {method !== "chooser" && (
              <Tooltip>
                <TooltipTrigger
                  render={
                    <button
                      type="button"
                      onClick={() => setMethod("chooser")}
                      className="-ml-1 rounded-sm p-1 text-muted-foreground opacity-70 transition-opacity hover:bg-accent/60 hover:opacity-100"
                      aria-label={t(($) => $.create.back_aria)}
                    >
                      <ArrowLeft className="h-3.5 w-3.5" />
                    </button>
                  }
                />
                <TooltipContent side="bottom">{t(($) => $.create.back)}</TooltipContent>
              </Tooltip>
            )}
            <div className="min-w-0">
              <DialogTitle className="truncate text-base font-medium">
                {t(($) => $.create.method[method].title)}
              </DialogTitle>
              <p className="mt-0.5 text-xs text-muted-foreground">
                {t(($) => $.create.method[method].desc)}
              </p>
            </div>
          </div>
          <Tooltip>
            <TooltipTrigger
              render={
                <button
                  type="button"
                  onClick={onClose}
                  className="rounded-sm p-1 text-muted-foreground opacity-70 transition-opacity hover:bg-accent/60 hover:opacity-100"
                  aria-label={t(($) => $.create.close_aria)}
                >
                  <XIcon className="h-3.5 w-3.5" />
                </button>
              }
            />
            <TooltipContent side="bottom">{t(($) => $.create.close)}</TooltipContent>
          </Tooltip>
        </div>

        {method === "chooser" && <MethodChooser onChoose={setMethod} allowImport={allowImport} />}
        {method === "manual" && (
          <ManualForm
            onCreated={handleCreated}
            onCancel={() => setMethod("chooser")}
          />
        )}
        {allowImport && method === "archive" && (
          <ArchiveForm
            onCreated={handleCreated}
            onCancel={() => setMethod("chooser")}
          />
        )}
        {allowImport && method === "url" && (
          <UrlForm
            onCreated={handleCreated}
            onCancel={() => setMethod("chooser")}
          />
        )}
      </DialogContent>
    </Dialog>
  );
}