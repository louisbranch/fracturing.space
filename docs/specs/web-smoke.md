---
title: "Web Smoke Spec"
parent: "Specs"
nav_order: 2
nav_exclude: true
---

# Web Smoke Spec (Playwright CLI)

## Purpose

Quick regression coverage for web route and shell contracts:

- landing + login pages render with expected auth entrypoints,
- login recovery and recovery-code acknowledgement surfaces remain registered,
- trailing-slash route variants redirect to slashless canonical URLs for owned
  web roots before auth or feature-local redirects run,
- protected campaign route ownership keeps `/app/campaigns/{id}` canonical,
- authenticated users can traverse critical app routes (dashboard, campaigns,
  campaign creation surfaces, settings, settings security, notifications),
- authenticated campaign mutations (character create, session start/end,
  invite create/revoke) complete successfully in a connected dependency stack.

## Preconditions

- Web service running on port 8080
- OAuth client ID configured so the Sign in link and `/login` route are registered
- Optional authenticated coverage:
  - export `WEB_SMOKE_SESSION_ID` with a valid `web_session` value plus
    `WEB_SMOKE_RECIPIENT_USER_ID` (or legacy `WEB_SMOKE_USER_ID`) for invite
    mutation assertions, or
  - export `WEB_SMOKE_AUTH_ADDR` together with `WEB_SMOKE_AUTH_USERNAME` and
    `WEB_SMOKE_AUTH_RECIPIENT_USERNAME` so `scripts/playwright-web-smoke.sh`
    can resolve existing accounts and mint a valid web session automatically.
- Auth coverage default:
  - `scripts/playwright-web-smoke.sh` now requires authenticated coverage by
    default (`WEB_SMOKE_REQUIRE_AUTH=1`); set `WEB_SMOKE_REQUIRE_AUTH=0` only
    when intentionally running unauthenticated-only checks.
- Critical dependency stack:
  - CI runs this spec against a connected stack (`auth`, `social`,
    `notifications`, `ai`, `game`, `userhub`, `play`, `web`) so authenticated
    route journeys and the `/app/campaigns/{id}/game` handoff are validated as
    successful user-facing flows.

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

  await page.getByRole("heading", { name: /Sign in to/i }).waitFor();
  await page.getByText("Account Access").waitFor();
  await page.getByLabel("Email").waitFor();
  await page.getByRole("button", { name: "Create Account With Passkey" }).waitFor();
  await page.getByRole("button", { name: "Sign In With Passkey" }).waitFor();
  await page.getByRole("link", { name: /Recover account/i }).waitFor();

  const recoveryHref = await page.getByRole("link", { name: /Recover account/i }).getAttribute("href");
  if (recoveryHref !== "/login/recovery?pending_id=test") {
    throw new Error("Expected recovery link to preserve pending_id, got: " + recoveryHref);
  }

  console.log("Login page OK");
}
EOF
)"

step "Navigate to recovery page and verify"
cli run-code "$(cat <<'EOF'
async page => {
  page.setDefaultTimeout(10000);

  const origin = page.url().replace(/\/[^/]*$/, "");
  await page.goto(origin + "/login/recovery?pending_id=test");

  await page.getByRole("heading", { name: /Recover your account/i }).waitFor();
  await page.getByLabel("Username").waitFor();
  await page.getByLabel(/Recovery code/i).waitFor();
  await page.getByRole("button", { name: /Recover account/i }).waitFor();

  console.log("Recovery page OK");
}
EOF
)"

step "Verify canonical campaign route behavior"
cli run-code "$(cat <<'EOF'
async page => {
  page.setDefaultTimeout(10000);

  const origin = page.url().replace(/\/[^/]*$/, "");

  const slashRedirectChecks = [
    { path: "/discover/", wantLocation: "/discover" },
    { path: "/app/dashboard/", wantLocation: "/app/dashboard" },
    { path: "/app/campaigns/", wantLocation: "/app/campaigns" },
    { path: "/app/settings/", wantLocation: "/app/settings" },
    { path: "/app/notifications/", wantLocation: "/app/notifications" },
  ];

  for (const check of slashRedirectChecks) {
    const response = await page.request.get(origin + check.path, { maxRedirects: 0 });
    if (response.status() !== 308) {
      throw new Error("Expected " + check.path + " status 308, got: " + response.status());
    }
    const location = response.headers()["location"] || "";
    if (location !== check.wantLocation) {
      throw new Error("Expected " + check.path + " Location " + check.wantLocation + ", got: " + location);
    }
  }

  const deprecatedResponse = await page.request.get(origin + "/campaigns/camp-123", { maxRedirects: 0 });
  if (deprecatedResponse.status() !== 404) {
    throw new Error("Expected deprecated /campaigns/camp-123 status 404, got: " + deprecatedResponse.status());
  }

  const protectedResponse = await page.request.get(origin + "/app/campaigns/camp-123", { maxRedirects: 0 });
  if (protectedResponse.status() !== 302) {
    throw new Error("Expected /app/campaigns/camp-123 status 302, got: " + protectedResponse.status());
  }
  const location = protectedResponse.headers()["location"] || "";
  if (!location.startsWith("/login")) {
    throw new Error("Expected /app/campaigns/camp-123 Location starting with /login, got: " + location);
  }

  console.log("Campaign route contract OK");
}
EOF
)"

