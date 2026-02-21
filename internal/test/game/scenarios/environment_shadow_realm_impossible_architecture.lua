local scene = Scenario.new("environment_shadow_realm_impossible_architecture")

-- Model navigation through impossible architecture with a progress countdown.
scene:campaign{
  name = "Environment Shadow Realm Impossible Architecture",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- Movement requires progress across shifting gravity.
scene:start_session("Impossible Architecture")

-- Missing DSL: apply progress countdown (8) and stress on failure.
scene:countdown_create{ name = "Chaos Traverse", kind = "progress", current = 0, max = 8, direction = "increase" }
scene:action_roll{ actor = "Frodo", trait = "agility", difficulty = 20, outcome = "success_fear" }
scene:apply_roll_outcome{
  on_success = {
    {kind = "countdown_update", name = "Chaos Traverse", delta = 1, reason = "navigate_shift"},
  },
}

scene:end_session()

return scene
