local scene = Scenario.new("leader_into_bramble")
local dh = scene:system("DAGGERHEART")

-- Model the Mirkwood Warden's Into the Bramble fear action.
scene:campaign{
  name = "Leader Into the Bramble",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary"
}

scene:pc("Frodo")
dh:adversary("Mirkwood Warden")
dh:adversary("Mirkwood Archer")

-- The leader spends Fear to reposition allies and hide them.
scene:start_session("Bramble Ambush")
dh:gm_fear(1)

-- Example: spotlight up to 1d4 allies and grant Hidden.
-- Partial mapping: fear spend + Hidden application are represented.
-- Missing DSL: ally reposition metadata and explicit outcome-journal linkage.
dh:gm_spend_fear(1):spotlight("Mirkwood Warden")
dh:apply_condition{ target = "Mirkwood Archer", add = { "HIDDEN" }, source = "into_bramble" }

scene:end_session()

return scene
