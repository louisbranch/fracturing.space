local scn = Scenario.new("environment_moria_ossuary_they_keep_coming")
local dh = scn:system("DAGGERHEART")

-- Model the undead reinforcements action in the ossuary.
scn:campaign{
  name = "Environment Ossuary They Keep Coming",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
dh:adversary("Orc Rabble")
dh:adversary("Uruk-hai")

-- The necromancer calls in more undead.
scn:start_session("They Just Keep Coming")
dh:gm_fear(1)

dh:adversary("Rotted Zombie Reinforcement")
-- Branch choice (rotted/perfected/legion) remains unresolved in this fixture.
dh:gm_spend_fear(1):spotlight("Rotted Zombie Reinforcement")

scn:end_session()

return scn
