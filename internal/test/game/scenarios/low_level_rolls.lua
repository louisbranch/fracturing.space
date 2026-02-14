local scene = Scenario.new("low_level_rolls")

scene:campaign{ name = "Low Level Rolls", system = "DAGGERHEART", gm_mode = "HUMAN" }
scene:pc("Nia")
scene:adversary("Goblin")
scene:start_session("Low Level")

scene:action_roll{ actor = "Nia", trait = "instinct", difficulty = 8, outcome = "hope" }
scene:apply_roll_outcome{}
scene:apply_attack_outcome{ targets = { "Goblin" } }

scene:damage_roll{ actor = "Nia", damage_dice = { { sides = 6, count = 2 } }, modifier = 1, seed = 21 }

scene:adversary_attack_roll{ actor = "Goblin", attack_modifier = 1, advantage = 1, seed = 33 }
scene:apply_adversary_attack_outcome{ targets = { "Nia" }, difficulty = 10 }

scene:reaction_roll{ actor = "Nia", trait = "agility", difficulty = 8, outcome = "hope" }
scene:apply_reaction_outcome{}

scene:end_session()
return scene
