import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../api";
import { useWorkspaceId } from "../hooks";

export interface UseIssueSimilarityInput {
  /** Existing Issue id selects the canonical, self-excluding endpoint. */
  issueId?: string;
  title: string;
  description?: string | null;
  projectId?: string | null;
  /** Increments after a canonical Issue save to bypass stale result reuse. */
  refreshKey?: number;
  enabled?: boolean;
  /** Test seam; production callers use the bounded default. */
  debounceMs?: number;
}

interface NormalizedIssueSimilarityInput {
  issueId?: string;
  title: string;
  description?: string;
  projectId?: string;
  refreshKey: number;
  key: string;
}

const DEFAULT_DEBOUNCE_MS = 350;

function normalizeInput(input: UseIssueSimilarityInput): NormalizedIssueSimilarityInput {
  const issueId = input.issueId?.trim() || undefined;
  const title = input.title.trim();
  const description = input.description?.trim() || undefined;
  const projectId = input.projectId?.trim() || undefined;
  const refreshKey = input.refreshKey ?? 0;
  return {
    issueId,
    title,
    description,
    projectId,
    refreshKey,
    key: JSON.stringify([issueId ?? null, title, description ?? null, projectId ?? null, refreshKey]),
  };
}

function useDebouncedSimilarityInput(
  input: NormalizedIssueSimilarityInput,
  debounceMs: number,
): NormalizedIssueSimilarityInput | null {
  const [debounced, setDebounced] = useState<NormalizedIssueSimilarityInput | null>(null);

  useEffect(() => {
    const timer = window.setTimeout(() => setDebounced(input), debounceMs);
    return () => window.clearTimeout(timer);
  }, [input.issueId, input.title, input.description, input.projectId, input.refreshKey, input.key, debounceMs]);

  return debounced;
}

/**
 * Workspace-local, warning-only similarity query. The result is intentionally
 * independent of Issue create/update mutations: callers can always continue
 * their requested write while the detector is pending or unavailable.
 */
export function useIssueSimilarity(input: UseIssueSimilarityInput) {
  const workspaceId = useWorkspaceId();
  const normalized = normalizeInput(input);
  const debounced = useDebouncedSimilarityInput(
    normalized,
    input.debounceMs ?? DEFAULT_DEBOUNCE_MS,
  );
  const ready = input.enabled !== false && !!debounced?.title;
  const query = useQuery({
    // The API resolves its workspace identity from the active route/session,
    // so the cache must be partitioned by the same workspace boundary.
    queryKey: ["issue-similarity", workspaceId, debounced?.key ?? "pending"],
    queryFn: () => {
      if (!debounced) throw new Error("Issue similarity request is not ready");
      if (debounced.issueId) return api.checkExistingIssueSimilarity(debounced.issueId);
      return api.checkIssueSimilarity({
        title: debounced.title,
        ...(debounced.description ? { description: debounced.description } : {}),
        ...(debounced.projectId ? { project_id: debounced.projectId } : {}),
      });
    },
    enabled: ready,
    retry: false,
    staleTime: 30_000,
    refetchOnWindowFocus: false,
  });

  return {
    ...query,
    inputPending: normalized.key !== debounced?.key,
  };
}
