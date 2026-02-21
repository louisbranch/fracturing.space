local scene = Scenario.new("combat_objectives_ritual_rescue_capture")

-- Track multiple combat objectives during a ritual confrontation.
-- Clarification-gated fixture (P31): do not infer implicit multi-objective fanout.
scene:campaign{
  name = "Combat Objectives Ritual Rescue Capture",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "objectives"
}

scene:pc("Frodo")
scene:pc("Sam")
scene:adversary("Saruman")
scene:npc("Bilbo")

-- The party tries to stop a ritual, save Bilbo, and capture Saruman.
scene:start_session("Ritual Objectives")

-- Example: three objectives run in parallel during the fight.
-- Missing DSL: connect action rolls to each objective's progress.
scene:countdown_create{ name = "Ritual Completion", kind = "consequence", current = 0, max = 6, direction = "increase" }
scene:countdown_create{ name = "Bilbo Rescued", kind = "progress", current = 0, max = 4, direction = "increase" }
scene:countdown_create{ name = "Saruman Captured", kind = "progress", current = 0, max = 4, direction = "increase" }

scene:end_session()

return scene
