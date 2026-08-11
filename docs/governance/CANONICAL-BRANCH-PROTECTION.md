# Canonical Integration Branch Protection

This document is the auditable specification for the repository-level ruleset.
It does not replace the remote GitHub setting.

## Target

- Repository: `hvritual/goclaw-team-runtime`
- Branch: `codex/multica-six-domain-baseline`
- Ruleset name: `canonical-integration`
- Enforcement: active

## Required rules

1. Require a pull request before merging.
2. Require at least one approving review from a reviewer other than the change
   author.
3. Dismiss stale approvals when new commits are pushed.
4. Require every review conversation to be resolved before merging.
5. Require status check `CI / required`.
6. Require the branch to be up to date before merging.
7. Block force pushes.
8. Block branch deletion.
9. Do not grant apps, Actions, Runner, models, or implementation authors bypass
   permission.
10. Merge and release remain separate human-controlled gates; a green check is
    evidence, not DoneGate acceptance.

## Stable status contract

The workflow is `.github/workflows/ci.yml` with workflow name `CI`. Its
aggregate job has the job name `required`, producing the stable context:

```text
CI / required
```

The aggregate fails unless all of these jobs succeed:

- governance-policy;
- canonical-backend;
- frontend;
- legacy-server-regression;
- Windows execution-environment regression;
- installer regression.

Conditional work is kept inside reporting jobs so a required status does not
remain pending because a job was omitted by a path filter.

## Verification

Remote completion requires inspecting the active GitHub ruleset or branch
protection and confirming:

- the target branch pattern is exact;
- enforcement is active;
- `CI / required` is the only required aggregate CI context;
- pull-request, approval, stale-review, conversation, force-push, and deletion
  rules match this document;
- no unintended bypass actor exists.

A committed copy of this file without matching remote state is an incomplete
hard gate.
