local scn = Scenario.new("companion_experience_stress_clear")
local dh = scn:system("DAGGERHEART")

-- Model the companion experience that clears 1 Stress on return.
-- Clarification-gated fixture (P31): companion experience completion semantics are unresolved.
scn:campaign{
  name = "Companion Experience Stress Clear",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "companions"
}

scn:pc("Frodo")

-- A companion completes an experience and returns, clearing Stress.
scn:start_session("Companion Return")

-- Missing DSL: companion experience completion and stress clear on return.
dh:action_roll{ actor = "Frodo", trait = "presence", difficulty = 12, outcome = "hope" }

scn:end_session()

return scn
