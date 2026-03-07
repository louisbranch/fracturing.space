local scn = Scenario.new("environment_dark_tower_usurpation_godslayer")
local dh = scn:system("DAGGERHEART")

-- Model the Godslayer action after the siege countdown triggers.
scn:campaign{
  name = "Environment Dark Tower Usurpation Godslayer",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
dh:adversary("Saruman")

-- The usurper slays a god and grows stronger.
scn:start_session("Godslayer")
dh:gm_fear(3)

-- Missing DSL: clear 2 HP and increase the usurper's stats after the action.
dh:gm_spend_fear(3):spotlight("Saruman")
dh:adversary_update{ target = "Saruman", evasion_delta = 1, notes = "godslayer_empowerment" }

scn:end_session()

return scn
