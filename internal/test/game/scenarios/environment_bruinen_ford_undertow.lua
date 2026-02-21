local scene = Scenario.new("environment_bruinen_ford_undertow")

-- Model the Bruinen Ford undertow action and its consequences.
scene:campaign{
  name = "Environment Bruinen Ford Undertow",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- The river lashes out during a dangerous crossing.
scene:start_session("Bruinen Ford")
scene:gm_fear(1)

-- Example: spend Fear, Agility reaction, damage + movement + Vulnerable on failure.
-- River movement and conditional stress on success remain unresolved.
scene:gm_spend_fear(1):spotlight("Bruinen Ford")
scene:reaction_roll{ actor = "Frodo", trait = "agility", difficulty = 10, outcome = "failure_fear" }
scene:apply_reaction_outcome{
  on_failure = {
    {kind = "apply_condition", target = "Frodo", add = {"VULNERABLE"}, source = "undertow"},
  },
}

scene:end_session()

return scene
