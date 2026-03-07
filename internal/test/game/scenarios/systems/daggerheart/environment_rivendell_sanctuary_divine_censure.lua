local scene = Scenario.new("environment_rivendell_sanctuary_divine_censure")
local dh = scene:system("DAGGERHEART")

-- Capture the divine censure reaction summoning defenders.
scene:campaign{
  name = "Environment Rivendell Sanctuary Divine Censure",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
dh:adversary("Elf Warden")
dh:adversary("Gondor Guards")

-- The temple answers trespass with summoned defenders.
scene:start_session("Divine Censure")
dh:gm_fear(1)

-- Example: spend Fear to summon a Elf Warden and 1d4 guards.
dh:adversary("Gondor Guard Reinforcement")
-- Variable guard count and proximity-to-priest placement remain unresolved.
dh:gm_spend_fear(1):spotlight("Elf Warden")

scene:end_session()

return scene
