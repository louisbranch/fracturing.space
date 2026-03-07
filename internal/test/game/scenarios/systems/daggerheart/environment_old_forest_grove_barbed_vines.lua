local scn = Scenario.new("environment_old_forest_grove_barbed_vines")
local dh = scn:system("DAGGERHEART")

-- Model the Barbed Vines action in the Old Forest Grove.
scn:campaign{
  name = "Environment Old Forest Grove Barbed Vines",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- The grove lashes out with restraining vines.
scn:start_session("Barbed Vines")

-- Example: Agility reaction or take damage and become Restrained.
-- Escape checks after becoming Restrained remain unresolved in this fixture.
dh:group_reaction{
  targets = {"Frodo"},
  trait = "agility",
  difficulty = 11,
  outcome = "fear",
  damage = 8,
  damage_type = "physical",
  failure_conditions = {"RESTRAINED"},
  source = "barbed_vines"
}

scn:end_session()

return scn
