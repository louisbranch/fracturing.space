---
name: workflow
description: Git branching, commits, and PR conventions for this project
user-invocable: true
---

# Git Workflow

Git branching, commits, and PR conventions.

## Branches

Create a feature branch before making changes; never work directly on main.

Prefixes:
- `feat/<name>` - New features
- `fix/<name>` - Bug fixes
- `chore/<name>` - Maintenance, dependencies
- `docs/<name>` - Documentation only

## Commits

Use matching prefixes with a short, why-focused subject:
- `feat:` - New features
- `fix:` - Bug fixes
- `chore:` - Maintenance
- `docs:` - Documentation

Example: `feat: add duality outcome tool`

## Pull Requests

- Match PR titles to commit prefix style
- Keep one intent per PR; split unrelated changes
- Prefer small, focused changes
- Avoid reformatting unrelated code
- Do not introduce new files unless required
- Mention missing tests or tooling in summaries

## Merge Strategy

- Prefer squash merge when enabling auto-merge
- Do not push to closed/merged PR branches; open new ones
