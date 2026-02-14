# Web Smoke Spec (Playwright CLI)

## Purpose

Quick regression coverage for the web UI: landing page renders with branding
and sign-in link, login page renders with passkey form elements.

## Preconditions

- Web service running on port 8086 (standalone, no auth service required)
- OAuth client ID configured so the Sign in link and `/login` route are registered

## Automation

Run via `scripts/playwright-web-smoke.sh` or the spec runner directly.

```playwright-cli
step "Open landing page"
open_browser

step "Verify landing page"
cli run-code "$(cat <<'EOF'
async page => {
  page.setDefaultTimeout(10000);

  await page.getByRole("heading", { name: "Fracturing.Space", level: 1 }).waitFor();
  await page.getByText("Open-source, server-authoritative engine").waitFor();
  await page.getByRole("link", { name: "Sign in" }).waitFor();

  const signInLink = page.getByRole("link", { name: "Sign in" });
  const href = await signInLink.getAttribute("href");
  if (href !== "/auth/login") {
    throw new Error("Expected Sign in href to be /auth/login, got: " + href);
  }

  console.log("Landing page OK");
}
EOF
)"

step "Navigate to login page and verify"
cli run-code "$(cat <<'EOF'
async page => {
  page.setDefaultTimeout(10000);

  const origin = page.url().replace(/\/[^/]*$/, "");
  await page.goto(origin + "/login?pending_id=test&client_id=test&client_name=Test");

  await page.getByRole("heading", { name: "Sign in to continue" }).waitFor();
  await page.getByText("Account Access").waitFor();
  await page.getByLabel("Username").waitFor();
  await page.getByRole("button", { name: "Create Account With Passkey" }).waitFor();
  await page.getByRole("button", { name: "Sign In With Passkey" }).waitFor();

  console.log("Login page OK");
}
EOF
)"
```
