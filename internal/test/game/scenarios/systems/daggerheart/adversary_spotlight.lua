local scn = Scenario.new("adversary_spotlight")
local dh = scn:system("DAGGERHEART")

-- Frame a battlefield with Frodo and a looming Nazgul.
scn:campaign{
  name = "Frodo Scene",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "spotlight"
}

-- Frodo clashes with Nazgul while the GM keeps an eye on the battlefield.
-- Nazgul is present as a looming threat the GM can spotlight.
scn:pc("Frodo", { armor = 1 })
dh:adversary("Nazgul")

-- The fight begins with the GM holding a pool of fear.
scn:start_session("Battlefield")
dh:gm_fear(6)

-- Frodo strikes Nazgul, but the roll lands on Fear so the GM takes control.
dh:attack{
  actor = "Frodo",
  target = "Nazgul",
  trait = "instinct",
  difficulty = 0,
  outcome = "fear",
  expect_gm_fear_delta = 1,
  expect_spotlight = "gm",
  expect_requires_complication = true,
  damage_type = "physical"
}

-- The GM presses the advantage, then Frodo breaks free.
dh:apply_condition{ target = "Frodo", add = { "VULNERABLE" } }
dh:gm_spend_fear(1):move("custom", {
  description = "Nazgul presses the attack.",
  expect_gm_fear_delta = -1,
  expect_gm_move = "custom",
  expect_gm_fear_spent = 1
})
dh:apply_condition{ target = "Frodo", remove = { "VULNERABLE" }, source = "break_free" }

-- The spotlight shifts to Nazgul, who lashes out at Frodo.
dh:adversary_attack{
  actor = "Nazgul",
  target = "Frodo",
  difficulty = 0,
  expect_hope_delta = 0,
  expect_stress_delta = 0,
  expect_hp_delta = -1,
  expect_armor_delta = -1,
  expect_damage_total = 4,
  expect_damage_severity = "minor",
  expect_damage_marks = 1,
  expect_armor_spent = 1,
  expect_damage_mitigated = true,
  expect_damage_critical = false,
  damage_type = "physical"
}

-- The GM spends fear to keep the pressure on Nazgul.
dh:gm_spend_fear(1):move("custom", {
  description = "Nazgul keeps the pressure on.",
  expect_gm_fear_delta = -1,
  expect_gm_move = "custom",
  expect_gm_fear_spent = 1
})

-- Spotlight returns to the players after the GM move resolves.
scn:set_spotlight{ target = "Frodo", expect_spotlight = "Frodo" }

return scn
