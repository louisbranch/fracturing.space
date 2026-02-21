local scene = Scenario.new("environment_caradhras_pass_avalanche")

-- Capture the avalanche action and reaction roll consequences.
scene:campaign{
  name = "Environment Caradhras Pass Avalanche",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- The GM triggers an avalanche down the pass.
scene:start_session("Avalanche")
scene:gm_fear(1)

-- Example: reaction roll or take 2d20 damage, knocked to Far, mark Stress.
-- Movement + stress consequences remain unresolved in the fixture DSL.
scene:gm_spend_fear(1):spotlight("Caradhras Pass")
scene:group_reaction{
  targets = {"Frodo"},
  trait = "agility",
  difficulty = 15,
  outcome = "fear",
  damage = 20,
  damage_type = "physical",
  source = "caradhras_avalanche"
}

scene:end_session()

return scene
