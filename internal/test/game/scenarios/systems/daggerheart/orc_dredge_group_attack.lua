local scn = Scenario.new("orc_dredge_group_attack")
local dh = scn:system("DAGGERHEART")

-- Model the orc raiders making a group attack.
scn:campaign{
  name = "Orc Raider Group Attack",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "battle"
}

scn:pc("Aragorn")
dh:adversary("Orc Raiders")

-- The GM spends fear to have the raiders strike as a group.
scn:start_session("Group Attack")
dh:gm_fear(1)

-- Example: single group attack roll against Aragorn's Evasion.
-- Missing DSL: represent group attack roll and shared damage.
dh:gm_spend_fear(1):spotlight("Orc Raiders")
dh:adversary_attack{
  actor = "Orc Raiders",
  target = "Aragorn",
  difficulty = 0,
  damage_type = "physical"
}

-- Close the session after the group strike.
scn:end_session()

return scn
