local scene = Scenario.new("environment_gondor_court_imperial_decree")

-- Capture the imperial decree ticking a long-term countdown.
scene:campaign{
  name = "Environment Gondor Court Imperial Decree",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- The empire accelerates its agenda.
scene:start_session("Imperial Decree")
scene:gm_fear(1)

-- Partial mapping: fear spend, long-term countdown activation, and tick are explicit.
-- Missing DSL: randomized 1d4 long-term countdown advancement.
scene:gm_spend_fear(1):spotlight("Gondor Court", { description = "imperial_decree_advances_agenda" })
scene:countdown_create{ name = "Imperial Agenda", kind = "long_term", current = 0, max = 8, direction = "increase" }
scene:countdown_update{ name = "Imperial Agenda", delta = 2, reason = "imperial_decree_d4_proxy" }

scene:end_session()

return scene
