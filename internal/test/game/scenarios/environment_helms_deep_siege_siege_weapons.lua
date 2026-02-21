local scene = Scenario.new("environment_helms_deep_siege_siege_weapons")

-- Model the siege weapons countdown breaching the walls.
scene:campaign{
  name = "Environment Helms Deep Siege Siege Weapons",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- Siege weapons grind down defenses.
scene:start_session("Siege Weapons")
scene:gm_fear(1)

scene:countdown_create{ name = "Breach the Walls", kind = "consequence", current = 0, max = 6, direction = "increase" }
scene:countdown_update{ name = "Breach the Walls", delta = 1, reason = "siege_weapon_strike" }
scene:gm_spend_fear(1):spotlight("Helms Deep Siege")

scene:end_session()

return scene
