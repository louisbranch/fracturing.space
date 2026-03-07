local scn = Scenario.new("airship_group_roll")
local dh = scn:system("DAGGERHEART")

-- Capture the airship crisis that calls for a group roll.
scn:campaign{
  name = "Airship Group Roll",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "teamwork"
}

scn:pc("Sam")
scn:pc("Frodo")
scn:pc("Gandalf")

-- The spellrider breaks the enchantment keeping the airship aloft.
scn:start_session("Airship Crisis")

-- Example: the GM calls for a group roll to keep the airship flying.
-- Partial mapping: explicit leader/supporter outcomes drive the shared branch.
dh:group_action{
  leader = "Sam",
  leader_trait = "presence",
  difficulty = 14,
  outcome = "hope",
  supporters = {
    { name = "Frodo", trait = "agility", outcome = "hope" },
    { name = "Gandalf", trait = "instinct", outcome = "fear" }
  }
}

-- Close the session after the group roll.
scn:end_session()

return scn
