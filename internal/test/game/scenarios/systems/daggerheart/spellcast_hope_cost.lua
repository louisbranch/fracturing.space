local scn = Scenario.new("spellcast_hope_cost")
local dh = scn:system("DAGGERHEART")

-- Capture the spellcast roll that costs Hope to cast.
scn:campaign{
  name = "Spellcast Hope Cost",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "spellcast"
}

scn:pc("Gandalf", { hope = 2 })

-- Gandalf casts a warding door with a Hope cost.
scn:start_session("Arcane Door")
dh:gm_fear(0)

-- Build enough Hope for the explicit spend step.
dh:action_roll{ actor = "Gandalf", trait = "presence", difficulty = 10, outcome = "success_hope" }
dh:apply_roll_outcome{}

-- Example: Spellcast roll Difficulty 13, success with Fear after spending Hope.
-- Partial mapping: explicit Hope spend on cast and GM Fear gain are represented.
-- Missing DSL: automatic Fear gain coupling to specific spellcast outcome branches.
dh:action_roll{
  actor = "Gandalf",
  trait = "spellcast",
  difficulty = 13,
  outcome = "fear",
  expect_hope_delta = -3,
  modifiers = {
    Modifiers.hope("hope_feature")
  }
}
dh:gm_fear(1)

-- Close the session after the spellcast.
scn:end_session()

return scn
