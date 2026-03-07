local scene = Scenario.new("environment_moria_ossuary_aura_of_death")
local dh = scene:system("DAGGERHEART")

-- Capture undead healing from the aura of death.
scene:campaign{
  name = "Environment Ossuary Aura of Death",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
dh:adversary("Uruk-hai")

-- The aura restores undead HP and Stress.
scene:start_session("Aura of Death")
dh:gm_fear(1)

-- Healing distribution across undead remains unresolved.
dh:gm_spend_fear(1):spotlight("Uruk-hai")

scene:end_session()

return scene
