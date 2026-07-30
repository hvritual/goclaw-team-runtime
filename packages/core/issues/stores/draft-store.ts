import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import type {
  IssueStatus,
  IssuePriority,
  IssueAssigneeType,
  IssuePropertyValues,
} from "../../types";
import {
  createWorkspaceAwareStorage,
  registerForWorkspaceRehydration,
} from "../../platform/workspace-storage";
import { defaultStorage } from "../../platform/storage";
import { registerDraftCleanup } from "../../drafts/cleanup-registry";
import {
  normalizeStoredUploads,
  type DraftUpload,
} from "../../drafts/draft-upload";

export interface IssueCreateShared {
  projectId?: string;
  priority: IssuePriority;
  dueDate: string | null;
  attachments: DraftUpload[];
}

export interface IssueCreateManual {
  title: string;
  description: string;
  status: IssueStatus;
  startDate: string | null;
  assigneeType?: IssueAssigneeType;
  assigneeId?: string;
  labelIds: string[];
  propertyValues: IssuePropertyValues;
}

export interface IssueCreateDraft {
  shared: IssueCreateShared;
  manual: IssueCreateManual;
}

const emptyShared = (): IssueCreateShared => ({
  projectId: undefined,
  priority: "none",
  dueDate: null,
  attachments: [],
});

const emptyManual = (): IssueCreateManual => ({
  title: "",
  description: "",
  status: "todo",
  startDate: null,
  assigneeType: undefined,
  assigneeId: undefined,
  labelIds: [],
  propertyValues: {},
});

interface IssueDraftStore {
  draft: IssueCreateDraft;
  lastAssigneeType?: IssueAssigneeType;
  lastAssigneeId?: string;
  setShared: (patch: Partial<IssueCreateShared>) => void;
  setManual: (patch: Partial<IssueCreateManual>) => void;
  clearDraft: () => void;
  setLastAssignee: (type?: IssueAssigneeType, id?: string) => void;
  hasDraft: () => boolean;
}

function isLegacyFlatDraft(d: Record<string, unknown>): boolean {
  return (
    !("manual" in d) &&
    !("shared" in d) &&
    ("title" in d || "status" in d || "labelIds" in d || "description" in d)
  );
}

function migrateDraft(raw: unknown): IssueCreateDraft {
  const draft =
    raw && typeof raw === "object"
      ? (raw as Record<string, unknown>)
      : {};

  if (isLegacyFlatDraft(draft)) {
    return {
      shared: {
        ...emptyShared(),
        projectId: draft.projectId as string | undefined,
        priority: (draft.priority as IssuePriority) ?? "none",
        dueDate: (draft.dueDate as string | null) ?? null,
        attachments: normalizeStoredUploads(draft.attachments),
      },
      manual: {
        ...emptyManual(),
        title: (draft.title as string) ?? "",
        description: (draft.description as string) ?? "",
        status: (draft.status as IssueStatus) ?? "todo",
        startDate: (draft.startDate as string | null) ?? null,
        assigneeType: draft.assigneeType as IssueAssigneeType | undefined,
        assigneeId: draft.assigneeId as string | undefined,
        labelIds: Array.isArray(draft.labelIds)
          ? (draft.labelIds as string[])
          : [],
        propertyValues:
          draft.propertyValues && typeof draft.propertyValues === "object"
            ? (draft.propertyValues as IssuePropertyValues)
            : {},
      },
    };
  }

  const shared =
    (draft.shared as Partial<IssueCreateShared> & {
      attachments?: unknown;
    }) ?? {};

  return {
    shared: {
      ...emptyShared(),
      ...shared,
      attachments: normalizeStoredUploads(shared.attachments),
    },
    manual: {
      ...emptyManual(),
      ...((draft.manual as Partial<IssueCreateManual>) ?? {}),
    },
  };
}

export const useIssueDraftStore = create<IssueDraftStore>()(
  persist(
    (set, get) => ({
      draft: migrateDraft(undefined),
      lastAssigneeType: undefined,
      lastAssigneeId: undefined,
      setShared: (patch) =>
        set((state) => ({
          draft: {
            ...state.draft,
            shared: { ...state.draft.shared, ...patch },
          },
        })),
      setManual: (patch) =>
        set((state) => ({
          draft: {
            ...state.draft,
            manual: { ...state.draft.manual, ...patch },
          },
        })),
      clearDraft: () =>
        set((state) => ({
          draft: {
            shared: emptyShared(),
            manual: {
              ...emptyManual(),
              assigneeType: state.lastAssigneeType,
              assigneeId: state.lastAssigneeId,
            },
          },
        })),
      setLastAssignee: (type, id) =>
        set({ lastAssigneeType: type, lastAssigneeId: id }),
      hasDraft: () => {
        const { manual, shared } = get().draft;
        return Boolean(
          manual.title ||
            manual.description ||
            Object.keys(manual.propertyValues).length > 0 ||
            shared.attachments.some(
              (upload) =>
                upload.status === "uploaded" ||
                upload.status === "uploading",
            ),
        );
      },
    }),
    {
      name: "multica_issue_draft",
      storage: createJSONStorage(() =>
        createWorkspaceAwareStorage(defaultStorage),
      ),
      merge: (persistedState, currentState) => {
        const persisted = (persistedState ?? {}) as Partial<IssueDraftStore> & {
          draft?: unknown;
        };
        return {
          ...currentState,
          ...persisted,
          draft: migrateDraft(persisted.draft),
        };
      },
    },
  ),
);

registerForWorkspaceRehydration(() =>
  useIssueDraftStore.persist.rehydrate(),
);

registerDraftCleanup({
  storageKey: "multica_issue_draft",
  workspaceScoped: true,
  resetInMemory: () =>
    useIssueDraftStore.setState({
      draft: migrateDraft(undefined),
      lastAssigneeType: undefined,
      lastAssigneeId: undefined,
    }),
});
