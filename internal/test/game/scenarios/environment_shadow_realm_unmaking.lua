local scene = Scenario.new("environment_shadow_realm_unmaking")

-- Model the Unmaking action and its direct magic damage.
scene:campaign{
  name = "Environment Shadow Realm Unmaking",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- The realm unravels a PC's essence.
scene:start_session("Unmaking")
scene:gm_fear(1)

-- Missing DSL: apply direct damage on failure and stress on success.
scene:gm_spend_fear(1):spotlight("Shadow Realm")
scene:reaction_roll{ actor = "Frodo", trait = "strength", difficulty = 20, outcome = "fear" }
scene:apply_reaction_outcome{}

scene:end_session()

return scene
