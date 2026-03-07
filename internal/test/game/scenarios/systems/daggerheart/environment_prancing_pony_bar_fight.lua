local scn = Scenario.new("environment_prancing_pony_bar_fight")
local dh = scn:system("DAGGERHEART")

-- Capture the bar fight hazard during tavern movement.
scn:campaign{
  name = "Environment Prancing Pony Bar Fight",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- A brawl erupts and movement triggers danger.
scn:start_session("Bar Fight")
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

scn:end_session()

return scn
