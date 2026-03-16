local scn = Scenario.new("rest_and_downtime")
local dh = scn:system("DAGGERHEART")

scn:campaign{
  name = "Rest and Downtime",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "rest"
}

scn:pc("Frodo", { hp = 5, hope = 1, stress = 1, armor = 0 })
scn:pc("Sam", { hp = 2, hope = 1, stress = 2, armor = 0 })

scn:start_session("Rest")

dh:rest{
  type = "short",
  seed = 42,
  expect_target = "Frodo",
  expect_hope_delta = 2,
  expect_rest_type = "short",
  expect_rest_interrupted = false,
  expect_gm_fear_delta = 2,
  expect_refresh_rest = true,
  expect_refresh_long_rest = false,
  expect_short_rests_after = 1,
  participants = {
    {
      character = "Frodo",
      downtime_moves = {
        { move = "tend_to_wounds", target = "Sam", seed = 11 },
        { move = "prepare", group_id = "watch" },
      },
    },
    {
      character = "Sam",
      downtime_moves = {
        { move = "prepare", group_id = "watch" },
      },
    },
  },
}

scn:end_session()

return scn
