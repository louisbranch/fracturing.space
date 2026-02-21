local scene = Scenario.new("environment_old_forest_grove_not_welcome")

-- Model the grove guardians confronting intruders.
scene:campaign{
  name = "Environment Old Forest Grove Not Welcome",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
scene:adversary("Ent Warden")
scene:adversary("Woodland Elves")
scene:adversary("Ent Saplings")

-- The grove guardians appear to challenge the party.
scene:start_session("Not Welcome")
scene:gm_fear(1)

-- Missing DSL: spawn adversaries equal to party size and shift spotlight.
scene:adversary("Ent Sapling Reinforcement")
-- Party-size scaling and exact guardian mix remain unresolved.
scene:gm_spend_fear(1):spotlight("Ent Warden")

scene:end_session()

return scene
