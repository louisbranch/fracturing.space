local scn = Scenario.new("environment_bree_outpost_shakedown")
local dh = scn:system("DAGGERHEART")

-- Model the crime boss shakedown that the PCs can intervene in.
scn:campaign{
  name = "Environment Bree Outpost Shakedown",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
dh:adversary("Orc Boss")

-- The party witnesses intimidation at a general goods store.
scn:start_session("Shakedown")
dh:gm_fear(1)

-- Example: the environment action introduces a threat without a roll.
-- Partial mapping: pressure setup and intervention branch are explicit.
-- Missing DSL: richer social leverage state (favors/debts/escalation posture).
dh:gm_spend_fear(1):spotlight("Orc Boss", { description = "storefront_extortion_demand" })
dh:scene_countdown_create{ name = "Shakedown Escalation", kind = "consequence", current = 0, max = 4, direction = "increase" }
dh:action_roll{ actor = "Frodo", trait = "presence", difficulty = 13, outcome = "failure_fear" }
dh:apply_roll_outcome{
  on_failure_fear = {
    {kind = "scene_countdown_update", name = "Shakedown Escalation", delta = 1, reason = "failed_intervention"},
  },
  on_success = {
    {kind = "set_spotlight", target = "Frodo"},
  },
}

scn:end_session()

return scn
