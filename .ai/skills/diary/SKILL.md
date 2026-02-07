---
name: diary
description: Capture session learnings in a structured diary entry
user-invocable: true
---

# Diary Entry

Capture learnings from the current session in a structured diary entry.

## When to Use

Invoke `/diary` at the end of a meaningful work session when you have:
- Made design decisions worth remembering
- Solved non-trivial problems
- Discovered project patterns or conventions
- Completed a feature or significant refactoring

Skip for trivial sessions (typo fixes, simple queries, routine commits).

## Process

1. Review the conversation to identify learnings
2. Create a diary entry at `.ai/memory/diary/YYYY-MM-DD-<topic>.md`
3. If multiple entries exist for the same day, append a sequence number

## Entry Template

# <Topic Title>

**Date**: YYYY-MM-DD
**Branch**: <branch-name-if-applicable>

### Session Summary
<One-line description of the main task or goal>

### Work Completed
- <Bullet points of changes made>

### Design Decisions
- **<Decision>**: <Rationale>

### Challenges & Solutions
**<Challenge>**: <Problem description>
**Solution**: <How it was resolved>

### Patterns Discovered
- <Reusable patterns or conventions learned about this codebase>

### References
- <Links to documentation, issues, PRs, or prior art consulted>

### Future Considerations
- <Ideas, technical debt, or follow-up items>

## Output

After creating the entry, display the file path and a brief summary of what was captured.
