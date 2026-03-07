local scene = Scenario.new("environment_pelennor_battle_raze")
local dh = scene:system("DAGGERHEART")

-- Model the raze-and-pillage escalation.
scene:campaign{
  name = "Environment Pelennor Battle Raze",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- The battle escalates with fire or abduction.
scene:start_session("Raze and Pillage")
dh:gm_fear(1)

-- Narrative branch selection and objective shifts remain unresolved.
dh:gm_spend_fear(1):spotlight("Battlefield")

scene:end_session()

return scene
