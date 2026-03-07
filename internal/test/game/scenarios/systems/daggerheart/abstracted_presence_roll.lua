local scn = Scenario.new("abstracted_presence_roll")
local dh = scn:system("DAGGERHEART")

-- Reflect the example of abstracting a social scene into a roll.
scn:campaign{
  name = "Abstracted Presence Roll",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "social"
}

-- Add the party member taking the roll.
scn:pc("Frodo")

-- Frodo works the crowd for leads, resolved by a single roll.
scn:start_session("Crowd Work")

-- Example: skip the full scene, call for a Presence roll.
dh:action_roll{ actor = "Frodo", trait = "presence", difficulty = 12, outcome = "hope" }

-- Close the session after the abstracted roll.
scn:end_session()

return scn
