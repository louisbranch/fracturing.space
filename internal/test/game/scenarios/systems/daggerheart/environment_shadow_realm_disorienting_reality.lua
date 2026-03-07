local scn = Scenario.new("environment_shadow_realm_disorienting_reality")
local dh = scn:system("DAGGERHEART")

-- Model the fear-triggered hope loss from disorienting reality.
scn:campaign{
  name = "Environment Shadow Realm Disorienting Reality",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- A fear result triggers a vision and Hope loss.
scn:start_session("Disorienting Reality")

-- Missing DSL: deduct Hope on Fear outcome and grant GM Fear if it was last Hope.
dh:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 20, outcome = "failure_fear" }
dh:apply_roll_outcome{}

scn:end_session()

return scn
