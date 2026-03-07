local scn = Scenario.new("head_guard_rally_guards")
local dh = scn:system("DAGGERHEART")

-- Model the leader action that spotlights allies for extra pressure.
scn:campaign{
  name = "Gondor Captain Rally Guards",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "gm_fear"
}

scn:pc("Frodo")
dh:adversary("Gondor Captain")
dh:adversary("Gondor Archers")

-- The GM spends Fear to rally the guards into coordinated action.
scn:start_session("Rally Guards")
dh:gm_fear(2)

-- Example: spend 2 Fear to spotlight the head guard and allies.
-- Partial mapping: exact fear spend plus bounded spotlight fanout are explicit.
-- Missing DSL: automatic ally-count expansion from adversary topology.
dh:gm_spend_fear(2):spotlight("Gondor Captain")

scn:end_session()

return scn
