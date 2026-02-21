local scene = Scenario.new("environment_mirkwood_blight_grasping_vines")

-- Model the grasping vines restrain + vulnerable action.
scene:campaign{
  name = "Environment Mirkwood Blight Grasping Vines",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Gandalf")

-- Vines whip out and bind a target.
scene:start_session("Grasping Vines")

-- Escape-roll follow-up damage and Hope loss remain unresolved in this fixture.
scene:group_reaction{
  targets = {"Gandalf"},
  trait = "agility",
  difficulty = 16,
  outcome = "fear",
  failure_conditions = {"RESTRAINED", "VULNERABLE"},
  source = "grasping_vines"
}

scene:end_session()

return scene
