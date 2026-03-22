local scn = Scenario.new("environment_shadow_realm_everything_you_are")
local dh = scn:system("DAGGERHEART")

-- Capture the looping countdown that drains a PC's highest trait.
scn:campaign{
  name = "Environment Shadow Realm Everything You Are",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- The realm threatens to strip core strength away.
scn:start_session("Everything You Are")

-- Missing DSL: loop countdown and reduce highest trait or mark stress.
dh:scene_countdown_create{ name = "Chaos Drain", kind = "loop", current = 0, max = 4, direction = "increase" }
dh:reaction_roll{ actor = "Frodo", trait = "presence", difficulty = 20, outcome = "fear" }
dh:apply_reaction_outcome{}

scn:end_session()

return scn
