---
name: pr-issues
description: PR review triage, fixes, testing, and auto-merge workflow
user-invocable: true
---

# PR Issues Workflow

Codify how to triage GitHub PR review comments, propose fixes, apply agreed changes, run tests, and set auto-merge with squash.

## When to Use

Use this skill when the user asks to:

- Triage PR review comments
- Respond to automated reviewer feedback
- Fix PR issues and update the PR
- Prepare a PR for auto-merge

## Core Workflow

1. **Fetch PR context**
   - Get the PR number with `gh pr view --json number`
   - Always fetch inline review comments (file/line) using the PR comments endpoint.

2. **Wait for automated reviewer (if expected)**
   - If the automated review posts after a delay, wait until it lands before making changes.
   - If the review has not arrived, report that status and retry as needed.

3. **Triage feedback**
   - Must-fix: correctness, security, test failures, required reviewer notes.
   - Should-fix: maintainability, clarity, small risks.
   - Won't-fix: stylistic preference, low impact, or conflicts with existing conventions.

4. **Recommend actions**
   - Provide a concise recommendation per comment.
   - If the user has not specified, default to fixing must-fix and high-confidence should-fix items.
   - Call out won’t-fix items with rationale.

5. **Implement approved changes**
   - Apply only agreed changes; avoid unrelated refactors.
   - Use relevant skills if changes touch those domains (`schema`, `error-handling`, `go-style`, `mcp`, `web-server`).

6. **Verify**
   - Run `make integration` after code changes (per repo policy).
   - Note failures and propose fixes before proceeding.

7. **Update PR and enable auto-merge**
   - Post a short summary comment (what changed, tests run).
   - Enable auto-merge squash when checks pass.

## GitHub CLI Commands

Use `gh` for all PR data and updates.

```bash
gh pr view <pr> --json title,number,headRefName,reviewDecision,checks
gh api repos/<owner>/<repo>/pulls/<pr>/comments
gh api repos/<owner>/<repo>/pulls/<pr>/reviews
gh pr comment <pr> --body "<summary>"
gh pr merge <pr> --auto --squash
```

Inline comments (file/line) come from `pulls/<pr>/comments`. Always use that endpoint for line-level feedback.

## Response Format

Provide a short triage report with:

- Must-fix: bullets with recommendation and rationale.
- Should-fix: bullets with recommendation and rationale.
- Won't-fix: bullets with recommendation and rationale.

After changes, report:

- Files updated.
- Tests run and results.
- PR update comment contents.
- Auto-merge status (enabled or blocked by checks).
