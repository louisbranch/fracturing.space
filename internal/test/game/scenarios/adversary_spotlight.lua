local scene = Scenario.new("adversary_spotlight")

-- Frame a battlefield with Frodo and a looming Nazgul.
scene:campaign{
  name = "Shepherd Scene",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "spotlight"
}

-- Frodo clashes with Nazgul while the GM keeps an eye on the battlefield.
-- Nazgul is present as a looming threat the GM can spotlight.
scene:pc("Frodo", { armor = 1 })
scene:adversary("Nazgul")

-- The fight begins with the GM holding a pool of fear.
scene:start_session("Battlefield")
scene:gm_fear(6)

-- Frodo strikes Nazgul, but the roll lands on Fear so the GM takes control.
-- Missing DSL: assert fear increases by 1 on a Fear outcome.
scene:attack{
  actor = "Frodo",
  target = "Nazgul",
  trait = "instinct",
  difficulty = 0,
  outcome = "fear",
  damage_type = "physical"
}

-- The GM briefly spotlights Frodo: vulnerable, then breaking free.
scene:apply_condition{ target = "Frodo", add = { "VULNERABLE" } }
scene:gm_spend_fear(1):spotlight("Nazgul")
scene:apply_condition{ target = "Frodo", remove = { "VULNERABLE" }, source = "break_free" }

-- The spotlight shifts to Nazgul, who lashes out at Frodo.
-- Missing DSL: specify the attack outcome and damage/armor consequences.
scene:adversary_attack{
  actor = "Nazgul",
  target = "Frodo",
  difficulty = 0,
  damage_type = "physical"
}

-- The GM spends fear to keep the spotlight on Nazgul.
-- Missing DSL: explicitly return spotlight to the players.
scene:gm_spend_fear(1):spotlight("Nazgul")

return scene
