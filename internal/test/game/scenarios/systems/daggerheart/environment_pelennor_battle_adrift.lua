local scn = Scenario.new("environment_pelennor_battle_adrift")
local dh = scn:system("DAGGERHEART")

-- Capture the movement restriction during active battle.
scn:campaign{
  name = "Environment Pelennor Battle Adrift",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- Movement requires agility under fire.
scn:start_session("Adrift on a Sea of Steel")

-- Missing DSL: restrict movement without a successful Agility roll.
dh:action_roll{ actor = "Frodo", trait = "agility", difficulty = 17, outcome = "hope" }
dh:apply_roll_outcome{}

scn:end_session()

return scn
