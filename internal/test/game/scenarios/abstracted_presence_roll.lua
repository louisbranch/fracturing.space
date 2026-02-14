local scene = Scenario.new("abstracted_presence_roll")

-- Reflect the example of abstracting a social scene into a roll.
scene:campaign{
  name = "Abstracted Presence Roll",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "social"
}

-- Add the party member taking the roll.
scene:pc("Frodo")

-- Frodo works the crowd for leads, resolved by a single roll.
scene:start_session("Crowd Work")

-- Example: skip the full scene, call for a Presence roll.
scene:action_roll{ actor = "Frodo", trait = "presence", difficulty = 12, outcome = "hope" }

-- Close the session after the abstracted roll.
scene:end_session()

return scene
