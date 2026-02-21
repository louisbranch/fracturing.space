local scene = Scenario.new("environment_mirkwood_blight_choking_ash")

-- Model the looping choking ash countdown.
scene:campaign{
  name = "Environment Mirkwood Blight Choking Ash",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Gandalf")

-- The ash periodically forces reaction rolls.
scene:start_session("Choking Ash")

-- Loop countdown progression remains unresolved in this fixture.
scene:countdown_create{ name = "Choking Ash", kind = "loop", current = 0, max = 4, direction = "increase" }
scene:group_reaction{
  targets = {"Gandalf"},
  trait = "strength",
  difficulty = 16,
  outcome = "fear",
  damage = 12,
  damage_type = "magic",
  direct = true,
  half_damage_on_success = true,
  source = "choking_ash"
}

scene:end_session()

return scene
