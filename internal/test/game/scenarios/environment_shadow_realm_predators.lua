local scene = Scenario.new("environment_shadow_realm_predators")

-- Capture the summoning of outer realms predators.
scene:campaign{
  name = "Environment Shadow Realm Predators",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
scene:adversary("Nameless Horror")
scene:adversary("Shadow Corruptor")
scene:adversary("Shadow Thralls")

-- The chaos realm spawns monstrosities near the party.
scene:start_session("Outer Realms Predators")
scene:gm_fear(1)

scene:adversary("Shadow Thrall Reinforcement 1")
scene:adversary("Shadow Thrall Reinforcement 2")
-- Variable thrall count and exact spawn placement remain unresolved.
scene:gm_spend_fear(1):spotlight("Nameless Horror")

scene:end_session()

return scene
