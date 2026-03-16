local scn = Scenario.new("gm_fear_environment_feature")
local dh = scn:system("DAGGERHEART")

scn:campaign{
  name = "GM Fear Environment Feature",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "gm_fear"
}

scn:pc("Frodo")

scn:start_session("Environment Feature")
dh:gm_fear(2)
dh:gm_spend_fear(2):environment_feature("environment.crumbling-bridge", "feature.crumbling-bridge-falling-stones", {
  description = "Loose stones thunder down from the broken arch."
})
dh:expect_gm_fear(0)
scn:end_session()

return scn
