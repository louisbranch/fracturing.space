local scene = Scenario.new("environment_waylayers_where_did_they_come_from")

-- Capture the ambushers' reaction granting advantage on the first strike.
scene:campaign{
  name = "Environment Orc Waylayers Where Did They Come From",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
scene:adversary("Orc Waylayers")

-- The PCs spring the trap, shifting the spotlight and boosting the first attack.
scene:start_session("Ambusher Reaction")
scene:gm_fear(2)

-- Example: lose 2 Fear and grant advantage on the first attack roll.
scene:gm_spend_fear(2):spotlight("Orc Waylayers")
scene:adversary_attack{
  actor = "Orc Waylayers",
  target = "Frodo",
  difficulty = 0,
  advantage = 1,
  damage_type = "physical"
}

scene:end_session()

return scene
