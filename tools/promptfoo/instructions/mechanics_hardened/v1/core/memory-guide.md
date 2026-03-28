# Memory Management

## Campaign Documents

- Read story.md for campaign-specific setup, starter context, and ongoing plot notes.
- Read and update memory.md to keep durable GM memory between turns.
- You may create and update additional markdown notes under working/{slug}.md when you need scratch documents.
- Treat skills.md as the operating contract for this GM workflow. Do not attempt to overwrite it.

## Section Tools

Use `campaign_memory_section_read` and `campaign_memory_section_update` to work with individual `## Heading` sections in memory.md without replacing the full file. This is safer and more efficient than rewriting the full file each turn.

Recommended section headings:

- **NPCs** — named characters the party has met, their roles, and dispositions.
- **Plot Hooks** — active story threads, quests, and unresolved tensions.
- **World State** — locations, factions, and environmental conditions.
- **Session Notes** — key events and decisions from the current and recent sessions.
- **Player Decisions** — significant choices players have made and their consequences.

You may create additional headings as needed. Section matching is case-insensitive.

## Memory Best Practices

- Update memory.md after significant story events, NPC introductions, or world-state changes.
- Prefer section-level updates to full-document rewrites.
- Use working notes for complex planning that does not need to persist beyond a few turns.
- Do not duplicate information that is already in story.md or the interaction state.
