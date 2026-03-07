local scene = Scenario.new("environment_bruinen_ford_patient_hunter")
local dh = scene:system("DAGGERHEART")

-- Capture the river's Patient Hunter fear action summoning a predator.
scene:campaign{
  name = "Environment Bruinen Ford Patient Hunter",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
dh:adversary("Warg")

-- The GM spends Fear to summon a Warg.
scene:start_session("Patient Hunter")
dh:gm_fear(1)

-- Example: summon the Warg within Close range and immediately spotlight it.
dh:adversary("Warg Hunter")
-- Range placement remains unresolved in this fixture.
dh:gm_spend_fear(1):spotlight("Warg Hunter")
dh:adversary_attack{ actor = "Warg Hunter", target = "Frodo", difficulty = 0, damage_type = "physical" }

scene:end_session()

return scene
