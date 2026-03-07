local scn = Scenario.new("group_finesse_sneak")
local dh = scn:system("DAGGERHEART")

-- Recreate the group finesse roll to sneak through the courtyard.
scn:campaign{
  name = "Group Finesse Sneak",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "teamwork"
}

scn:pc("Aragorn")
scn:pc("Frodo")
scn:pc("Gandalf")
scn:pc("Sam")

-- Aragorn leads a group finesse roll while allies contribute.
scn:start_session("Courtyard Infiltration")

-- Example: two allies succeed (+1 each), one fails (-1), net +1 for the leader.
-- Partial mapping: supporter outcomes encode the +1/+1/-1 contribution pattern.
dh:group_action{
  leader = "Aragorn",
  leader_trait = "finesse",
  difficulty = 12,
  outcome = "hope",
  supporters = {
    { name = "Frodo", trait = "finesse", outcome = "hope" },
    { name = "Gandalf", trait = "finesse", outcome = "hope" },
    { name = "Sam", trait = "finesse", outcome = "fear" }
  }
}

-- Close the session after the group roll.
scn:end_session()

return scn
