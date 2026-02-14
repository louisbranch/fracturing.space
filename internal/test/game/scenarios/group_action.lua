local scene = Scenario.new("group_action")

-- Assemble Frodo, Sam, and Gandalf for a group action.
scene:campaign{
  name = "Group Action",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "teamwork"
}

scene:pc("Frodo")
scene:pc("Sam")
scene:pc("Gandalf")

-- The party attempts a coordinated group action led by Frodo.
scene:start_session("Group Action")

-- Sam and Gandalf support Frodo's roll with their own traits.
-- Missing DSL: assert who succeeds/fails and any hope/stress shifts.
scene:group_action{
  leader = "Frodo",
  leader_trait = "instinct",
  difficulty = 10,
  supporters = {
    { name = "Sam", trait = "agility" },
    { name = "Gandalf", trait = "presence" }
  }
}

-- Close the session after the group action.
scene:end_session()

return scene
