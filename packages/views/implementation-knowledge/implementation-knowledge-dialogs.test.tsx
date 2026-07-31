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

  it("requires a lesson before recording a retrospective", () => {
    const onSubmit = vi.fn();
    renderWithI18n(<ProjectRetrospectiveDialog open onOpenChange={() => {}} onSubmit={onSubmit} />, { locale: "zh-Hans" });

    fireEvent.change(screen.getByLabelText("复盘总结"), { target: { value: "首批交付完成" } });
    expect(screen.getByRole("button", { name: "记录复盘" })).toBeDisabled();
    fireEvent.change(screen.getByLabelText("经验教训"), { target: { value: "更早安排验收" } });
    fireEvent.click(screen.getByRole("button", { name: "记录复盘" }));
    expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({
      summary: "首批交付完成",
      lessons: ["更早安排验收"],
    }));
  });
});
