local scene = Scenario.new("environment_dark_tower_usurpation_final_preparations")

-- Capture the final preparations long-term countdown and fear cap.
scene:campaign{
  name = "Environment Dark Tower Usurpation Final Preparations",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
scene:adversary("Saruman")

-- The usurper's ritual begins when the environment takes spotlight.
scene:start_session("Final Preparations")

-- Fear-cap override (15) remains unresolved in this fixture.
scene:countdown_create{ name = "Saruman Ritual", kind = "long_term", current = 0, max = 8, direction = "increase" }
scene:countdown_update{ name = "Saruman Ritual", delta = 1, reason = "long_term_tick" }

scene:end_session()

return scene
