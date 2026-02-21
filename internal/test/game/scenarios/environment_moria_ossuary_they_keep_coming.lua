local scene = Scenario.new("environment_moria_ossuary_they_keep_coming")

-- Model the undead reinforcements action in the ossuary.
scene:campaign{
  name = "Environment Ossuary They Keep Coming",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
scene:adversary("Orc Rabble")
scene:adversary("Uruk-hai")

-- The necromancer calls in more undead.
scene:start_session("They Just Keep Coming")
scene:gm_fear(1)

scene:adversary("Rotted Zombie Reinforcement")
-- Branch choice (rotted/perfected/legion) remains unresolved in this fixture.
scene:gm_spend_fear(1):spotlight("Rotted Zombie Reinforcement")

scene:end_session()

return scene
