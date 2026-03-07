local scn = Scenario.new("environment_helms_deep_siege_collateral_damage")
local dh = scn:system("DAGGERHEART")

-- Model collateral damage from siege weapons after an adversary falls.
scn:campaign{
  name = "Environment Helms Deep Siege Collateral Damage",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- A stray attack lands where the fight rages.
scn:start_session("Collateral Damage")
dh:gm_fear(1)

-- Stress-on-success/failure remains unresolved in the fixture DSL.
dh:gm_spend_fear(1):spotlight("Helms Deep Siege")
dh:group_reaction{
  targets = {"Frodo"},
  trait = "agility",
  difficulty = 17,
  outcome = "fear",
  damage = 15,
  damage_type = "physical",
  source = "collateral_damage"
}

scn:end_session()

return scn
