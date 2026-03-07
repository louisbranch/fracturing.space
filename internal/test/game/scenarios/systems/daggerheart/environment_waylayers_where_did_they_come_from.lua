local scene = Scenario.new("environment_waylayers_where_did_they_come_from")
local dh = scene:system("DAGGERHEART")

-- Capture the ambushers' reaction granting advantage on the first strike.
scene:campaign{
  name = "Environment Orc Waylayers Where Did They Come From",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
dh:adversary("Orc Waylayers")

-- The PCs spring the trap, shifting the spotlight and boosting the first attack.
scene:start_session("Ambusher Reaction")
dh:gm_fear(2)

-- Example: lose 2 Fear and grant advantage on the first attack roll.
dh:gm_spend_fear(2):spotlight("Orc Waylayers")
dh:adversary_attack{
  actor = "Orc Waylayers",
  target = "Frodo",
  difficulty = 0,
  advantage = 1,
  damage_type = "physical"
}

scene:end_session()

return scene
