local scene = Scenario.new("death_reaction_dig_two_graves")

-- Recreate the on-death reaction that lashes out and steals Hope.
-- Clarification-gated fixture (P31): no generic on-death reaction pipeline exists yet.
scene:campaign{
  name = "Death Reaction Dig Two Graves",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "death"
}

scene:pc("Sam", { armor = 1 })
scene:adversary("Nazgul")

-- The knight falls but triggers a final reaction.
scene:start_session("Death Reaction")

-- Example: on-death attack deals 12 damage and steals 2 Hope.
-- Missing DSL: model the death-triggered reaction and Hope loss.
scene:adversary_attack{
  actor = "Nazgul",
  target = "Sam",
  difficulty = 0,
  damage_type = "physical"
}

-- Close the session after the death reaction.
scene:end_session()

return scene
