local scn = Scenario.new("gm_fear_adversary_experience")
local dh = scn:system("DAGGERHEART")

scn:campaign{
  name = "GM Fear Adversary Experience",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "gm_fear"
}

scn:pc("Frodo")
dh:adversary("Shadow Hound")

scn:start_session("Adversary Experience")
dh:gm_fear(1)
dh:gm_spend_fear(1):adversary_experience("Shadow Hound", "Pack Hunter", {
  description = "The pack circles together before the strike."
})
dh:expect_gm_fear(0)
dh:adversary_attack{
  actor = "Shadow Hound",
  target = "Frodo",
  difficulty = 0,
  damage_type = "physical",
  seed = 33,
}
scn:end_session()

return scn
