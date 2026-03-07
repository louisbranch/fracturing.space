local scn = Scenario.new("environment_isengard_ritual_blasphemous_might")
local dh = scn:system("DAGGERHEART")

-- Model the ritual action that imbues a cultist with power.
scn:campaign{
  name = "Environment Isengard Ritual Blasphemous Might",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
dh:adversary("Nazgul")

-- The GM imbues a cultist, granting attack advantage or extra damage.
scn:start_session("Blasphemous Might")
dh:gm_fear(1)

-- Advantage/bonus-damage/Relentless branch selection remains unresolved.
dh:gm_spend_fear(1):spotlight("Nazgul")

scn:end_session()

return scn
