local scn = Scenario.new("death_move_last_hope_slot")
local dh = scn:system("DAGGERHEART")

scn:campaign{
  name = "Death Move Last Hope Slot",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "death"
}

scn:pc("Frodo", { level = 10, hp = 0, hope = 1, hope_max = 1, stress = 2 })

scn:start_session("Death")

dh:death_move{
  target = "Frodo",
  move = "avoid_death",
  seed = 1,
  expect_hp_delta = 0,
  expect_hope_delta = -1,
  expect_stress_delta = 0,
  expect_life_state = "dead",
  expect_scar_gained = true,
  expect_hope_die = 8,
  expect_hope_max = 0
}

scn:end_session()

return scn
