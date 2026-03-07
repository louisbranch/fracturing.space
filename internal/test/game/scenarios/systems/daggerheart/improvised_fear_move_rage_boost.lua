local scene = Scenario.new("improvised_fear_move_rage_boost")
local dh = scene:system("DAGGERHEART")

-- Model the improvised fear move that boosts a solo adversary's damage.
scene:campaign{
  name = "Improvised Fear Move Rage Boost",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "gm_fear"
}

scene:pc("Frodo")
dh:adversary("Uruk-hai Brute")

-- The GM spends Fear to increase a solo adversary's damage output.
scene:start_session("Rage Boost")
dh:gm_fear(2)

-- Example: the adversary flies into a rage for the remainder of the scene.
-- Partial mapping: fear spend, temporary power marker, and boosted strike are explicit.
-- Missing DSL: first-class temporary damage bonus duration semantics.
dh:gm_spend_fear(1):spotlight("Uruk-hai Brute", { description = "rage_boost_empowerment" })
dh:adversary_update{ target = "Uruk-hai Brute", stress_delta = 1, notes = "rage_damage_bonus_active" }
dh:adversary_attack{
  actor = "Uruk-hai Brute",
  target = "Frodo",
  difficulty = 0,
  attack_modifier = 2,
  damage_dice = {{count = 2, sides = 8}},
  damage_type = "physical"
}
scene:set_spotlight{ target = "Frodo" }

-- Close the session after the fear move.
scene:end_session()

return scene
