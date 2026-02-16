local scene = Scenario.new("advantage_disguise_roll")

-- Highlight advantage on a roll due to a disguise.
scene:campaign{
  name = "Advantage Disguise Roll",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "advantage"
}

scene:pc("Aragorn")

-- Aragorn slips past the head guard using a stolen uniform.
scene:start_session("Sneak Past Guard")

-- Example: Difficulty 15 Presence roll with advantage from the disguise.
scene:action_roll{
  actor = "Aragorn",
  trait = "presence",
  difficulty = 15,
  advantage = 1,
  outcome = "hope"
}

-- Close the session after the stealth attempt.
scene:end_session()

return scene
