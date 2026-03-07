local scn = Scenario.new("environment_mirkwood_blight_grasping_vines")
local dh = scn:system("DAGGERHEART")

-- Model the grasping vines restrain + vulnerable action.
scn:campaign{
  name = "Environment Mirkwood Blight Grasping Vines",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Gandalf")

-- Vines whip out and bind a target.
scn:start_session("Grasping Vines")

-- Escape-roll follow-up damage and Hope loss remain unresolved in this fixture.
dh:group_reaction{
  targets = {"Gandalf"},
  trait = "agility",
  difficulty = 16,
  outcome = "fear",
  failure_conditions = {"RESTRAINED", "VULNERABLE"},
  source = "grasping_vines"
}

scn:end_session()

return scn
