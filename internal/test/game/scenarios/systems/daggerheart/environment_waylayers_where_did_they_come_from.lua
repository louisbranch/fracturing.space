local scn = Scenario.new("environment_waylayers_where_did_they_come_from")
local dh = scn:system("DAGGERHEART")

-- Capture the ambushers' reaction granting advantage on the first strike.
scn:campaign{
  name = "Environment Orc Waylayers Where Did They Come From",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
dh:adversary("Orc Waylayers")

-- The PCs spring the trap, shifting the spotlight and boosting the first attack.
scn:start_session("Ambusher Reaction")
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

scn:end_session()

return scn
