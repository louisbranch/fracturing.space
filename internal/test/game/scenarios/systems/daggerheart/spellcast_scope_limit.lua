local scn = Scenario.new("spellcast_scope_limit")
local dh = scn:system("DAGGERHEART")

-- Model the spellcast scope limit example.
scn:campaign{
  name = "Spellcast Scope Limit",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "spellcast"
}

scn:pc("Gandalf")

-- A spellcast roll should be disallowed if the effect isn't on a spell.
scn:start_session("Scope Limit")

-- Partial mapping: spellcast roll timeline is represented.
-- Missing DSL: explicit action/outcome rejection for out-of-scope effects.
dh:action_roll{ actor = "Gandalf", trait = "spellcast", difficulty = 0, outcome = "fear" }

scn:end_session()

return scn
