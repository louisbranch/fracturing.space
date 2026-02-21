---
title: "Admin Smoke Spec"
parent: "Specs"
nav_order: 1
---

# Admin Smoke Spec (Playwright CLI)

## Purpose
Quick regression coverage for the admin UI: navigation, user creation, impersonation,
and campaigns read-only visibility.

## Preconditions
- Services running (`make run`). This will automatically generate and export dev join-grant keys when missing.
- If starting services manually (not via `make run`), ensure join-grant configuration is exported, for example:
  `eval "$(go run ./cmd/join-grant-key)"; export FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY FRACTURING_SPACE_JOIN_GRANT_ISSUER FRACTURING_SPACE_JOIN_GRANT_AUDIENCE`
- Demo data seeded (recommended): `make seed-variety`

## Automation

Run via `scripts/playwright-admin-smoke.sh` or the spec runner directly.

```playwright-cli
step "Open admin"
open_browser

step "Run admin smoke flow"
cli run-code "$(cat <<'EOF'
async page => {
  const ts = Date.now();
  const email = "playwright-smoke-" + ts + "@example.com";

  await page.setViewportSize({ width: 1280, height: 720 });
  page.setDefaultTimeout(20000);

  await page.getByRole("heading", { name: "Dashboard", level: 2 }).waitFor();
  await page.getByRole("heading", { name: "Recent Activity", level: 3 }).waitFor();

  await page.getByRole("link", { name: "Systems" }).click();
  await page.getByRole("heading", { name: "Systems", level: 2 }).waitFor();
  await page.waitForFunction(() => {
    const table = document.querySelector("table");
    const text = document.body.innerText || "";
    return table || text.includes("No systems registered.") || text.includes("Systems unavailable.") || text.includes("System service unavailable.");
  });

  await page.getByRole("link", { name: "Users" }).click();
  await page.getByRole("heading", { name: "Users", level: 2 }).waitFor();
  await page.getByRole("heading", { name: "All Users", level: 3 }).waitFor();
  await page.waitForFunction(() => {
    const table = document.querySelector("table");
    const text = document.body.innerText || "";
    return table || text.includes("No users yet.") || text.includes("Users unavailable.") || text.includes("User service unavailable.");
  });

  const emailInput = page.locator("form[action=\"/users/create\"] input[name=\"email\"]");
  await emailInput.waitFor();
  await emailInput.fill(email);
  await page.locator("form[action=\"/users/create\"]").getByRole("button", { name: "Create" }).click();
  await page.getByRole("heading", { name: email, level: 2 }).waitFor();

  await page.getByRole("button", { name: "Impersonate" }).click();
  await page.getByText("Currently impersonating").waitFor();

  await page.getByRole("link", { name: "Campaigns" }).click();
  await page.getByRole("heading", { name: "Campaigns", level: 2 }).waitFor();
  await page.waitForFunction(() => {
    const table = document.querySelector("table");
    const text = document.body.innerText || "";
    return table || text.includes("No campaigns yet.") || text.includes("Campaigns unavailable.") || text.includes("Campaign service unavailable.");
  });

  if (await page.getByRole("link", { name: "Create Campaign" }).count() !== 0) {
    throw new Error("Admin unexpectedly shows Create Campaign link.");
  }
  if (await page.locator("form[action=\"/campaigns/create\"]").count() !== 0) {
    throw new Error("Admin unexpectedly renders campaign create form.");
  }

  console.log("Created user: " + email);
}
EOF
)"
```
