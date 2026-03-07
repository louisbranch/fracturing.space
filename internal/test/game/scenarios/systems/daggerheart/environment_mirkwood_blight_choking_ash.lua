local scn = Scenario.new("environment_mirkwood_blight_choking_ash")
local dh = scn:system("DAGGERHEART")

-- Model the looping choking ash countdown.
scn:campaign{
  name = "Environment Mirkwood Blight Choking Ash",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Gandalf")

-- The ash periodically forces reaction rolls.
scn:start_session("Choking Ash")

-- Loop countdown progression remains unresolved in this fixture.
dh:countdown_create{ name = "Choking Ash", kind = "loop", current = 0, max = 4, direction = "increase" }
dh:group_reaction{
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

scn:end_session()

return scn
