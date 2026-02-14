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
-- Missing DSL: assert the resulting recovery or consequence.
scene:death_move{ target = "Frodo", move = "avoid_death" }

-- Close the session after the death move resolves.
scene:end_session()

return scene
