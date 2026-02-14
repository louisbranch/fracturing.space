local scene = Scenario.new("rest_and_downtime")

-- Frame Frodo between encounters for rest and downtime.
scene:campaign{
  name = "Rest and Downtime",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "rest"
}

scene:pc("Frodo", { hp = 3, stress = 3, armor = 1 })

-- Frodo pauses to recover between encounters.
scene:start_session("Rest")

-- A short rest to regain footing, followed by a Prepare downtime move.
-- Missing DSL: assert specific recovery outcomes (HP/stress/hope).
scene:rest{ type = "short", party_size = 1 }
scene:downtime_move{ target = "Frodo", move = "prepare", prepare_with_group = false }

-- Close the session once the rest is done.
scene:end_session()

return scene
