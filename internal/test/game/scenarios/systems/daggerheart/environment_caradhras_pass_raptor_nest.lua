local scn = Scenario.new("environment_caradhras_pass_raptor_nest")
local dh = scn:system("DAGGERHEART")

-- Capture the raptor nest reaction that summons predators.
scn:campaign{
  name = "Environment Caradhras Pass Raptor Nest",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
dh:adversary("Great Eagles")

-- The PCs enter a hunting ground and predators appear.
scn:start_session("Raptor Nest")
dh:gm_fear(1)

dh:adversary("Great Eagle Scout 1")
dh:adversary("Great Eagle Scout 2")
-- Range placement semantics remain unresolved in this fixture.
dh:gm_spend_fear(1):spotlight("Great Eagle Scout 1")

scn:end_session()

return scn
