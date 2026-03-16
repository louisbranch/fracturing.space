local scn = Scenario.new("adversary_minion_spillover")
local dh = scn:system("DAGGERHEART")

scn:campaign{
  name = "Adversary Minion Spillover",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary_rules"
}

scn:pc("Frodo")
dh:adversary("Giant Rat A", { adversary_entry_id = "adversary.giant-rat" })
dh:adversary("Giant Rat B", { adversary_entry_id = "adversary.giant-rat" })
dh:adversary("Giant Rat C", { adversary_entry_id = "adversary.giant-rat" })

scn:start_session("Minion Spillover")

dh:combined_damage{
  target = "Giant Rat A",
  damage_type = "physical",
  sources = {
    { character = "Frodo", amount = 6 }
  },
  expect_adversary_deleted = true
}

dh:adversary_attack_roll{
  actor = "Giant Rat B",
  expect_error = { code = "NOT_FOUND" }
}

dh:adversary_attack_roll{
  actor = "Giant Rat C",
  expect_error = { code = "NOT_FOUND" }
}

scn:end_session()

return scn
