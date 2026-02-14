local scene = Scenario.new("tag_team")

-- Pair Frodo and Sam for a tag team maneuver.
scene:campaign{
  name = "Tag Team",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "teamwork"
}

scene:pc("Frodo")
scene:pc("Sam")

-- Frodo and Sam attempt a tag team maneuver to tackle a shared obstacle.
scene:start_session("Tag Team")

-- Frodo is selected as the final roller after both contribute.
-- Missing DSL: specify the chosen outcome and any resource effects.
scene:tag_team{
  first = "Frodo",
  first_trait = "instinct",
  second = "Sam",
  second_trait = "agility",
  selected = "Frodo",
  difficulty = 10
}

-- Close the session after the tag team attempt.
scene:end_session()

return scene
