# Playwright-CLI: Learnings & Patterns

Concise reference for using `playwright-cli` for exploratory testing, spike QA,
and manual verification. Distilled from real usage on a Vite + React app.

## When to use playwright-cli

| Use case | Tool |
|----------|------|
| Spike / POC QA, manual smoke tests | **playwright-cli** |
| Repeatable CI test suite | Playwright Test runner |
| Quick one-off screenshot or form fill | **playwright-cli** |

Playwright-CLI is ideal when you need fast, interactive browser automation
without the ceremony of a full test framework. Once patterns stabilize, migrate
the valuable scenarios to Playwright Test for CI.

## Setup

Examples below use `playwright-cli` for brevity. In this repo, prefer
`npx -y @playwright/cli@0.1.0` (or set `PLAYWRIGHT_CLI_PKG`) unless you have a
global `playwright-cli` binary.

### Local setup (Go repo)

- Install Node.js 20 LTS (for `npx`).
- Install Playwright CLI browsers once per machine:

```bash
npx -y @playwright/cli@0.1.0 install
```

- Start services (long-running) and seed demo data:

```bash
# Terminal 1
make run

# Terminal 2
make seed-variety
scripts/playwright-admin-smoke.sh
```

Environment overrides:

```bash
BASE_URL="http://localhost:8082" \
PLAYWRIGHT_OPEN_ARGS="--headed" \
ARTIFACT_ROOT="artifacts/playwright" \
PLAYWRIGHT_CLI_PKG="@playwright/cli@0.1.0" \
scripts/playwright-admin-smoke.sh
```

### Specs

The admin smoke flow lives in `docs/specs/admin-smoke.md` and is executed by
the spec runner:

```bash
scripts/playwright-run-spec.sh docs/specs/admin-smoke.md
```

See `docs/specs/playwright-cli-qa-workflows.md` for the spec format.

### Config file

Use `.playwright/cli.config.json` to set defaults (browser, viewport, etc.).
This repo checks in a baseline config for Chromium. The skill at
`.ai/skills/playwright-cli/SKILL.md` has the full command reference.

### .gitignore

Add these entries so ephemeral browser data and test artifacts stay out of
version control:

```gitignore
.playwright-cli/
artifacts/
```

### Browser choice

```bash
playwright-cli open --browser=chrome   # Chromium (default)
playwright-cli open --browser=firefox
playwright-cli open --browser=webkit
```

## Patterns that worked

### 1. Semantic selectors over snapshot element refs

Prefer `getByRole`, `getByText`, and `getByLabel` locators inside `run-code`
blocks. Snapshot element refs (`e1`, `e5`) are handy for quick interactive use,
but they change across page loads and are meaningless in scripts.

```bash
# Good — readable and resilient
playwright-cli run-code "async page => {
  await page.getByRole('button', { name: 'Submit' }).click();
}"

# Fragile — ref changes on every snapshot
playwright-cli click e5
```

### 2. Shell-script wrappers with step() function

Wrap each logical step in a `step()` helper that tracks pass/fail and writes to
a report file:

```bash
step() {
  local label="$1"
  shift
  echo "==> ${label}"
  set +e
  "$@"
  local status=$?
  set -e
  if [[ $status -ne 0 ]]; then
    echo "FAIL: ${label}" >&2
    report_line "FAIL" "Step failed: ${label}"
    exit $status
  fi
  echo "PASS: ${label}"
}

step "Open app" playwright-cli open "$base_url"
step "Click submit" playwright-cli run-code "async page => { ... }"
```

This gives clear console output per step and a machine-parseable report for
sharing in PRs or Linear comments.

### 3. run-code for complex interactions

When a scenario needs XPath traversal, conditional logic, or multi-step
sequences that can't be expressed as single CLI commands:

```bash
playwright-cli run-code "async page => {
  const heading = page.getByRole('heading', { name: 'Poll Title' }).first();
  const card = heading.locator('xpath=../..');
  await card.getByRole('button', { name: /yes/i }).click();
  await page.waitForTimeout(800);
}"
```

### 4. Parameterized test data via env vars

All scenario scripts accept configuration through environment variables with
sensible defaults:

```bash
base_url=${BASE_URL:-http://localhost:5173}
poll_title=${POLL_TITLE:-My Poll Title}
option_text=${OPTION_TEXT:-Option A}
```

This makes scripts reusable across environments (local dev, staging, etc.)
without editing the script itself.

### 5. Timestamped artifact directories

Store screenshots, traces, and reports in a predictable directory structure:

