local scn = Scenario.new("environment_dark_tower_usurpation_divine_blessing")
local dh = scn:system("DAGGERHEART")

-- Model the critical success blessing to refresh abilities.
scn:campaign{
  name = "Environment Dark Tower Usurpation Divine Blessing",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- A critical success allows spending Hope to refresh an ability.
scn:start_session("Divine Blessing")

-- Missing DSL: spend 2 Hope to refresh a limited-use ability.
dh:action_roll{ actor = "Frodo", trait = "presence", difficulty = 20, outcome = "critical" }
dh:apply_roll_outcome{}

scn:end_session()

return scn
