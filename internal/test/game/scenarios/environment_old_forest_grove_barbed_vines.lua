local scene = Scenario.new("environment_old_forest_grove_barbed_vines")

-- Model the Barbed Vines action in the Old Forest Grove.
scene:campaign{
  name = "Environment Old Forest Grove Barbed Vines",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- The grove lashes out with restraining vines.
scene:start_session("Barbed Vines")

-- Example: Agility reaction or take damage and become Restrained.
-- Escape checks after becoming Restrained remain unresolved in this fixture.
scene:group_reaction{
  targets = {"Frodo"},
  trait = "agility",
  difficulty = 11,
  outcome = "fear",
  damage = 8,
  damage_type = "physical",
  failure_conditions = {"RESTRAINED"},
  source = "barbed_vines"
}

scene:end_session()

return scene
