"use client";

import {
  ArrowDown,
  ArrowUp,
  Calendar,
  CalendarClock,
  ExternalLink,
  Link2,
  Network,
  Pin,
  PinOff,
  Plus,
  Trash2,
  Unlink,
  UserMinus,
} from "lucide-react";
import type { Issue } from "@multica/core/types";
import { todayDateOnly, addDaysDateOnly } from "@multica/core/issues/date";
import {
  ALL_STATUSES,
  PRIORITY_ORDER,
  PRIORITY_CONFIG,
} from "@multica/core/issues/config";
import { StatusIcon } from "../components/status-icon";
import { PriorityIcon } from "../components/priority-icon";
import {
  DropdownMenuItem,
  DropdownMenuSub,
  DropdownMenuSubTrigger,
  DropdownMenuSubContent,
  DropdownMenuSeparator,
} from "@multica/ui/components/ui/dropdown-menu";
import {
  ContextMenuItem,
  ContextMenuSub,
  ContextMenuSubTrigger,
  ContextMenuSubContent,
  ContextMenuSeparator,
} from "@multica/ui/components/ui/context-menu";
import type { UseIssueActionsResult } from "./use-issue-actions";
import { useT } from "../../i18n";

// Both Dropdown and Context menu wrappers expose an API-compatible surface
// (variant, inset, onClick, etc.). We bundle the primitives we need into a
// single object so `IssueActionsMenuItems` can render the same JSX for both.
export interface MenuPrimitives {
  Item: typeof DropdownMenuItem;
  Sub: typeof DropdownMenuSub;
  SubTrigger: typeof DropdownMenuSubTrigger;
  SubContent: typeof DropdownMenuSubContent;
  Separator: typeof DropdownMenuSeparator;
}

export const dropdownPrimitives: MenuPrimitives = {
  Item: DropdownMenuItem,
  Sub: DropdownMenuSub,
  SubTrigger: DropdownMenuSubTrigger,
  SubContent: DropdownMenuSubContent,
  Separator: DropdownMenuSeparator,
};

// Context primitives are API-compatible with Dropdown primitives, but their
// TypeScript identities differ. Cast once here and call it a day — this is the
// single bridge between the two primitive sets.
export const contextPrimitives: MenuPrimitives = {
  Item: ContextMenuItem as unknown as typeof DropdownMenuItem,
  Sub: ContextMenuSub as unknown as typeof DropdownMenuSub,
  SubTrigger: ContextMenuSubTrigger as unknown as typeof DropdownMenuSubTrigger,
  SubContent: ContextMenuSubContent as unknown as typeof DropdownMenuSubContent,
  Separator: ContextMenuSeparator as unknown as typeof DropdownMenuSeparator,
};

interface IssueActionsMenuItemsProps {
  issue: Issue;
  actions: UseIssueActionsResult;
  primitives: MenuPrimitives;
  /** Called when the user clicks the Assignee menu item. The parent should
   *  close the surrounding menu and open the shared `AssigneePicker` popover.
   *  Decoupled this way so the same item can drive both the dropdown
   *  (3-dot button) and the context menu (right-click) wrappers. */
  onOpenAssignee: () => void;
  /** If set, leave the page after the issue is deleted (used by the detail
   *  page, which renders the issue being deleted). The delete modal goes back
   *  to the list the user came from and only falls back to this path when
   *  there is no in-app history. List surfaces leave it unset and stay put. */
  onDeletedFallbackPath?: string;
}

