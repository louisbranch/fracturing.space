local scn = Scenario.new("environment_old_forest_grove_not_welcome")
local dh = scn:system("DAGGERHEART")

-- Model the grove guardians confronting intruders.
scn:campaign{
  name = "Environment Old Forest Grove Not Welcome",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
dh:adversary("Ent Warden")
dh:adversary("Woodland Elves")
dh:adversary("Ent Saplings")

-- The grove guardians appear to challenge the party.
scn:start_session("Not Welcome")
dh:gm_fear(1)

-- Missing DSL: spawn adversaries equal to party size and shift spotlight.
dh:adversary("Ent Sapling Reinforcement")
-- Party-size scaling and exact guardian mix remain unresolved.
dh:gm_spend_fear(1):spotlight("Ent Warden")

scn:end_session()

return scn
