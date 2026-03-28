local scn = Scenario.new("hope_spend_then_hope_outcome")
local dh = scn:system("DAGGERHEART")

scn:campaign{
  name = "Hope Spend Then Hope Outcome",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "hope"
}

scn:pc("Frodo", { hope = 2 })
dh:adversary("Orc Raider")

scn:start_session("Spend Then Gain")

dh:attack{
  actor = "Frodo",
  target = "Orc Raider",
  trait = "presence",
  difficulty = 0,
  outcome = "hope",
  hope_spends = {
    Modifiers.hope("experience"),
  },
  modifiers = {
    Modifiers.mod("training", 3),
  },
  expect_outcome = "hope",
  expect_hope_delta = 0,
  damage_type = "physical"
}

scn:end_session()

return scn
