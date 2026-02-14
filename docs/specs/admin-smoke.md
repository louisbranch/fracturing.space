# Admin Smoke Spec (Playwright CLI)

## Purpose
Quick regression coverage for the admin UI: navigation, user creation, impersonation,
and campaign creation.

## Preconditions
- Services running (`make run`)
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
  const displayName = "Playwright Smoke " + ts;
  const campaignName = "Playwright Campaign " + ts;

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

  const displayInput = page.locator("form[action=\"/users/create\"] input[name=\"display_name\"]");
  await displayInput.waitFor();
  await displayInput.fill(displayName);
  await page.locator("form[action=\"/users/create\"]").getByRole("button", { name: "Create" }).click();
  await page.getByRole("heading", { name: displayName, level: 2 }).waitFor();

  await page.getByRole("button", { name: "Impersonate" }).click();
  await page.getByText("Currently impersonating").waitFor();

  await page.getByRole("link", { name: "Campaigns" }).click();
  await page.getByRole("heading", { name: "Campaigns", level: 2 }).waitFor();
  await page.getByRole("link", { name: "Create Campaign" }).click();
  await page.getByRole("heading", { name: "Create Campaign", level: 2 }).waitFor();

  const campaignInput = page.locator("form[action=\"/campaigns/create\"] input[name=\"name\"]");
  await campaignInput.waitFor();
  await campaignInput.fill(campaignName);
  await page.getByRole("button", { name: "Create Campaign" }).click();
  await page.waitForURL(/\/campaigns\/[a-z0-9]+$/);
  await page.getByRole("heading", { name: campaignName, level: 2 }).waitFor();

  console.log("Created user: " + displayName);
  console.log("Created campaign: " + campaignName);
}
EOF
)"
```
