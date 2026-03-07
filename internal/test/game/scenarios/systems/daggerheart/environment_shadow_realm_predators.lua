local scn = Scenario.new("environment_shadow_realm_predators")
local dh = scn:system("DAGGERHEART")

-- Capture the summoning of outer realms predators.
scn:campaign{
  name = "Environment Shadow Realm Predators",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
dh:adversary("Nameless Horror")
dh:adversary("Shadow Corruptor")
dh:adversary("Shadow Thralls")

-- The chaos realm spawns monstrosities near the party.
scn:start_session("Outer Realms Predators")
dh:gm_fear(1)

dh:adversary("Shadow Thrall Reinforcement 1")
dh:adversary("Shadow Thrall Reinforcement 2")
-- Variable thrall count and exact spawn placement remain unresolved.
dh:gm_spend_fear(1):spotlight("Nameless Horror")

scn:end_session()

return scn
