local scene = Scenario.new("environment_caradhras_pass_raptor_nest")
local dh = scene:system("DAGGERHEART")

-- Capture the raptor nest reaction that summons predators.
scene:campaign{
  name = "Environment Caradhras Pass Raptor Nest",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
dh:adversary("Great Eagles")

-- The PCs enter a hunting ground and predators appear.
scene:start_session("Raptor Nest")
dh:gm_fear(1)

dh:adversary("Great Eagle Scout 1")
dh:adversary("Great Eagle Scout 2")
-- Range placement semantics remain unresolved in this fixture.
dh:gm_spend_fear(1):spotlight("Great Eagle Scout 1")

scene:end_session()

return scene
