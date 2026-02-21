local scene = Scenario.new("spellcast_hope_cost")

-- Capture the spellcast roll that costs Hope to cast.
scene:campaign{
  name = "Spellcast Hope Cost",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "spellcast"
}

scene:pc("Gandalf", { hope = 2 })

-- Gandalf casts a warding door with a Hope cost.
scene:start_session("Arcane Door")
scene:gm_fear(0)

-- Build enough Hope for the explicit spend step.
scene:action_roll{ actor = "Gandalf", trait = "presence", difficulty = 10, outcome = "success_hope" }
scene:apply_roll_outcome{}

-- Example: Spellcast roll Difficulty 13, success with Fear after spending Hope.
-- Partial mapping: explicit Hope spend on cast and GM Fear gain are represented.
-- Missing DSL: automatic Fear gain coupling to specific spellcast outcome branches.
scene:action_roll{
  actor = "Gandalf",
  trait = "spellcast",
  difficulty = 13,
  outcome = "fear",
  expect_hope_delta = -1,
  modifiers = {
    Modifiers.hope("hope_feature")
  }
}
scene:gm_fear(1)

-- Close the session after the spellcast.
scene:end_session()

return scene
