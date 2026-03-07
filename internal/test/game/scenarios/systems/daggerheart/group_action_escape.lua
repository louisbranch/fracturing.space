local scn = Scenario.new("group_action_escape")
local dh = scn:system("DAGGERHEART")

-- Recreate the collapsing stronghold group action example.
scn:campaign{
  name = "Group Action Escape",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "teamwork"
}

scn:pc("Sam")
scn:pc("Frodo")
scn:pc("Gandalf")

-- The party bolts for the exit as the stronghold comes down.
scn:start_session("Escape")

-- Sam leads the group action to remember the way out.
-- Partial mapping: explicit leader/supporter outcomes are represented.
-- Missing DSL: direct assertions for supporter-only roll effects.
dh:group_action{
  leader = "Sam",
  leader_trait = "instinct",
  difficulty = 12,
  outcome = "fear",
  supporters = {
    { name = "Frodo", trait = "presence", outcome = "hope" },
    { name = "Gandalf", trait = "agility", outcome = "fear" }
  }
}

-- Close the session after the escape attempt.
scn:end_session()

return scn
