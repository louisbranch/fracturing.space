local scene = Scenario.new("head_guard_rally_guards")
local dh = scene:system("DAGGERHEART")

-- Model the leader action that spotlights allies for extra pressure.
scene:campaign{
  name = "Gondor Captain Rally Guards",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "gm_fear"
}

scene:pc("Frodo")
dh:adversary("Gondor Captain")
dh:adversary("Gondor Archers")

-- The GM spends Fear to rally the guards into coordinated action.
scene:start_session("Rally Guards")
dh:gm_fear(2)

-- Example: spend 2 Fear to spotlight the head guard and allies.
-- Partial mapping: exact fear spend plus bounded spotlight fanout are explicit.
-- Missing DSL: automatic ally-count expansion from adversary topology.
dh:gm_spend_fear(2):spotlight("Gondor Captain")

scene:end_session()

return scene
