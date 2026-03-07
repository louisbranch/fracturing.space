local scn = Scenario.new("environment_caradhras_pass_avalanche")
local dh = scn:system("DAGGERHEART")

-- Capture the avalanche action and reaction roll consequences.
scn:campaign{
  name = "Environment Caradhras Pass Avalanche",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- The GM triggers an avalanche down the pass.
scn:start_session("Avalanche")
dh:gm_fear(1)

-- Example: reaction roll or take 2d20 damage, knocked to Far, mark Stress.
-- Movement + stress consequences remain unresolved in the fixture DSL.
dh:gm_spend_fear(1):spotlight("Caradhras Pass")
dh:group_reaction{
  targets = {"Frodo"},
  trait = "agility",
  difficulty = 15,
  outcome = "fear",
  damage = 20,
  damage_type = "physical",
  source = "caradhras_avalanche"
}

scn:end_session()

return scn
