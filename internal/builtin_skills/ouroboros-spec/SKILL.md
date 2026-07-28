---
name: ouroboros-spec
description: Turn a software-development request into an approval-gated, immutable Ouroboros Seed through a Socratic interview. Use when a user asks to build, change, repair, or evolve project code and the GoClaw Ouroboros tools are available.
---

# Ouroboros specification

Run specification work in this order:

1. Start or resume one project-scoped interview.
2. Ask only the questions returned by Go Core.
3. Record the user's explicit answers without filling gaps yourself.
4. Reassess ambiguity until Go Core reports `seed_ready`.
5. Crystallize an immutable Seed proposal.
6. Direct the user to Obsidian or the local CLI for approval and compilation.

## Tool routing

- Start with `ouroboros_start`.
- Read a known session with `ouroboros_get`.
- Submit answers with `ouroboros_answer`.
- Use `ouroboros_reassess` only after new facts are available or Go Core requests a second readiness pass.
- Use `ouroboros_crystallize` only when status is `seed_ready`.

Keep `project_id` aligned with the current Feishu/Obsidian project. Use the configured repository path and `HEAD` unless the human explicitly gives another base reference.

## Safety boundary

- Treat the returned ambiguity score and status as authoritative.
- Never fabricate an answer, acceptance criterion, evidence result, or approval.
- Never describe a proposed Seed as approved.
- Never claim that crystallization changed code or created an executable task.
- Do not attempt Seed approval, task compilation, execution, acceptance, evolution approval, Harness promotion, or knowledge mutation from a chat channel.
- Explain that human approval grants permission only; deterministic verification and independent evidence still decide correctness.
- If questions repeat or status becomes `clarification_required` or `blocked`, surface the unresolved decision to the human.

Summarize the current state with the session ID, ambiguity percentage, blocking questions, readiness streak, and the next permitted action.
