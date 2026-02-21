local scene = Scenario.new("environment_moria_ossuary_skeletal_burst")

-- Model the skeletal burst shrapnel attack.
scene:campaign{
  name = "Environment Ossuary Skeletal Burst",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- The ossuary detonates around the party.
scene:start_session("Skeletal Burst")

scene:group_reaction{
  targets = {"Frodo"},
  trait = "agility",
  difficulty = 19,
  outcome = "fear",
  damage = 24,
  damage_type = "physical",
  source = "skeletal_burst"
}

scene:end_session()

return scene
