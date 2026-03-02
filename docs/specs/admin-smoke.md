---
title: "Admin Smoke Spec"
parent: "Specs"
nav_order: 1
nav_exclude: true
---

# Admin Smoke Spec (Playwright CLI)

## Purpose
Quick regression coverage for the admin UI: navigation, users lookup/list visibility,
and campaigns read-only visibility.

## Preconditions
- Services running (`make up` recommended). `make up` starts the devcontainer and watchers together (or just re-starts watchers when run inside the devcontainer). These flows initialize `.env` and generate dev join-grant keys when missing.
- If starting services manually (not via `make up`), ensure join-grant configuration is exported, for example:
  `eval "$(go run ./cmd/join-grant-key)"; export FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY FRACTURING_SPACE_JOIN_GRANT_ISSUER FRACTURING_SPACE_JOIN_GRANT_AUDIENCE`
- Demo data seeded (recommended): `make seed`

## Automation

Run via `scripts/playwright-admin-smoke.sh` or the spec runner directly.

```playwright-cli
step "Open admin"
open_browser

step "Run admin smoke flow"
cli run-code "$(cat <<'EOF'
async page => {
  await page.setViewportSize({ width: 1280, height: 720 });
  page.setDefaultTimeout(20000);

  await page.getByRole("heading", { name: "Dashboard", exact: true }).waitFor();
  await page.getByRole("heading", { name: "Recent Activity", exact: true }).waitFor();

  await page.getByRole("link", { name: "Systems" }).click();
  await page.getByRole("heading", { name: "Systems", exact: true }).waitFor();
  await page.waitForFunction(() => {
    const table = document.querySelector("table");
    const text = document.body.innerText || "";
    return table || text.includes("No systems registered.") || text.includes("Systems unavailable.") || text.includes("System service unavailable.");
  });

  await page.getByRole("link", { name: "Users" }).click();
  await page.getByRole("heading", { name: "Users", exact: true }).waitFor();
  await page.getByRole("heading", { name: "All Users", exact: true }).waitFor();
  await page.waitForFunction(() => {
    const table = document.querySelector("table");
    const text = document.body.innerText || "";
    return table || text.includes("No users yet.") || text.includes("Users unavailable.") || text.includes("User service unavailable.");
  });
  const lookupForm = page.locator("form[action=\"/users/lookup\"]");
  await lookupForm.waitFor();
  await lookupForm.locator("input[name=\"user_id\"]").waitFor();

  await page.getByRole("link", { name: "Campaigns" }).click();
  await page.getByRole("heading", { name: "Campaigns", exact: true }).waitFor();
  await page.waitForFunction(() => {
    const table = document.querySelector("table");
    const text = document.body.innerText || "";
    return table || text.includes("No campaigns yet.") || text.includes("Campaigns unavailable.") || text.includes("Campaign service unavailable.");
  });
}
EOF
)"
```
