local scene = Scenario.new("environment_bree_outpost_shakedown")

-- Model the crime boss shakedown that the PCs can intervene in.
scene:campaign{
  name = "Environment Bree Outpost Shakedown",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
scene:adversary("Orc Boss")

-- The party witnesses intimidation at a general goods store.
scene:start_session("Shakedown")
scene:gm_fear(1)

-- Example: the environment action introduces a threat without a roll.
-- Partial mapping: pressure setup and intervention branch are explicit.
-- Missing DSL: richer social leverage state (favors/debts/escalation posture).
scene:gm_spend_fear(1):spotlight("Orc Boss", { description = "storefront_extortion_demand" })
scene:countdown_create{ name = "Shakedown Escalation", kind = "consequence", current = 0, max = 4, direction = "increase" }
scene:action_roll{ actor = "Frodo", trait = "presence", difficulty = 13, outcome = "failure_fear" }
scene:apply_roll_outcome{
  on_failure_fear = {
    {kind = "countdown_update", name = "Shakedown Escalation", delta = 1, reason = "failed_intervention"},
  },
  on_success = {
    {kind = "set_spotlight", target = "Frodo"},
  },
}

scene:end_session()

return scene
