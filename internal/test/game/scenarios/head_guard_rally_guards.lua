local scene = Scenario.new("head_guard_rally_guards")

-- Model the leader action that spotlights allies for extra pressure.
scene:campaign{
  name = "Gondor Captain Rally Guards",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "gm_fear"
}

scene:pc("Frodo")
scene:adversary("Gondor Captain")
scene:adversary("Gondor Archers")

-- The GM spends Fear to rally the guards into coordinated action.
scene:start_session("Rally Guards")
scene:gm_fear(2)

-- Example: spend 2 Fear to spotlight the head guard and allies.
-- Partial mapping: exact fear spend plus bounded spotlight fanout are explicit.
-- Missing DSL: automatic ally-count expansion from adversary topology.
scene:gm_spend_fear(2):spotlight("Gondor Captain")

scene:end_session()

return scene
