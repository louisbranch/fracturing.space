local scn = Scenario.new("gm_fear_adversary_feature")
local dh = scn:system("DAGGERHEART")

scn:campaign{
  name = "GM Fear Adversary Feature",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "gm_fear"
}

scn:pc("Frodo")
dh:adversary("Shadow Hound")

scn:start_session("Adversary Feature")
dh:gm_fear(1)
dh:gm_spend_fear(1):adversary_feature("Shadow Hound", "feature.shadow-hound-pounce", {
  description = "The Shadow Hound pounces from the rafters."
})
dh:expect_gm_fear(0)
scn:end_session()

return scn
