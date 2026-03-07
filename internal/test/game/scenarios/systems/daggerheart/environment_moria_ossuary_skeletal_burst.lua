local scn = Scenario.new("environment_moria_ossuary_skeletal_burst")
local dh = scn:system("DAGGERHEART")

-- Model the skeletal burst shrapnel attack.
scn:campaign{
  name = "Environment Ossuary Skeletal Burst",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- The ossuary detonates around the party.
scn:start_session("Skeletal Burst")

dh:group_reaction{
  targets = {"Frodo"},
  trait = "agility",
  difficulty = 19,
  outcome = "fear",
  damage = 24,
  damage_type = "physical",
  source = "skeletal_burst"
}

scn:end_session()

return scn
