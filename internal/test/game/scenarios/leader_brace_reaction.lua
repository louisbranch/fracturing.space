local scene = Scenario.new("leader_brace_reaction")

-- Capture the Mirkwood Warden Brace reaction reducing HP loss.
scene:campaign{
  name = "Leader Brace Reaction",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary"
}

scene:pc("Frodo")
scene:adversary("Mirkwood Warden")

-- The Mirkwood Warden marks Stress to reduce HP marked.
scene:start_session("Brace")

-- Example: when the Mirkwood Warden marks HP, they can mark Stress to mark 1 fewer.
-- Partial mapping: explicit stress spend and reaction update are represented.
-- Missing DSL: automatic "mark 1 fewer HP" mitigation tied to qualifying HP-loss windows.
scene:attack{
  actor = "Frodo",
  target = "Mirkwood Warden",
  trait = "instinct",
  difficulty = 0,
  outcome = "hope",
  damage_type = "physical"
}
scene:adversary_update{
  target = "Mirkwood Warden",
  stress_delta = 1,
  notes = "brace_reaction"
}

scene:end_session()

return scene
