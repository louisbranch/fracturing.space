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
-- TODO: Update rest deltas if rest recovery rules change.
scene:rest{
  type = "short",
  seed = 42,
  party_size = 1,
  expect_target = "Frodo",
  expect_hp_delta = 0,
  expect_stress_delta = 0,
  expect_armor_delta = 0,
  expect_hope_delta = 0,
  expect_rest_type = "short",
  expect_rest_interrupted = false,
  expect_gm_fear_delta = 1,
  expect_refresh_rest = true,
  expect_refresh_long_rest = false,
  expect_short_rests_after = 1
}
scene:downtime_move{
  target = "Frodo",
  move = "prepare",
  prepare_with_group = false,
  expect_hope_delta = 1,
  expect_stress_delta = 0,
  expect_armor_delta = 0,
  expect_downtime_move = "prepare"
}

-- Close the session once the rest is done.
scene:end_session()

return scene
