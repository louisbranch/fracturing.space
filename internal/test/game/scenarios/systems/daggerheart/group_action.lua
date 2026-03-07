local scn = Scenario.new("group_action")
local dh = scn:system("DAGGERHEART")

-- Assemble Frodo, Sam, and Gandalf for a group action.
scn:campaign{
  name = "Group Action",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "teamwork"
}

scn:pc("Frodo")
scn:pc("Sam")
scn:pc("Gandalf")

-- The party attempts a coordinated group action led by Frodo.
scn:start_session("Group Action")

-- Sam and Gandalf support Frodo's roll with their own traits.
dh:group_action{
  leader = "Frodo",
  leader_trait = "instinct",
  difficulty = 10,
  expect_hope_delta = 0,
  expect_stress_delta = 0,
  expect_gm_fear_delta = 1,
  supporters = {
    { name = "Sam", trait = "agility" },
    { name = "Gandalf", trait = "presence" }
  }
}

-- Close the session after the group action.
scn:end_session()

return scn
