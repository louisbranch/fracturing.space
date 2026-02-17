---
name: schema
description: Database migrations and proto field ordering rules
user-invocable: true
---

# Schema Changes

This is a fast-moving prototype. Favor logical, clear structure over backwards compatibility.

## Database Migrations

- Do: `CREATE TABLE` with full schema definition.
- Don't: `ALTER TABLE`, rename columns, or add fields to existing tables.

When the schema changes, create a new migration that drops and recreates the table.
This keeps migrations simple and avoids migration ordering issues during rapid development.

## Proto Fields

- Do: Reorder fields for logical grouping and clarity.
- Don't: Preserve field numbers or ordering for backwards compatibility.

When adding or reorganizing proto fields, renumber them to maintain a clean sequence.
Regenerate with `make proto`.
