local scene = Scenario.new("companion_experience_stress_clear")

-- Model the companion experience that clears 1 Stress on return.
-- Clarification-gated fixture (P31): companion experience completion semantics are unresolved.
scene:campaign{
  name = "Companion Experience Stress Clear",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "companions"
}

scene:pc("Frodo")

-- A companion completes an experience and returns, clearing Stress.
scene:start_session("Companion Return")

-- Missing DSL: companion experience completion and stress clear on return.
scene:action_roll{ actor = "Frodo", trait = "presence", difficulty = 12, outcome = "hope" }

scene:end_session()

return scene
