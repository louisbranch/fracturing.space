local scene = Scenario.new("environment_shadow_realm_unmaking")
local dh = scene:system("DAGGERHEART")

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
dh:gm_fear(1)

-- Missing DSL: apply direct damage on failure and stress on success.
dh:gm_spend_fear(1):spotlight("Shadow Realm")
dh:reaction_roll{ actor = "Frodo", trait = "strength", difficulty = 20, outcome = "fear" }
dh:apply_reaction_outcome{}

scene:end_session()

return scene
