local scene = Scenario.new("adversary_spotlight")
local dh = scene:system("DAGGERHEART")

-- Frame a battlefield with Frodo and a looming Nazgul.
scene:campaign{
  name = "Frodo Scene",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "spotlight"
}

-- Frodo clashes with Nazgul while the GM keeps an eye on the battlefield.
-- Nazgul is present as a looming threat the GM can spotlight.
scene:pc("Frodo", { armor = 1 })
dh:adversary("Nazgul")

-- The fight begins with the GM holding a pool of fear.
scene:start_session("Battlefield")
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

-- The GM briefly spotlights Frodo: vulnerable, then breaking free.
dh:apply_condition{ target = "Frodo", add = { "VULNERABLE" } }
dh:gm_spend_fear(1):spotlight("Nazgul", { expect_gm_fear_delta = -1, expect_gm_move = "spotlight", expect_gm_fear_spent = 1 })
dh:apply_condition{ target = "Frodo", remove = { "VULNERABLE" }, source = "break_free" }

-- The spotlight shifts to Nazgul, who lashes out at Frodo.
dh:adversary_attack{
  actor = "Nazgul",
  target = "Frodo",
  difficulty = 0,
  expect_hope_delta = 0,
  expect_stress_delta = 0,
  expect_hp_delta = -2,
  expect_armor_delta = -1,
  expect_damage_total = 4,
  expect_damage_severity = "major",
  expect_damage_marks = 2,
  expect_armor_spent = 1,
  expect_damage_mitigated = true,
  expect_damage_critical = false,
  damage_type = "physical"
}

-- The GM spends fear to keep the spotlight on Nazgul.
dh:gm_spend_fear(1):spotlight("Nazgul", { expect_gm_fear_delta = -1, expect_gm_move = "spotlight", expect_gm_fear_spent = 1 })

-- Spotlight returns to the players after the GM move resolves.
scene:set_spotlight{ target = "Frodo", expect_spotlight = "Frodo" }

return scene
