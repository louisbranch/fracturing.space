local scn = Scenario.new("low_level_rolls")
local dh = scn:system("DAGGERHEART")

-- Set a low-stakes skirmish to exercise roll plumbing.
scn:campaign{ name = "Low Level Rolls", system = "DAGGERHEART", gm_mode = "HUMAN" }
scn:pc("Sam")
dh:adversary("Goblin")

-- Open a session to run through low-level rolls.
scn:start_session("Low Level")

-- Resolve Sam's action roll and apply the attack outcome.
dh:action_roll{ actor = "Sam", trait = "instinct", difficulty = 8, outcome = "hope" }
dh:apply_roll_outcome{}
dh:apply_attack_outcome{ targets = { "Goblin" } }

-- Roll damage for Sam's attack.
dh:damage_roll{ actor = "Sam", damage_dice = { { sides = 6, count = 2 } }, modifier = 1, seed = 21 }

-- Resolve the goblin's counterattack and apply the outcome.
dh:adversary_attack_roll{ actor = "Goblin", attack_modifier = 1, advantage = 1, seed = 33 }
dh:apply_adversary_attack_outcome{ targets = { "Sam" }, difficulty = 10 }

-- Resolve Sam's reaction roll and apply the outcome.
dh:reaction_roll{ actor = "Sam", trait = "agility", difficulty = 8, outcome = "hope" }
dh:apply_reaction_outcome{}

-- Close the session once the roll chain completes.
scn:end_session()
return scn