```bash
timestamp=$(date -u +"%Y-%m-%dT%H%MZ")
dir="artifacts/playwright/${flow_name}__${timestamp}"
mkdir -p "$dir"
```

Produces: `artifacts/playwright/poll-vote__2026-02-11T1504Z/`

### 6. "Known issue" mode

For limitations you want to document without failing the suite, use a
dual-mode handler controlled by an env var:

```bash
expect_failure=${EXPECT_KNOWN_FAILURE:-true}

if [[ $check_status -ne 0 ]]; then
  if [[ "$expect_failure" == "true" ]]; then
    report_line "KNOWN ISSUE" "Feature X not yet implemented"
  else
    report_line "FAIL" "Feature X broke"
    exit $check_status
  fi
fi
```

Set the env var to `false` once the feature is implemented to turn the known
issue into a hard failure.

### 7. ASCII table reports

Write results as pipe-delimited rows, then render as an ASCII table for
PR/Linear sharing:

```
+------------------+--------------+------------------------------------------+
| Scenario         | Status       | Reason                                   |
+------------------+--------------+------------------------------------------+
| dashboard-vote   | PASS         | Vote recorded on dashboard               |
| vote-persist     | KNOWN ISSUE  | Dashboard did not reflect detail vote    |
+------------------+--------------+------------------------------------------+
```

The rendering script computes column widths dynamically from the data.

### 8. Viewport switching for responsive testing

Switch between desktop and mobile viewports within a single scenario:

```bash
# Desktop
playwright-cli run-code "async page => {
  await page.setViewportSize({ width: 1280, height: 720 });
}"
playwright-cli run-code "async page => {
  await page.waitForTimeout(300);
}"

# Mobile
playwright-cli run-code "async page => {
  await page.setViewportSize({ width: 375, height: 812 });
}"
playwright-cli run-code "async page => {
  await page.waitForTimeout(300);
}"
```

Verify visibility with `boundingBox()`:

```bash
playwright-cli run-code "async page => {
  const box = await page.getByRole('button', { name: 'Menu' }).boundingBox();
  if (!box) throw new Error('Menu button not visible at mobile width');
}"
```

## Gotchas

### XPath `../..` for parent traversal

When a semantic selector finds a child element but you need to interact with its
parent container, `xpath=../..` navigates up two DOM levels. This is brittle
(breaks if nesting depth changes) but sometimes unavoidable when semantic
selectors can't reach the right container.

```javascript
const heading = page.getByRole('heading', { name: 'Title' }).first();
const card = heading.locator('xpath=../..'); // up 2 levels to card
await card.getByRole('button', { name: 'Vote' }).click();
```

### Hard-coded waitForTimeout is fragile

`page.waitForTimeout(800)` works during development but is unreliable in
different environments. Prefer waiting on specific elements:

```javascript
// Fragile
await page.waitForTimeout(800);

// Better
await page.getByText('Vote recorded').waitFor({ timeout: 5000 });
```

The one exception is viewport changes, where a short ~300ms delay for layout
reflow is reasonable.

### Bash `${var@Q}` escaping for JS string interpolation

Shell variables containing special characters (apostrophes, question marks)
break the double-quoted `run-code` argument. Use Bash 4+ quoting:

```bash
js_title=${poll_title@Q}
playwright-cli run-code "async page => {
  const title = ${js_title};
  // ...
}"
```

`${var@Q}` produces a single-quoted, shell-escaped version of the value. Clever
but non-obvious — add a comment when using this pattern.

### Check both exit status AND stderr from run-code

`playwright-cli run-code` does not always set a non-zero exit code on JS errors.
Double-check by inspecting the output text:

```bash
set +e
output=$(playwright-cli run-code "async page => { ... }")
status=$?
set -e

if [[ "$output" == *"Error:"* ]]; then
  status=1
fi
```

### No test isolation

Scenarios share localhost state (cookies, localStorage, server-side data).
If scenario A writes data that scenario B depends on, run them in order. If
they must be independent, clear state explicitly between runs or use separate
browser sessions (`playwright-cli -s=session1`).

## Migration path to Playwright Test

Once patterns stabilize and you want CI integration:

1. Use the generated Playwright code from `run-code` output as a starting point
2. Replace `step()` / report wrappers with `test()` / `expect()` blocks
3. Replace env-var parameterization with Playwright Test fixtures
4. Replace `waitForTimeout` with proper `waitFor` / `expect` assertions
5. Add `playwright.config.ts` with `baseURL`, browser matrix, and CI settings
6. Move known-issue scenarios to `test.fixme()` blocks
