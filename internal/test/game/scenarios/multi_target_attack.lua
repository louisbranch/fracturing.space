local scene = Scenario.new("multi_target_attack")

-- Introduce two targets for a sweeping strike.
scene:campaign{
  name = "Multi Target Attack",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "battle"
}

scene:pc("Frodo")
scene:adversary("Nazgul")
scene:adversary("Golum")

-- Frodo swings wide to catch multiple enemies in one blow.
scene:start_session("Multi Target")

-- A single critical roll applies to both targets.
-- Missing DSL: assert damage outcomes for each target individually.
scene:multi_attack{
  actor = "Frodo",
  targets = { "Nazgul", "Golum" },
  trait = "instinct",
  difficulty = 0,
  outcome = "critical",
  damage_type = "physical",
  damage_dice = { { sides = 6, count = 1 } }
}

-- Close the session after the sweeping strike.
scene:end_session()

return scene
