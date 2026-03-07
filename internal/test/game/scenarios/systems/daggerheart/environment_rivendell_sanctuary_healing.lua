local scene = Scenario.new("environment_rivendell_sanctuary_healing")
local dh = scene:system("DAGGERHEART")

-- Model the automatic healing from resting in the temple.
scene:campaign{
  name = "Environment Rivendell Sanctuary Healing",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- Resting in the temple clears HP.
scene:start_session("Temple Rest")

-- Missing DSL: clear all HP on rest in this environment.
dh:rest{ type = "short", party_size = 1 }

scene:end_session()

return scene