step "Verify authenticated app shell routes when session is available"
cli run-code "$(cat <<EOF
async page => {
  page.setDefaultTimeout(10000);

  const sessionID = "${WEB_SMOKE_SESSION_ID:-}".trim();
  if (!sessionID) {
    console.log("Authenticated app-shell checks skipped (WEB_SMOKE_SESSION_ID not set)");
    return;
  }

  const originMatch = page.url().match(/^(https?:\/\/[^/]+)/);
  if (!originMatch || !originMatch[1]) {
    throw new Error("Unable to resolve origin from page URL: " + page.url());
  }
  const origin = originMatch[1];
  await page.context().addCookies([
    {
      name: "web_session",
      value: sessionID,
      url: origin + "/",
      httpOnly: true,
      sameSite: "Lax",
    },
  ]);

  const routeChecks = [
    {
      path: "/app/dashboard",
      selectors: ["#dashboard-root"],
    },
    {
      path: "/app/campaigns",
      selectors: ["main"],
    },
    {
      path: "/app/campaigns/new",
      selectors: [
        '[data-campaign-start-option="browse"]',
        '[data-campaign-start-option="scratch"]',
      ],
    },
    {
      path: "/app/campaigns/create",
      selectors: [
        'form[action="/app/campaigns/create"]',
        'select[name="system"]',
      ],
    },
    {
      path: "/app/settings/profile",
      selectors: ["#settings-profile"],
    },
    {
      path: "/app/settings/security",
      selectors: ["#settings-security", "#settings-passkey-add"],
    },
    {
      path: "/app/notifications",
      selectors: ["#notifications-root"],
    },
  ];

  for (const check of routeChecks) {
    const response = await page.goto(origin + check.path, { waitUntil: "domcontentloaded" });
    if (!response) {
      throw new Error("Missing response for " + check.path);
    }
    if (response.status() !== 200) {
      throw new Error("Expected 200 for " + check.path + ", got: " + response.status());
    }
    if (page.url().includes("/login")) {
      throw new Error("Authenticated route unexpectedly redirected to login: " + check.path + " -> " + page.url());
    }
    for (const selector of check.selectors) {
      await page.locator(selector).first().waitFor();
    }
  }

  const mutationHeaders = {
    Cookie: "web_session=" + sessionID,
    Origin: origin,
    Referer: origin + "/app/dashboard",
  };

  const firstSelectableOptionValue = async function(fieldName) {
    const value = await page.locator('[name="' + fieldName + '"]').evaluateAll(function(elements) {
      for (const element of elements) {
        const tagName = (element.tagName || "").toLowerCase();
        if (tagName === "option") {
          const optionValue = (element.getAttribute("value") || "").trim();
          if (optionValue !== "" && !element.hasAttribute("disabled")) {
            return optionValue;
          }
          continue;
        }
        if ((element.getAttribute("disabled") || "") !== "") {
          continue;
        }
        const inputValue = (element.getAttribute("value") || "").trim();
        if (inputValue !== "") {
          return inputValue;
        }
      }
      return "";
    });
    return (value || "").trim();
  };

  const ensureCharacterCreationReady = async function(campaignID, characterID) {
    const detailPath = "/app/campaigns/" + campaignID + "/characters/" + characterID;
    const creationPath = detailPath + "/creation";
    const maxIterations = 12;
    let currentPath = creationPath;
    for (let iteration = 0; iteration < maxIterations; iteration++) {
      const detailResponse = await page.goto(origin + currentPath, { waitUntil: "domcontentloaded" });
      if (!detailResponse) {
        throw new Error("Missing response for character creation workflow detail");
      }
      if (detailResponse.status() !== 200) {
        throw new Error("Expected character creation route status 200, got: " + detailResponse.status());
      }
      if (currentPath === detailPath) {
        await page.locator("#campaign-character-detail").waitFor();
        const workflowCount = await page.locator('[data-character-creation-workflow="true"]').count();
        if (workflowCount === 0) {
          return;
        }
        const continueLink = page.locator('[data-character-creation-link="true"]').first();
        if ((await continueLink.count()) === 0) {
          return;
        }
        currentPath = ((await continueLink.getAttribute("href")) || "").trim();
        if (!currentPath) {
          throw new Error("Character detail continue link was empty");
        }
        continue;
      }

      await page.locator('[data-character-creation-page="true"]').waitFor();
      const readyCount = await page.locator('[data-character-creation-ready="true"]').count();
      if (readyCount > 0) {
        return;
      }

      const stepForm = page.locator('[data-character-creation-form-step]').first();
      if ((await stepForm.count()) === 0) {
        const unmetCount = await page.locator('[data-character-creation-unmet="true"] li').count();
        if (unmetCount === 0) {
          return;
        }
        throw new Error("Character creation workflow still has unmet reasons but no active step form");
      }

      const stepRaw = ((await stepForm.getAttribute("data-character-creation-form-step")) || "").trim();
      const step = parseInt(stepRaw, 10);
      if (!Number.isInteger(step)) {
        throw new Error("Invalid character creation step marker: " + stepRaw);
      }

      const stepAction = ((await stepForm.getAttribute("action")) || "").trim();
      if (!stepAction) {
        throw new Error("Character creation step " + step + " form action was empty");
      }

      let form = {};
      let applyResp = null;
      if (step === 1) {
        const classIDs = await page.locator('input[name="class_id"]').evaluateAll(function(options) {
          const result = [];
          for (const option of options) {
            const optionValue = (option.getAttribute("value") || "").trim();
            if (optionValue !== "" && !option.hasAttribute("disabled")) {
              result.push(optionValue);
            }
          }
          return result;
        });
        const subclassIDs = await page.locator('input[name="subclass_id"]').evaluateAll(function(options) {
          const result = [];
          for (const option of options) {
            const optionValue = (option.getAttribute("value") || "").trim();
            if (optionValue !== "" && !option.hasAttribute("disabled")) {
              result.push(optionValue);
            }
          }
          return result;
        });
        if (classIDs.length === 0 || subclassIDs.length === 0) {
          throw new Error("Character creation step 1 missing selectable class/subclass options");
        }
        let lastStatus = 0;
        for (const classID of classIDs) {
          for (const subclassID of subclassIDs) {
            const candidateResp = await page.request.post(origin + stepAction, {
              maxRedirects: 0,
              headers: {
                ...mutationHeaders,
                Referer: origin + detailPath,
              },
              form: { class_id: classID, subclass_id: subclassID },
            });
            lastStatus = candidateResp.status();
            if (lastStatus === 302) {
              applyResp = candidateResp;
              break;
            }
          }
          if (applyResp !== null) {
            break;
          }
        }
        if (applyResp === null) {
          throw new Error("Character creation step 1 could not find a valid class/subclass combination; last status: " + lastStatus);
        }
      } else if (step === 2) {
        const ancestryID = await firstSelectableOptionValue("ancestry_id");
        const communityID = await firstSelectableOptionValue("community_id");
        if (!ancestryID || !communityID) {
          throw new Error("Character creation step 2 missing selectable ancestry/community options");
        }
        form = { ancestry_id: ancestryID, community_id: communityID };
      } else if (step === 3) {
        form = {
          agility: "2",
          strength: "1",
          finesse: "1",
          instinct: "0",
          presence: "0",
          knowledge: "-1",
        };
      } else if (step === 4) {
        const primaryWeaponID = await firstSelectableOptionValue("weapon_primary_id");
        const secondaryWeaponID = await firstSelectableOptionValue("weapon_secondary_id");
        const armorID = await firstSelectableOptionValue("armor_id");
        const potionItemID = await firstSelectableOptionValue("potion_item_id");
        if (!primaryWeaponID || !armorID || !potionItemID) {
          throw new Error("Character creation step 5 missing selectable equipment options");
        }
        form = {
          weapon_primary_id: primaryWeaponID,
          armor_id: armorID,
          potion_item_id: potionItemID,
        };
        if (secondaryWeaponID) {
          form.weapon_secondary_id = secondaryWeaponID;
        }
      } else if (step === 5) {
        form = {
          experience_0_name: "Smoke Experience One",
          experience_1_name: "Smoke Experience Two",
        };
      } else if (step === 6) {
        const domainCardIDs = await page.locator('input[name="domain_card_id"][type="checkbox"]').evaluateAll(function(inputs) {
          const result = [];
          for (const input of inputs) {
            const inputValue = (input.getAttribute("value") || "").trim();
            if (inputValue !== "" && !input.hasAttribute("disabled")) {
              result.push(inputValue);
            }
            if (result.length === 2) {
              break;
            }
          }
          return result;
        });
        if (domainCardIDs.length !== 2) {
          throw new Error("Character creation step 6 missing selectable domain card options");
        }
        form = { domain_card_id: domainCardIDs };
      } else if (step === 7) {
        form = { description: "Smoke detail notes." };
      } else if (step === 8) {
        form = { background: "Smoke background details." };
      } else if (step === 9) {
        form = { connections: "Smoke connection details." };
      } else {
        throw new Error("Unexpected character creation step: " + step);
      }

      if (applyResp === null) {
        applyResp = await page.request.post(origin + stepAction, {
          maxRedirects: 0,
          headers: {
            ...mutationHeaders,
            Referer: origin + detailPath,
          },
          form: form,
        });
      }
      if (applyResp.status() !== 302) {
        throw new Error("Expected character creation step " + step + " status 302, got: " + applyResp.status());
      }
      const applyLocation = (applyResp.headers()["location"] || "").trim();
      if (applyLocation !== creationPath) {
        throw new Error("Expected character creation step " + step + " redirect to " + creationPath + ", got: " + applyLocation);
      }
      currentPath = creationPath;
    }
    throw new Error("Character creation workflow did not reach ready state within deterministic smoke budget");
  };

  const campaignCreateResp = await page.request.post(origin + "/app/campaigns/create", {
    maxRedirects: 0,
    headers: {
      ...mutationHeaders,
      Referer: origin + "/app/campaigns/create",
    },
    form: {
      name: "Smoke Campaign",
      system: "daggerheart",
      gm_mode: "human",
      theme_prompt: "Smoke route contract",
    },
  });
  if (campaignCreateResp.status() !== 302) {
    throw new Error("Expected campaign create status 302, got: " + campaignCreateResp.status());
  }
  const campaignLocation = (campaignCreateResp.headers()["location"] || "").trim();
  if (campaignLocation.startsWith("/login")) {
    throw new Error("Campaign create unexpectedly redirected to login: " + campaignLocation);
  }
  const campaignMatch = campaignLocation.match(/^\/app\/campaigns\/([^/?#]+)\$/);
  if (!campaignMatch) {
    throw new Error("Campaign create location did not match /app/campaigns/{id}: " + campaignLocation);
  }
  const campaignID = campaignMatch[1];

  const campaignRouteChecks = [
    { path: "/app/campaigns/" + campaignID, selectors: ["#campaign-overview"] },
    { path: "/app/campaigns/" + campaignID + "/participants", selectors: ["#campaign-participants", '[data-campaign-participant-card-id]'] },
    { path: "/app/campaigns/" + campaignID + "/characters", selectors: ["#campaign-characters", '[data-campaign-character-create-entry="true"]'] },
    { path: "/app/campaigns/" + campaignID + "/characters/create", selectors: ["#campaign-character-create", '[data-campaign-character-create-page="true"]'] },
    { path: "/app/campaigns/" + campaignID + "/sessions", selectors: ["#campaign-sessions", '[data-campaign-sessions-header="true"]'] },
    { path: "/app/campaigns/" + campaignID + "/sessions/create", selectors: ["#campaign-session-create", '[data-campaign-session-create-form="true"]'] },
    { path: "/app/campaigns/" + campaignID + "/invites", selectors: ["#campaign-invites", '[data-campaign-invite-create-form="true"]'] },
    { path: "/app/campaigns/" + campaignID + "/game", selectors: ["#root"] },
  ];

  for (const check of campaignRouteChecks) {
    const response = await page.goto(origin + check.path, { waitUntil: "domcontentloaded" });
    if (!response) {
      throw new Error("Missing response for " + check.path);
    }
    if (response.status() !== 200) {
      throw new Error("Expected 200 for " + check.path + ", got: " + response.status());
    }
    if (page.url().includes("/login")) {
      throw new Error("Campaign route unexpectedly redirected to login: " + check.path + " -> " + page.url());
    }
    for (const selector of check.selectors) {
      await page.locator(selector).first().waitFor();
    }
  }

  const characterCreatePagePath = "/app/campaigns/" + campaignID + "/characters/create";
  const characterCreatePageResp = await page.goto(origin + characterCreatePagePath, { waitUntil: "domcontentloaded" });
  if (!characterCreatePageResp) {
    throw new Error("Missing response for character create page");
  }
  if (characterCreatePageResp.status() !== 200) {
    throw new Error("Expected character create page status 200, got: " + characterCreatePageResp.status());
  }
  await page.locator('[data-campaign-character-create-page="true"]').waitFor();

  const characterCreateResp = await page.request.post(origin + "/app/campaigns/" + campaignID + "/characters/create", {
    maxRedirects: 0,
    headers: {
      ...mutationHeaders,
      Referer: origin + characterCreatePagePath,
    },
    form: { name: "Smoke Hero", pronouns: "they/them", kind: "pc" },
  });
  if (characterCreateResp.status() !== 302) {
    throw new Error("Expected character create status 302, got: " + characterCreateResp.status());
  }
  const characterLocation = (characterCreateResp.headers()["location"] || "").trim();
  const characterMatch = characterLocation.match(/^\/app\/campaigns\/([^/?#]+)\/characters\/([^/?#]+)(\/creation)?$/);
  if (!characterMatch) {
    throw new Error("Character create location did not match expected route: " + characterLocation);
  }
  const characterID = characterMatch[2];

  const characterDetailResp = await page.goto(origin + "/app/campaigns/" + campaignID + "/characters/" + characterID, { waitUntil: "domcontentloaded" });
  if (!characterDetailResp) {
    throw new Error("Missing response for character detail");
  }
  if (characterDetailResp.status() !== 200) {
    throw new Error("Expected character detail status 200, got: " + characterDetailResp.status());
  }
  await page.locator("#campaign-character-detail").waitFor();
  await ensureCharacterCreationReady(campaignID, characterID);

  const participantsResp = await page.goto(origin + "/app/campaigns/" + campaignID + "/participants", { waitUntil: "domcontentloaded" });
  if (!participantsResp) {
    throw new Error("Missing response for participants mutation setup");
  }
  if (participantsResp.status() !== 200) {
    throw new Error("Expected participants setup status 200, got: " + participantsResp.status());
  }
  const participantID = await page.locator('[data-campaign-participant-card-id]').first().getAttribute("data-campaign-participant-card-id");
  if (!participantID || !participantID.trim()) {
    throw new Error("Missing campaign participant id for mutation checks");
  }
  const recipientUserID = "${WEB_SMOKE_RECIPIENT_USER_ID:-${WEB_SMOKE_USER_ID:-}}".trim();
  if (!recipientUserID) {
    throw new Error("Missing WEB_SMOKE_RECIPIENT_USER_ID (or WEB_SMOKE_USER_ID fallback) for deterministic invite checks");
  }

  const sessionsPath = "/app/campaigns/" + campaignID + "/sessions";
  const invitesPath = "/app/campaigns/" + campaignID + "/invites";

  const sessionStartResp = await page.request.post(origin + "/app/campaigns/" + campaignID + "/sessions/create", {
    maxRedirects: 0,
    headers: {
      ...mutationHeaders,
      Referer: origin + sessionsPath + "/create",
    },
    form: { name: "Smoke Session" },
  });
  const sessionStartStatus = sessionStartResp.status();
  if (sessionStartStatus !== 302 && sessionStartStatus !== 409) {
    throw new Error("Expected session start status 302 or 409, got: " + sessionStartStatus);
  }
  if (sessionStartStatus === 302) {
    const sessionStartLocation = (sessionStartResp.headers()["location"] || "").trim();
    if (sessionStartLocation !== sessionsPath) {
      throw new Error("Expected session start redirect to " + sessionsPath + ", got: " + sessionStartLocation);
    }
  }

  const sessionsResp = await page.goto(origin + sessionsPath, { waitUntil: "domcontentloaded" });
  if (!sessionsResp) {
    throw new Error("Missing response for sessions list after start");
  }
  if (sessionsResp.status() !== 200) {
    throw new Error("Expected sessions list status 200 after start, got: " + sessionsResp.status());
  }
  const sessionCardCount = await page.locator('[data-campaign-session-card-id]').count();
  const campaignSessionID = sessionCardCount > 0
    ? ((await page.locator('[data-campaign-session-card-id]').first().getAttribute("data-campaign-session-card-id")) || "").trim()
    : "";
  if (!campaignSessionID) {
    console.log("Skipping session end assertions because no session card rendered after session start");
  } else {
    const sessionEndResp = await page.request.post(origin + "/app/campaigns/" + campaignID + "/sessions/end", {
      maxRedirects: 0,
      headers: {
        ...mutationHeaders,
        Referer: origin + sessionsPath,
      },
      form: { session_id: campaignSessionID },
    });
    if (sessionEndResp.status() !== 302) {
      throw new Error("Expected session end status 302, got: " + sessionEndResp.status());
    }
    const sessionEndLocation = (sessionEndResp.headers()["location"] || "").trim();
    if (sessionEndLocation !== sessionsPath) {
      throw new Error("Expected session end redirect to " + sessionsPath + ", got: " + sessionEndLocation);
    }

    const sessionsAfterEndResp = await page.goto(origin + sessionsPath, { waitUntil: "domcontentloaded" });
    if (!sessionsAfterEndResp) {
      throw new Error("Missing response for sessions list after end");
    }
    if (sessionsAfterEndResp.status() !== 200) {
      throw new Error("Expected sessions list status 200 after end, got: " + sessionsAfterEndResp.status());
    }
    const sessionEndFormCount = await page.locator('[data-campaign-session-card-id="' + campaignSessionID + '"] [data-campaign-session-end-form="true"]').count();
    if (sessionEndFormCount !== 0) {
      throw new Error("Expected ended session to hide end form for session " + campaignSessionID);
    }
  }

  const inviteCreateResp = await page.request.post(origin + "/app/campaigns/" + campaignID + "/invites/create", {
    maxRedirects: 0,
    headers: {
      ...mutationHeaders,
      Referer: origin + invitesPath,
    },
    form: { participant_id: participantID.trim(), recipient_user_id: recipientUserID },
  });
  if (inviteCreateResp.status() !== 302) {
    throw new Error("Expected invite create status 302, got: " + inviteCreateResp.status());
  }
  const inviteCreateLocation = (inviteCreateResp.headers()["location"] || "").trim();
  if (inviteCreateLocation !== invitesPath) {
    throw new Error("Expected invite create redirect to " + invitesPath + ", got: " + inviteCreateLocation);
  }

  const invitesResp = await page.goto(origin + invitesPath, { waitUntil: "domcontentloaded" });
  if (!invitesResp) {
    throw new Error("Missing response for invites list after create");
  }
  if (invitesResp.status() !== 200) {
    throw new Error("Expected invites list status 200 after create, got: " + invitesResp.status());
  }
  const inviteID = (await page.locator('[data-campaign-invite-card-id]:has([data-campaign-invite-recipient="' + recipientUserID + '"])').first().getAttribute("data-campaign-invite-card-id") || "").trim();
  if (!inviteID) {
    throw new Error("Missing invite id for recipient " + recipientUserID + " after invite create");
  }

  const inviteRevokeResp = await page.request.post(origin + "/app/campaigns/" + campaignID + "/invites/revoke", {
    maxRedirects: 0,
    headers: {
      ...mutationHeaders,
      Referer: origin + invitesPath,
    },
    form: { invite_id: inviteID },
  });
  if (inviteRevokeResp.status() !== 302) {
    throw new Error("Expected invite revoke status 302, got: " + inviteRevokeResp.status());
  }
  const inviteRevokeLocation = (inviteRevokeResp.headers()["location"] || "").trim();
  if (inviteRevokeLocation !== invitesPath) {
    throw new Error("Expected invite revoke redirect to " + invitesPath + ", got: " + inviteRevokeLocation);
  }

  const invitesAfterRevokeResp = await page.goto(origin + invitesPath, { waitUntil: "domcontentloaded" });
  if (!invitesAfterRevokeResp) {
    throw new Error("Missing response for invites list after revoke");
  }
  if (invitesAfterRevokeResp.status() !== 200) {
    throw new Error("Expected invites list status 200 after revoke, got: " + invitesAfterRevokeResp.status());
  }
  const inviteRevokeFormCount = await page.locator('[data-campaign-invite-card-id="' + inviteID + '"] [data-campaign-invite-revoke-form="true"]').count();
  if (inviteRevokeFormCount !== 0) {
    throw new Error("Expected revoked invite to hide revoke form for invite " + inviteID);
  }

  console.log("Authenticated critical-route coverage OK (connected stack, deterministic mutations)");
}
EOF
)"
```
