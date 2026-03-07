local scn = Scenario.new("ranged_take_cover")
local dh = scn:system("DAGGERHEART")

-- Model the Ranger of the North's Take Cover reaction.
scn:campaign{
  name = "Ranged Take Cover",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary"
}

scn:pc("Frodo")
dh:adversary("Ranger of the North")

-- The hunter marks Stress to impose disadvantage and reduce damage tier.
scn:start_session("Take Cover")

-- Missing DSL: apply disadvantage to the attack and reduce damage severity.
dh:attack{ actor = "Frodo", target = "Ranger of the North", trait = "instinct", difficulty = 0, outcome = "hope", damage_type = "physical" }

scn:end_session()

return scn
