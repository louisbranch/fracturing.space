local scn = Scenario.new("environment_pelennor_battle_raze")
local dh = scn:system("DAGGERHEART")

-- Model the raze-and-pillage escalation.
scn:campaign{
  name = "Environment Pelennor Battle Raze",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- The battle escalates with fire or abduction.
scn:start_session("Raze and Pillage")
dh:gm_fear(1)

-- Narrative branch selection and objective shifts remain unresolved.
dh:gm_spend_fear(1):spotlight("Battlefield")

scn:end_session()

return scn
