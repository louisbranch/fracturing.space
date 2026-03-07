local scene = Scenario.new("environment_dark_tower_usurpation_godslayer")
local dh = scene:system("DAGGERHEART")

-- Model the Godslayer action after the siege countdown triggers.
scene:campaign{
  name = "Environment Dark Tower Usurpation Godslayer",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
dh:adversary("Saruman")

-- The usurper slays a god and grows stronger.
scene:start_session("Godslayer")
dh:gm_fear(3)

-- Missing DSL: clear 2 HP and increase the usurper's stats after the action.
dh:gm_spend_fear(3):spotlight("Saruman")
dh:adversary_update{ target = "Saruman", evasion_delta = 1, notes = "godslayer_empowerment" }

scene:end_session()

return scene
