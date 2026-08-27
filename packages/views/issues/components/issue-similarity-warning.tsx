"use client";

import { AlertTriangle, LoaderCircle } from "lucide-react";
import type { IssueSimilarityResponse } from "@multica/core/types";
import { cn } from "@multica/ui/lib/utils";
import { useT } from "../../i18n";

export interface IssueSimilarityWarningProps {
  result?: IssueSimilarityResponse;
  checking?: boolean;
  unavailable?: boolean;
  className?: string;
}

/**
 * Informational-only similarity disclosure. It deliberately exposes neither
 * a confirm action nor a mutation path, so create/edit can always proceed.
 */
export function IssueSimilarityWarning({
  result,
  checking = false,
  unavailable = false,
  className,
}: IssueSimilarityWarningProps) {
  const { t } = useT("issues");
  const detectorUnavailable = unavailable || result?.detector_available === false;

  if (checking) {
    return (
      <div
        aria-live="polite"
        className={cn("flex items-center gap-2 text-xs text-muted-foreground", className)}
        data-testid="issue-similarity-checking"
        role="status"
      >
        <LoaderCircle aria-hidden className="size-3.5 animate-spin" />
        {t(($) => $.similarity.checking)}
      </div>
    );
  }

  if (detectorUnavailable) {
    return (
      <div
        aria-live="polite"
        className={cn("flex items-start gap-2 rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-sm text-foreground", className)}
        role="alert"
      >
        <AlertTriangle aria-hidden className="mt-0.5 size-4 shrink-0 text-amber-600 dark:text-amber-400" />
        <span>{t(($) => $.similarity.unavailable)}</span>
      </div>
    );
  }

  if (!result || result.candidates.length === 0) return null;

  return (
    <section
      aria-label={t(($) => $.similarity.heading)}
      className={cn("rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2", className)}
      role="region"
    >
      <div className="flex items-center gap-2 text-sm font-medium text-foreground">
        <AlertTriangle aria-hidden className="size-4 shrink-0 text-amber-600 dark:text-amber-400" />
        {t(($) => $.similarity.heading)}
      </div>
      <ul className="mt-2 space-y-1.5" data-testid="issue-similarity-candidates">
        {result.candidates.map((candidate) => (
          <li
            className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm"
            key={candidate.id}
          >
            <span className="font-mono text-xs text-muted-foreground">
              {candidate.identifier}
            </span>
            <span className="min-w-0 flex-1 truncate">{candidate.title}</span>
            <span className="text-xs text-muted-foreground">
              {t(($) => $.similarity.score, { score: Math.round(candidate.score) })}
            </span>
            {candidate.same_project ? (
              <span className="rounded bg-background/70 px-1.5 py-0.5 text-xs text-muted-foreground">
                {t(($) => $.similarity.same_project)}
              </span>
            ) : null}
            {candidate.closed ? (
              <span className="rounded bg-background/70 px-1.5 py-0.5 text-xs text-muted-foreground">
                {t(($) => $.similarity.closed)}
              </span>
            ) : null}
          </li>
        ))}
      </ul>
      {result.truncated ? (
        <p className="mt-2 text-xs text-muted-foreground">
          {t(($) => $.similarity.truncated)}
        </p>
      ) : null}
    </section>
  );
}
