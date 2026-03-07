local scn = Scenario.new("experience_spend_modifier")
local dh = scn:system("DAGGERHEART")

-- Model spending Hope to apply an Experience modifier.
scn:campaign{
  name = "Experience Spend Modifier",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "experience"
}

scn:pc("Frodo")

-- Frodo uses a relevant Experience by spending Hope for a modifier.
scn:start_session("Experience Modifier")

dh:action_roll{
  actor = "Frodo",
  trait = "presence",
  difficulty = 12,
  outcome = "hope",
  modifiers = {
    Modifiers.hope("experience"),
    Modifiers.mod("training", 3),
  }
}

scn:end_session()

return scn
