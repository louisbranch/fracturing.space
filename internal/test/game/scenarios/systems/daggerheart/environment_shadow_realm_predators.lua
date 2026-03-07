local scene = Scenario.new("environment_shadow_realm_predators")
local dh = scene:system("DAGGERHEART")

-- Capture the summoning of outer realms predators.
scene:campaign{
  name = "Environment Shadow Realm Predators",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
dh:adversary("Nameless Horror")
dh:adversary("Shadow Corruptor")
dh:adversary("Shadow Thralls")

-- The chaos realm spawns monstrosities near the party.
scene:start_session("Outer Realms Predators")
dh:gm_fear(1)

dh:adversary("Shadow Thrall Reinforcement 1")
dh:adversary("Shadow Thrall Reinforcement 2")
-- Variable thrall count and exact spawn placement remain unresolved.
dh:gm_spend_fear(1):spotlight("Nameless Horror")

scene:end_session()

return scene
