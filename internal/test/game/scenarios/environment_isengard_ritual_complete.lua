local scene = Scenario.new("environment_isengard_ritual_complete")

-- Model the ritual leader's protection reaction.
scene:campaign{
  name = "Environment Isengard Ritual Complete",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
scene:adversary("Saruman")
scene:adversary("Orc Raider")

-- An ally steps in to take a hit meant for the leader.
scene:start_session("Complete the Ritual")

-- Trigger timing/eligibility for the redirect remains unresolved.
scene:adversary_update{ target = "Orc Raider", stress_delta = 1, notes = "protect_ritual_leader" }
scene:attack{ actor = "Frodo", target = "Orc Raider", trait = "instinct", difficulty = 0, outcome = "hope", damage_type = "physical" }

scene:end_session()

return scene
