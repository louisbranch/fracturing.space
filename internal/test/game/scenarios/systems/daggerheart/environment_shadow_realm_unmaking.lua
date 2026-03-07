local scn = Scenario.new("environment_shadow_realm_unmaking")
local dh = scn:system("DAGGERHEART")

-- Model the Unmaking action and its direct magic damage.
scn:campaign{
  name = "Environment Shadow Realm Unmaking",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- The realm unravels a PC's essence.
scn:start_session("Unmaking")
dh:gm_fear(1)

-- Missing DSL: apply direct damage on failure and stress on success.
dh:gm_spend_fear(1):spotlight("Shadow Realm")
dh:reaction_roll{ actor = "Frodo", trait = "strength", difficulty = 20, outcome = "fear" }
dh:apply_reaction_outcome{}

scn:end_session()

return scn
