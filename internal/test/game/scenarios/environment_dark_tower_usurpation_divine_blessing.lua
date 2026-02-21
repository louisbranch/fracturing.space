local scene = Scenario.new("environment_dark_tower_usurpation_divine_blessing")

-- Model the critical success blessing to refresh abilities.
scene:campaign{
  name = "Environment Dark Tower Usurpation Divine Blessing",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- A critical success allows spending Hope to refresh an ability.
scene:start_session("Divine Blessing")

-- Missing DSL: spend 2 Hope to refresh a limited-use ability.
scene:action_roll{ actor = "Frodo", trait = "presence", difficulty = 20, outcome = "critical" }
scene:apply_roll_outcome{}

scene:end_session()

return scene
