local scn = Scenario.new("social_village_elder_peace")
local dh = scn:system("DAGGERHEART")

-- Capture the village elder's social reactions and deterrents.
scn:campaign{
  name = "Social Shire Elder Peace",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "social"
}

scn:pc("Frodo")
dh:adversary("Shire Elder")

-- The elder forbids hospitality and invokes peace when attacked.
scn:start_session("Village Council")
dh:gm_fear(2)

-- Example: No Hospitality action and There Will Be Peace reaction.
-- Partial mapping: trigger roll, fear spend, and incapacitating outcome are explicit.
-- Missing DSL: direct Hope/Stress social penalties and hospitality access state.
dh:action_roll{ actor = "Frodo", trait = "presence", difficulty = 15, outcome = "failure_fear" }
dh:apply_roll_outcome{
  on_failure_fear = {
    {kind = "gm_spend_fear", amount = 2, target = "Shire Elder", description = "there_will_be_peace_rebuke"},
    {kind = "apply_condition", target = "Frodo", add = {"VULNERABLE"}, source = "no_hospitality"},
    {kind = "apply_condition", target = "Frodo", life_state = "unconscious", source = "there_will_be_peace"},
  },
}

scn:end_session()

return scn
