local scn = Scenario.new("environment_helms_deep_siege_siege_weapons")
local dh = scn:system("DAGGERHEART")

-- Model the siege weapons countdown breaching the walls.
scn:campaign{
  name = "Environment Helms Deep Siege Siege Weapons",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- Siege weapons grind down defenses.
scn:start_session("Siege Weapons")
dh:gm_fear(1)

dh:scene_countdown_create{ name = "Breach the Walls", kind = "consequence", current = 0, max = 6, direction = "increase" }
dh:scene_countdown_update{ name = "Breach the Walls", delta = 1, reason = "siege_weapon_strike" }
dh:gm_spend_fear(1):spotlight("Helms Deep Siege")

scn:end_session()

return scn
