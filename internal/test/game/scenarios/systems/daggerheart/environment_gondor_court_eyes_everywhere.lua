local scn = Scenario.new("environment_gondor_court_eyes_everywhere")
local dh = scn:system("DAGGERHEART")

-- Model the fear-triggered eavesdropping reaction.
scn:campaign{
  name = "Environment Gondor Court Eyes Everywhere",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- A fear result risks being overheard.
scn:start_session("Eyes Everywhere")
dh:gm_fear(1)

-- Missing DSL: spend Fear to trigger witness and Instinct reaction to notice.
dh:gm_spend_fear(1):spotlight("Gondor Court")
dh:reaction_roll{ actor = "Frodo", trait = "instinct", difficulty = 20, outcome = "fear" }
dh:apply_reaction_outcome{}

scn:end_session()

return scn
