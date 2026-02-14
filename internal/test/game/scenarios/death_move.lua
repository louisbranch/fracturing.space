local scene = Scenario.new("death_move")

-- Frame Frodo at 0 HP to trigger a death move.
scene:campaign{
  name = "Death Move",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "death"
}

scene:pc("Frodo", { hp = 0, stress = 2 })

-- Frodo is down and must confront a death move.
scene:start_session("Death")

-- Avoid Death is chosen to stay in the fight.
scene:death_move{
  target = "Frodo",
  move = "avoid_death",
  seed = 42,
  expect_hp_delta = 0,
  expect_stress_delta = 0,
  expect_life_state = "unconscious",
  expect_scar_gained = false,
  expect_hope_die = 6,
  expect_hope_max = 6
}

-- Close the session after the death move resolves.
scene:end_session()

return scene
