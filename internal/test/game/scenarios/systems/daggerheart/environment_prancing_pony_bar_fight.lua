local scene = Scenario.new("environment_prancing_pony_bar_fight")
local dh = scene:system("DAGGERHEART")

-- Capture the bar fight hazard during tavern movement.
scene:campaign{
  name = "Environment Prancing Pony Bar Fight",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- A brawl erupts and movement triggers danger.
scene:start_session("Bar Fight")
dh:gm_fear(1)

-- Example: spend Fear; moving requires Agility/Presence or take 1d6+2 damage.
-- Trait choice (Agility vs Presence) remains unresolved; this fixture uses Agility.
dh:gm_spend_fear(1):spotlight("Prancing Pony")
dh:group_reaction{
  targets = {"Frodo"},
  trait = "agility",
  difficulty = 10,
  outcome = "fear",
  damage = 5,
  damage_type = "physical",
  source = "bar_fight_hazard"
}

scene:end_session()

return scene
