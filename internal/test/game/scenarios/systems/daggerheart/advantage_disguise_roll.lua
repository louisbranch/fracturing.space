local scn = Scenario.new("advantage_disguise_roll")
local dh = scn:system("DAGGERHEART")

-- Highlight advantage on a roll due to a disguise.
scn:campaign{
  name = "Advantage Disguise Roll",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "advantage"
}

scn:pc("Aragorn")

-- Aragorn slips past the head guard using a stolen uniform.
scn:start_session("Sneak Past Guard")

-- Example: Difficulty 15 Presence roll with advantage from the disguise.
dh:action_roll{
  actor = "Aragorn",
  trait = "presence",
  difficulty = 15,
  advantage = 1,
  outcome = "hope"
}

-- Close the session after the stealth attempt.
scn:end_session()

return scn
