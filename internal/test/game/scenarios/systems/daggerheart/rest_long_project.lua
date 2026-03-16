local scn = Scenario.new("rest_long_project")
local dh = scn:system("DAGGERHEART")

scn:campaign{
  name = "Long Rest Project",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "countdown"
}

scn:pc("Frodo", { hope = 1, stress = 2 })

scn:start_session("Campfire Project")

dh:countdown_create{
  name = "Map the Hidden Pass",
  kind = "progress",
  current = 0,
  max = 4,
  direction = "increase",
}

dh:rest{
  type = "long",
  expect_target = "Frodo",
  expect_rest_type = "long",
  expect_rest_interrupted = false,
  expect_gm_fear_delta = 0,
  expect_refresh_rest = true,
  expect_refresh_long_rest = true,
  expect_short_rests_after = 0,
  participants = {
    {
      character = "Frodo",
      downtime_moves = {
        { move = "work_on_project", countdown = "Map the Hidden Pass" },
      },
    },
  },
}

scn:end_session()

return scn
