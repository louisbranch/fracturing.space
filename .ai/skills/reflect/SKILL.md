---
name: reflect
description: Analyze diary entries and crystallize patterns into AGENTS.md
user-invocable: true
---

# Reflect on Diary Entries

Analyze recent diary entries and identify patterns worth adding to AGENTS.md.

## When to Use

Invoke `/reflect` periodically when:
- Multiple diary entries have accumulated (3-5+)
- You notice repeated patterns across sessions
- Before starting a major new feature

## Process

1. Read all diary entries from `.ai/memory/diary/`
2. Read current AGENTS.md content
3. Identify recurring patterns (2+ occurrences = pattern)
4. Propose specific additions grouped by AGENTS.md section
5. Show diff preview and wait for user confirmation
6. Apply changes only after approval

## What to Extract

**Add to AGENTS.md**:
- Recurring architectural patterns
- Non-obvious conventions discovered through experience
- Common pitfalls and solutions
- Testing strategies that work well

**Keep in Diary Only**:
- One-off implementation details
- Session-specific context
- Temporary workarounds

## After Reflection

Optionally archive processed entries to `.ai/memory/diary/archive/` (create directory if needed)
