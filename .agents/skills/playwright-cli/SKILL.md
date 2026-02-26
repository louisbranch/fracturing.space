---
name: playwright-cli
description: Browser automation workflow for navigation, interaction, and capture
allowed-tools: Bash(playwright-cli:*)
---

# Browser Automation with playwright-cli

Use this skill when a task needs browser navigation, form interaction, screenshots,
or data extraction from web pages.

## Default workflow

1. Open a browser session.
2. Navigate to the target page.
3. Capture a snapshot to get element refs (`e1`, `e2`, ...).
4. Interact using those refs.
5. Snapshot or screenshot results.
6. Close the browser and clean session data when done.

## Core commands

### Session and navigation

```bash
playwright-cli open
playwright-cli open https://example.com
playwright-cli open --browser=chrome
playwright-cli goto https://example.com/page
playwright-cli reload
playwright-cli go-back
playwright-cli go-forward
playwright-cli close
```

### Interaction and inspection

```bash
playwright-cli snapshot
playwright-cli click e3
playwright-cli fill e5 "user@example.com"
playwright-cli type "search text"
playwright-cli press Enter
playwright-cli select e7 "option-value"
playwright-cli check e9
playwright-cli uncheck e9
playwright-cli hover e4
playwright-cli eval "document.title"
playwright-cli eval "el => el.textContent" e5
```

### Output, tabs, and state

```bash
playwright-cli screenshot
playwright-cli screenshot e5 --filename=field.png
playwright-cli pdf --filename=page.pdf
playwright-cli tab-new https://example.com/other
playwright-cli tab-list
playwright-cli tab-select 0
playwright-cli state-save auth.json
playwright-cli state-load auth.json
playwright-cli close
playwright-cli delete-data
```

## Working style

- Snapshot before and after meaningful interactions.
- Prefer stable element refs from snapshot output over coordinate-based actions.
- Use named sessions (`-s=<name>`) only when you need multiple concurrent browsers.
- Use persistent profiles only when explicitly required, and clean them up afterward.
- Use `close-all`/`kill-all` only for stuck sessions.

## Quick example

```bash
playwright-cli open https://example.com/login
playwright-cli snapshot
playwright-cli fill e1 "user@example.com"
playwright-cli fill e2 "password123"
playwright-cli click e3
playwright-cli screenshot --filename=after-login.png
playwright-cli close
```

## Task references

- [Request mocking](references/request-mocking.md)
- [Running Playwright code](references/running-code.md)
- [Browser session management](references/session-management.md)
- [Storage state](references/storage-state.md)
- [Test generation](references/test-generation.md)
- [Tracing](references/tracing.md)
- [Video recording](references/video-recording.md)
