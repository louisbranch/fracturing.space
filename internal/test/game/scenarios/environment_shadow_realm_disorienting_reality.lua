local scene = Scenario.new("environment_shadow_realm_disorienting_reality")

-- Model the fear-triggered hope loss from disorienting reality.
scene:campaign{
  name = "Environment Shadow Realm Disorienting Reality",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- A fear result triggers a vision and Hope loss.
scene:start_session("Disorienting Reality")

-- Missing DSL: deduct Hope on Fear outcome and grant GM Fear if it was last Hope.
scene:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 20, outcome = "failure_fear" }
scene:apply_roll_outcome{}

scene:end_session()

return scene
