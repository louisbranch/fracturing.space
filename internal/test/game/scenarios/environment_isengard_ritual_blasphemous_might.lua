local scene = Scenario.new("environment_isengard_ritual_blasphemous_might")

-- Model the ritual action that imbues a cultist with power.
scene:campaign{
  name = "Environment Isengard Ritual Blasphemous Might",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
scene:adversary("Nazgul")

-- The GM imbues a cultist, granting attack advantage or extra damage.
scene:start_session("Blasphemous Might")
scene:gm_fear(1)

-- Advantage/bonus-damage/Relentless branch selection remains unresolved.
scene:gm_spend_fear(1):spotlight("Nazgul")

scene:end_session()

return scene
