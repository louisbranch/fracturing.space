local scn = Scenario.new("environment_gondor_court_imperial_decree")
local dh = scn:system("DAGGERHEART")

-- Capture the imperial decree ticking a long-term countdown.
scn:campaign{
  name = "Environment Gondor Court Imperial Decree",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- The empire accelerates its agenda.
scn:start_session("Imperial Decree")
dh:gm_fear(1)

-- Partial mapping: fear spend, long-term countdown activation, and tick are explicit.
-- Missing DSL: randomized 1d4 long-term countdown advancement.
dh:gm_spend_fear(1):spotlight("Gondor Court", { description = "imperial_decree_advances_agenda" })
dh:scene_countdown_create{ name = "Imperial Agenda", kind = "long_term", current = 0, max = 8, direction = "increase" }
dh:scene_countdown_update{ name = "Imperial Agenda", delta = 2, reason = "imperial_decree_d4_proxy" }

scn:end_session()

return scn
