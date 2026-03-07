local scn = Scenario.new("fear_spotlight_armor_mitigation")
local dh = scn:system("DAGGERHEART")

-- Recreate a fear-triggered spotlight shift with armor mitigation.
scn:campaign{
  name = "Fear Spotlight Armor Mitigation",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "gm_fear"
}

scn:pc("Frodo", { armor = 1 })
dh:adversary("Uruk-hai Brute")

-- Frodo strikes, the roll lands on Fear, and the GM takes over.
scn:start_session("Spotlight Shift")
dh:gm_fear(6)

dh:attack{
  actor = "Frodo",
  target = "Uruk-hai Brute",
  trait = "instinct",
  difficulty = 0,
  outcome = "fear",
  damage_type = "physical"
}

-- The GM spotlights the adversary breaking free from Vulnerable.
dh:apply_condition{ target = "Uruk-hai Brute", add = { "VULNERABLE" } }
dh:gm_spend_fear(1):spotlight("Uruk-hai Brute")
dh:apply_condition{ target = "Uruk-hai Brute", remove = { "VULNERABLE" }, source = "break_free" }

-- The adversary counterattacks for 9 damage; armor reduces Major to Minor.
-- Missing DSL: set the adversary hit, damage total, and armor slot spend.
dh:adversary_attack{
  actor = "Uruk-hai Brute",
  target = "Frodo",
  difficulty = 0,
  damage_type = "physical"
}

-- Close the session after the spotlight exchange.
scn:end_session()

return scn
