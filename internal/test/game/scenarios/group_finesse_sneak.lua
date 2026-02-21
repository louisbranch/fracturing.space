local scene = Scenario.new("group_finesse_sneak")

-- Recreate the group finesse roll to sneak through the courtyard.
scene:campaign{
  name = "Group Finesse Sneak",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "teamwork"
}

scene:pc("Aragorn")
scene:pc("Frodo")
scene:pc("Gandalf")
scene:pc("Sam")

-- Aragorn leads a group finesse roll while allies contribute.
scene:start_session("Courtyard Infiltration")

-- Example: two allies succeed (+1 each), one fails (-1), net +1 for the leader.
-- Partial mapping: supporter outcomes encode the +1/+1/-1 contribution pattern.
scene:group_action{
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
scene:end_session()

return scene
