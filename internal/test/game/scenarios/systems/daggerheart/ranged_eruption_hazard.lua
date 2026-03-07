local scene = Scenario.new("ranged_eruption_hazard")
local dh = scene:system("DAGGERHEART")

-- Model the Saruman's eruption hazard action.
scene:campaign{
  name = "Ranged Eruption Hazard",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary"
}

scene:pc("Frodo")
dh:adversary("Saruman")

-- The wizard spends Fear to erupt terrain and force reaction rolls.
scene:start_session("Eruption")
dh:gm_fear(1)

-- Example: targets roll Agility 14 or take 2d10 damage and are moved.
dh:gm_spend_fear(1):spotlight("Saruman")
dh:group_reaction{
  targets = {"Frodo"},
  trait = "agility",
  difficulty = 14,
  damage = 10,
  damage_type = "physical",
  half_damage_on_success = true,
  source = "eruption_hazard"
}
-- Missing DSL: forced movement/range-band reposition metadata per target.

scene:end_session()

return scene
