local scn = Scenario.new("environment_old_forest_grove_defiler")
local dh = scn:system("DAGGERHEART")

-- Capture the Defiler fear action summoning a chaos elemental.
scn:campaign{
  name = "Environment Old Forest Grove Defiler",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
dh:adversary("Shadow Wraith")

-- The grove draws in a chaotic threat.
scn:start_session("Defiler")
dh:gm_fear(1)

-- Example: spend Fear to summon an elemental that immediately takes spotlight.
dh:adversary("Chaos Elemental")
-- Chosen-PC proximity semantics remain unresolved in this fixture.
dh:gm_spend_fear(1):spotlight("Chaos Elemental")

scn:end_session()

return scn
