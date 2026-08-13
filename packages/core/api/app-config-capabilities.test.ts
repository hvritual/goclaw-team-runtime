import { describe, expect, it } from "vitest";
import { AppConfigSchema } from "./schemas";

describe("AppConfigSchema issue capability flags", () => {
  it("preserves explicit canonical false flags and legacy absence", () => {
    const canonical = AppConfigSchema.parse({
      cdn_domain: "",
      allow_signup: true,
      feature_flags: { issue_list: true, issue_timeline: false },
    });
    expect(canonical.feature_flags).toEqual({ issue_list: true, issue_timeline: false });

    const legacy = AppConfigSchema.parse({ cdn_domain: "", allow_signup: true });
    expect(legacy.feature_flags).toEqual({});
  });

  it("degrades malformed flags to disabled values without rejecting config", () => {
    const config = AppConfigSchema.parse({
      cdn_domain: "",
      allow_signup: true,
      feature_flags: { issue_timeline: "yes", issue_members: null },
    });
    expect(config.feature_flags).toEqual({ issue_timeline: false, issue_members: false });
  });
});
