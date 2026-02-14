# Playwright CLI QA Workflows

These specs are lightweight, readable automation scripts for Playwright CLI.
Each spec is a Markdown file with runnable `playwright-cli` code blocks.

## Running a spec

```bash
scripts/playwright-run-spec.sh docs/specs/admin-smoke.md
```

The runner executes fenced code blocks marked with `playwright-cli` and ignores
all other content.

## Spec format

- Use a fenced code block starting with ` ```playwright-cli`.
- Use `step "<label>"` to name the next CLI action.
- Use `cli <command> ...` to invoke Playwright CLI commands.
- Use `open_browser` to open the browser with `BASE_URL` and optional args.

Each `step` label applies to the next `cli` call.

### Example

```playwright-cli
step "Open admin"
open_browser

step "Check dashboard"
cli run-code "$(cat <<'EOF'
async page => {
  await page.getByRole('heading', { name: 'Dashboard', level: 2 }).waitFor();
}
EOF
)"
```

## Environment variables

- `BASE_URL` (default `http://localhost:8082`)
- `PLAYWRIGHT_OPEN_ARGS` (example: `--headed`)
- `ARTIFACT_ROOT` (default `artifacts/playwright`)
- `FLOW_NAME` (defaults to spec filename without `.md`)
- `PLAYWRIGHT_CLI_PKG` (default `@playwright/cli@0.1.0`)
- `PLAYWRIGHT_CLI_CMD` (optional override path to a CLI binary)

Artifacts and a `report.txt` file are written under:

```
artifacts/playwright/<flow>__<timestamp>/
```

## Notes

- The function passed to `run-code` runs in the Playwright (Node.js) context and
  controls the `page` object. Use `page.evaluate(...)` to run code in the
  browser context, and pass values via shell variables or function arguments.
