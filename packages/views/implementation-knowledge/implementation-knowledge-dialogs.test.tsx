// @vitest-environment jsdom
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import {
  AcceptanceConclusionDialog,
  ProjectRetrospectiveDialog,
} from "./implementation-knowledge-dialogs";
import { renderWithI18n } from "../test/i18n";

describe("implementation knowledge dialogs", () => {
  it("lets a user complete an issue without fabricating a conclusion", () => {
    const onSubmit = vi.fn();
    renderWithI18n(<AcceptanceConclusionDialog open onOpenChange={() => {}} onSubmit={onSubmit} />, { locale: "zh-Hans" });

    fireEvent.click(screen.getByRole("button", { name: "直接完成" }));
    expect(onSubmit).toHaveBeenCalledWith(null);
  });

  it("submits an explicit acceptance conclusion", () => {
    const onSubmit = vi.fn();
    renderWithI18n(<AcceptanceConclusionDialog open onOpenChange={() => {}} onSubmit={onSubmit} />, { locale: "zh-Hans" });

    fireEvent.change(screen.getByLabelText("验收说明"), { target: { value: "恢复检查全部通过" } });
    fireEvent.change(screen.getByLabelText("证据引用"), { target: { value: "artifact://report/v2" } });
    fireEvent.click(screen.getByRole("button", { name: "完成并沉淀" }));
    expect(onSubmit).toHaveBeenCalledWith({
      result: "accepted",
      rationale: "恢复检查全部通过",
      evidenceRefs: ["artifact://report/v2"],
    });
  });

  it("supports post-completion capture without offering another completion", () => {
    const onSubmit = vi.fn();
    renderWithI18n(
      <AcceptanceConclusionDialog open mode="capture" onOpenChange={() => {}} onSubmit={onSubmit} />,
      { locale: "zh-Hans" },
    );
    expect(screen.queryByRole("button", { name: "直接完成" })).toBeNull();
    fireEvent.change(screen.getByLabelText("验收说明"), { target: { value: "补录验收记录" } });
    fireEvent.click(screen.getByRole("button", { name: "记录验收结论" }));
    expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ rationale: "补录验收记录" }));
  });

  it("requires a lesson and submits structured participants and action items", () => {
    const onSubmit = vi.fn();
    renderWithI18n(
      <ProjectRetrospectiveDialog
        open
        onOpenChange={() => {}}
        onSubmit={onSubmit}
        members={[
          { id: "member-1", workspace_id: "workspace-1", user_id: "user-1", role: "member", created_at: "2026-08-01T00:00:00Z", name: "林舟", email: "lin@example.com", avatar_url: null },
          { id: "member-2", workspace_id: "workspace-1", user_id: "user-2", role: "member", created_at: "2026-08-01T00:00:00Z", name: "周宁", email: "zhou@example.com", avatar_url: null },
        ]}
      />,
      { locale: "zh-Hans" },
    );

    fireEvent.change(screen.getByLabelText("复盘总结"), { target: { value: "首批交付完成" } });
    expect(screen.getByRole("button", { name: "创建复盘草稿" })).toBeDisabled();
    fireEvent.change(screen.getByLabelText("经验教训"), { target: { value: "更早安排验收" } });
    fireEvent.click(screen.getByRole("button", { name: "添加行动项" }));
    fireEvent.change(screen.getByLabelText("行动项标题 1"), { target: { value: "安排发布前评审" } });
    fireEvent.change(screen.getByLabelText("行动项负责人 1"), { target: { value: "member-2" } });
    fireEvent.change(screen.getByLabelText("行动项截止日期 1"), { target: { value: "2026-08-30" } });
    fireEvent.click(screen.getByLabelText("加入参与者：周宁"));
    fireEvent.change(screen.getByLabelText("参与角色：周宁"), { target: { value: "facilitator" } });
    fireEvent.click(screen.getByRole("button", { name: "创建复盘草稿" }));
    expect(onSubmit).toHaveBeenCalledWith({
      content: {
        summary: "首批交付完成",
        successes: [],
        problems: [],
        lessons: ["更早安排验收"],
        actionItems: [{
          title: "安排发布前评审",
          assigneeId: "member-2",
          dueDate: "2026-08-30",
        }],
      },
      participants: [{ memberId: "member-2", role: "facilitator" }],
    });
  });

  it("retains server-owned action item IDs while editing a draft", () => {
    const onSubmit = vi.fn();
    renderWithI18n(
      <ProjectRetrospectiveDialog
        open
        mode="save_draft"
        onOpenChange={() => {}}
        onSubmit={onSubmit}
        initialContent={{
          summary: "Draft",
          successes: [],
          problems: [],
          lessons: ["Keep IDs"],
          actionItems: [{ id: "action-1", title: "Existing action" }],
        }}
        initialParticipants={[]}
      />,
      { locale: "en" },
    );

    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
    expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({
      content: expect.objectContaining({
        actionItems: [expect.objectContaining({ id: "action-1", title: "Existing action" })],
      }),
    }));
  });
});
