local scene = Scenario.new("environment_helms_deep_siege_collateral_damage")

-- Model collateral damage from siege weapons after an adversary falls.
scene:campaign{
  name = "Environment Helms Deep Siege Collateral Damage",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- A stray attack lands where the fight rages.
scene:start_session("Collateral Damage")
scene:gm_fear(1)

-- Stress-on-success/failure remains unresolved in the fixture DSL.
scene:gm_spend_fear(1):spotlight("Helms Deep Siege")
scene:group_reaction{
  targets = {"Frodo"},
  trait = "agility",
  difficulty = 17,
  outcome = "fear",
  damage = 15,
  damage_type = "physical",
  source = "collateral_damage"
}

scene:end_session()

return scene
