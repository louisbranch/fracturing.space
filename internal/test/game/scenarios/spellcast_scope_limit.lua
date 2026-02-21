local scene = Scenario.new("spellcast_scope_limit")

-- Model the spellcast scope limit example.
scene:campaign{
  name = "Spellcast Scope Limit",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "spellcast"
}

scene:pc("Gandalf")

-- A spellcast roll should be disallowed if the effect isn't on a spell.
scene:start_session("Scope Limit")

-- Partial mapping: spellcast roll timeline is represented.
-- Missing DSL: explicit action/outcome rejection for out-of-scope effects.
scene:action_roll{ actor = "Gandalf", trait = "spellcast", difficulty = 0, outcome = "fear" }

scene:end_session()

return scene
