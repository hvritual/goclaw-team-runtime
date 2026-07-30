"use client";

import { useState } from "react";
import { UserMinus } from "lucide-react";
import type { IssueAssigneeType, UpdateIssueRequest } from "@multica/core/types";
import { useQuery } from "@tanstack/react-query";
import { useActorName } from "@multica/core/workspace/hooks";
import { useWorkspaceId } from "@multica/core/hooks";
import { memberListOptions } from "@multica/core/workspace/queries";
import { ActorAvatar } from "../../../common/actor-avatar";
import { DeferredPopup } from "../../../common/deferred-popup";
import {
  PropertyPicker,
  PickerItem,
  PickerSection,
  PickerEmpty,
  PICKER_TRIGGER_CLASS,
} from "./property-picker";
import { useT } from "../../../i18n";
import { matchesPinyin } from "../../../editor/extensions/pinyin-match";

interface AssigneePickerProps {
  assigneeType: IssueAssigneeType | null;
  assigneeId: string | null;
  mixed?: boolean;
  onUpdate: (updates: Partial<UpdateIssueRequest>) => void;
  trigger?: React.ReactNode;
  triggerRender?: React.ReactElement<Record<string, unknown>>;
  open?: boolean;
  onOpenChange?: (value: boolean) => void;
  align?: "start" | "center" | "end";
}

export function AssigneePicker(props: AssigneePickerProps) {
  const canDefer =
    props.open === undefined &&
    props.onOpenChange === undefined &&
    (props.trigger !== undefined || props.triggerRender?.props.children != null);
  if (!canDefer) return <AssigneePickerImpl {...props} />;
  return (
    <DeferredPopup
      trigger={props.trigger}
      triggerRender={props.triggerRender}
      triggerClassName={PICKER_TRIGGER_CLASS}
    >
      {(open, onOpenChange) => (
        <AssigneePickerImpl {...props} open={open} onOpenChange={onOpenChange} />
      )}
    </DeferredPopup>
  );
}

function AssigneePickerImpl({
  assigneeType,
  assigneeId,
  mixed = false,
  onUpdate,
  trigger,
  triggerRender,
  open: controlledOpen,
  onOpenChange,
  align,
}: AssigneePickerProps) {
  const { t } = useT("issues");
  const workspaceId = useWorkspaceId();
  const { data: members = [] } = useQuery(memberListOptions(workspaceId));
  const { getActorName } = useActorName();
  const [internalOpen, setInternalOpen] = useState(false);
  const [filter, setFilter] = useState("");
  const open = controlledOpen ?? internalOpen;
  const setOpen = onOpenChange ?? setInternalOpen;
  const query = filter.trim().toLocaleLowerCase();
  const filteredMembers = members
    .filter(
      (member) =>
        member.name.toLocaleLowerCase().includes(query) ||
        matchesPinyin(member.name, query),
    )
    .sort((a, b) => a.name.localeCompare(b.name));
  const triggerLabel =
    assigneeType === "member" && assigneeId
      ? getActorName("member", assigneeId)
      : t(($) => $.pickers.assignee.trigger_unassigned);

  return (
    <PropertyPicker
      open={open}
      onOpenChange={(value) => {
        setOpen(value);
        if (!value) setFilter("");
      }}
      width="w-64"
      align={align}
      searchable
      searchPlaceholder={t(($) => $.pickers.assignee.search_placeholder)}
      onSearchChange={setFilter}
      triggerRender={triggerRender}
      trigger={
        trigger ??
        (assigneeType === "member" && assigneeId ? (
          <>
            <ActorAvatar actorType="member" actorId={assigneeId} size="sm" />
            <span className="truncate">{triggerLabel}</span>
          </>
        ) : (
          <span className="text-muted-foreground">{triggerLabel}</span>
        ))
      }
    >
      {!query ? (
        <PickerItem
          selected={!mixed && !assigneeType && !assigneeId}
          onClick={() => {
            onUpdate({ assignee_type: null, assignee_id: null });
            setOpen(false);
          }}
        >
          <UserMinus className="size-3.5 text-muted-foreground" />
          <span className="text-muted-foreground">
            {t(($) => $.pickers.assignee.trigger_unassigned)}
          </span>
        </PickerItem>
      ) : null}
      {filteredMembers.length > 0 ? (
        <PickerSection label={t(($) => $.pickers.assignee.members_group)}>
          {filteredMembers.map((member) => (
            <PickerItem
              key={member.user_id}
              selected={
                assigneeType === "member" && assigneeId === member.user_id
              }
              onClick={() => {
                onUpdate({
                  assignee_type: "member",
                  assignee_id: member.user_id,
                });
                setOpen(false);
              }}
            >
              <ActorAvatar
                actorType="member"
                actorId={member.user_id}
                size="sm"
              />
              <span className="truncate">{member.name}</span>
            </PickerItem>
          ))}
        </PickerSection>
      ) : filter ? (
        <PickerEmpty />
      ) : null}
    </PropertyPicker>
  );
}
