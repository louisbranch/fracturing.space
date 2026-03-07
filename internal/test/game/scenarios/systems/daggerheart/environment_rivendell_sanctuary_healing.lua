local scn = Scenario.new("environment_rivendell_sanctuary_healing")
local dh = scn:system("DAGGERHEART")

-- Model the automatic healing from resting in the temple.
scn:campaign{
  name = "Environment Rivendell Sanctuary Healing",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- Resting in the temple clears HP.
scn:start_session("Temple Rest")

-- Missing DSL: clear all HP on rest in this environment.
dh:rest{ type = "short", party_size = 1 }

scn:end_session()

return scn
