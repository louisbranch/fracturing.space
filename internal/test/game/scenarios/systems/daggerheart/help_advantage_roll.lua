local scn = Scenario.new("help_advantage_roll")
local dh = scn:system("DAGGERHEART")

-- Mirror the help action that grants an advantage die.
scn:campaign{
  name = "Help Advantage Roll",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "help"
}

scn:pc("Aragorn")
scn:pc("Gandalf")

-- Gandalf spends Hope to help Aragorn's Instinct roll.
scn:start_session("Help an Ally")

-- Example: help adds an advantage die and a +3 bonus from assistance.
dh:action_roll{
  actor = "Aragorn",
  trait = "instinct",
  difficulty = 10,
  outcome = "fear",
  advantage = 1,
  hope_spends = {
    Modifiers.hope("help"),
  },
  modifiers = {
    Modifiers.mod("training", 3),
  }
}

-- Close the session after the assisted roll.
scn:end_session()

return scn
