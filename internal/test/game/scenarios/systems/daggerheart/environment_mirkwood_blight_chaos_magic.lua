local scn = Scenario.new("environment_mirkwood_blight_chaos_magic")
local dh = scn:system("DAGGERHEART")

-- Model the chaos locus forcing double Fear dice on spellcasting.
scn:campaign{
  name = "Environment Mirkwood Blight Chaos Magic",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Gandalf")

-- Spellcasting draws extra Fear in the corrupted woods.
scn:start_session("Chaos Magic Locus")

-- Missing DSL: roll two Fear dice and take the higher on Spellcast.
dh:action_roll{ actor = "Gandalf", trait = "spellcast", difficulty = 16, outcome = "fear" }
dh:apply_roll_outcome{}

scn:end_session()

return scn