export function IssueActionsMenuItems({
  issue,
  actions,
  primitives: P,
  onOpenAssignee,
  onDeletedFallbackPath,
}: IssueActionsMenuItemsProps) {
  const { t } = useT("issues");
  const {
    isPinned,
    updateField,
    openInNewTab,
    togglePin,
    copyLink,
    openCreateSubIssue,
    openSetParent,
    removeParent,
    openAddChild,
    openDeleteConfirm,
  } = actions;

  return (
    <>
      {/* Status */}
      <P.Sub>
        <P.SubTrigger>
          <StatusIcon status={issue.status} className="h-3.5 w-3.5" />
          {t(($) => $.actions.status)}
        </P.SubTrigger>
        <P.SubContent>
          {ALL_STATUSES.map((s) => (
            <P.Item key={s} onClick={() => updateField({ status: s })}>
              <StatusIcon status={s} className="h-3.5 w-3.5" />
              {t(($) => $.status[s])}
              {issue.status === s && (
                <span className="ml-auto text-xs text-muted-foreground">{"✓"}</span>
              )}
            </P.Item>
          ))}
        </P.SubContent>
      </P.Sub>

      {/* Priority */}
      <P.Sub>
        <P.SubTrigger>
          <PriorityIcon priority={issue.priority} />
          {t(($) => $.actions.priority)}
        </P.SubTrigger>
        <P.SubContent>
          {PRIORITY_ORDER.map((p) => (
            <P.Item key={p} onClick={() => updateField({ priority: p })}>
              <span
                className={`inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-xs font-medium ${PRIORITY_CONFIG[p].badgeBg} ${PRIORITY_CONFIG[p].badgeText}`}
              >
                <PriorityIcon priority={p} className="h-3 w-3" inheritColor />
                {t(($) => $.priority[p])}
              </span>
              {issue.priority === p && (
                <span className="ml-auto text-xs text-muted-foreground">{"✓"}</span>
              )}
            </P.Item>
          ))}
        </P.SubContent>
      </P.Sub>

      {/* Assignee — closes this menu and hands off to the shared
          AssigneePicker (members + agents + squads, with search and
          permission checks). Keeps a single source of truth for the
          assignee UX across detail sidebar, board cards, and right-click /
          3-dot menus. */}
      <P.Item onClick={onOpenAssignee}>
        <UserMinus className="h-3.5 w-3.5" />
        {t(($) => $.actions.assignee)}
      </P.Item>

      {/* Start date */}
      <P.Sub>
        <P.SubTrigger>
          <CalendarClock className="h-3.5 w-3.5" />
          {t(($) => $.actions.start_date)}
        </P.SubTrigger>
        <P.SubContent>
          <P.Item onClick={() => updateField({ start_date: todayDateOnly() })}>
            {t(($) => $.actions.start_today)}
          </P.Item>
          <P.Item onClick={() => updateField({ start_date: addDaysDateOnly(1) })}>
            {t(($) => $.actions.start_tomorrow)}
          </P.Item>
          <P.Item onClick={() => updateField({ start_date: addDaysDateOnly(7) })}>
            {t(($) => $.actions.start_next_week)}
          </P.Item>
          {issue.start_date && (
            <>
              <P.Separator />
              <P.Item onClick={() => updateField({ start_date: null })}>
                {t(($) => $.actions.start_clear)}
              </P.Item>
            </>
          )}
        </P.SubContent>
      </P.Sub>

      {/* Due date */}
      <P.Sub>
        <P.SubTrigger>
          <Calendar className="h-3.5 w-3.5" />
          {t(($) => $.actions.due_date)}
        </P.SubTrigger>
        <P.SubContent>
          <P.Item onClick={() => updateField({ due_date: todayDateOnly() })}>
            {t(($) => $.actions.due_today)}
          </P.Item>
          <P.Item onClick={() => updateField({ due_date: addDaysDateOnly(1) })}>
            {t(($) => $.actions.due_tomorrow)}
          </P.Item>
          <P.Item onClick={() => updateField({ due_date: addDaysDateOnly(7) })}>
            {t(($) => $.actions.due_next_week)}
          </P.Item>
          {issue.due_date && (
            <>
              <P.Separator />
              <P.Item onClick={() => updateField({ due_date: null })}>
                {t(($) => $.actions.due_clear)}
              </P.Item>
            </>
          )}
        </P.SubContent>
      </P.Sub>

      <P.Separator />

      {/* Leads the "do something with this issue itself" group: the only
          discoverable way to open an issue elsewhere for users who don't know
          modifier-click, so it sits above the copy actions. */}
      <P.Item onClick={openInNewTab}>
        <ExternalLink className="h-3.5 w-3.5" />
        {t(($) => $.actions.open_in_new_tab)}
      </P.Item>
      <P.Item onClick={togglePin}>
        {isPinned ? (
          <PinOff className="h-3.5 w-3.5" />
        ) : (
          <Pin className="h-3.5 w-3.5" />
        )}
        {isPinned ? t(($) => $.actions.unpin_from_sidebar) : t(($) => $.actions.pin_to_sidebar)}
      </P.Item>
      <P.Item onClick={copyLink}>
        <Link2 className="h-3.5 w-3.5" />
        {t(($) => $.actions.copy_link)}
      </P.Item>

      <P.Separator />

      {/* Relationship actions live under "Relations" — a semantically explicit
          label (unlike the old "More") so the first level tells you what the
          submenu does. Holds parent/sub-issue links today, and will grow
          (blocks, duplicates, related) as we add more relation types. */}
      <P.Sub>
        <P.SubTrigger>
          <Network className="h-3.5 w-3.5" />
          {t(($) => $.actions.relations)}
        </P.SubTrigger>
        <P.SubContent>
          <P.Item onClick={openCreateSubIssue}>
            <Plus className="h-3.5 w-3.5" />
            {t(($) => $.actions.create_sub_issue)}
          </P.Item>
          <P.Item onClick={openSetParent}>
            <ArrowUp className="h-3.5 w-3.5" />
            {t(($) => $.actions.set_parent_issue)}
          </P.Item>
          {issue.parent_issue_id && (
            <P.Item onClick={removeParent}>
              <Unlink className="h-3.5 w-3.5" />
              {t(($) => $.actions.remove_parent_issue)}
            </P.Item>
          )}
          <P.Item onClick={openAddChild}>
            <ArrowDown className="h-3.5 w-3.5" />
            {t(($) => $.actions.add_sub_issue)}
          </P.Item>
        </P.SubContent>
      </P.Sub>

      <P.Separator />

      <P.Item
        variant="destructive"
        onClick={() => openDeleteConfirm({ onDeletedFallbackPath })}
      >
        <Trash2 className="h-3.5 w-3.5" />
        {t(($) => $.actions.delete_issue)}
      </P.Item>
    </>
  );
}
