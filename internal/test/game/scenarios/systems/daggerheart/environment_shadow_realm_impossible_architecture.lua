local scn = Scenario.new("environment_shadow_realm_impossible_architecture")
local dh = scn:system("DAGGERHEART")

-- Model navigation through impossible architecture with a progress countdown.
scn:campaign{
  name = "Environment Shadow Realm Impossible Architecture",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- Movement requires progress across shifting gravity.
scn:start_session("Impossible Architecture")

-- Missing DSL: apply progress countdown (8) and stress on failure.
dh:countdown_create{ name = "Chaos Traverse", kind = "progress", current = 0, max = 8, direction = "increase" }
dh:action_roll{ actor = "Frodo", trait = "agility", difficulty = 20, outcome = "success_fear" }
dh:apply_roll_outcome{
  on_success = {
    {kind = "countdown_update", name = "Chaos Traverse", delta = 1, reason = "navigate_shift"},
  },
}

scn:end_session()

return scn
