local scene = Scenario.new("social_village_elder_peace")

-- Capture the village elder's social reactions and deterrents.
scene:campaign{
  name = "Social Shire Elder Peace",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "social"
}

scene:pc("Frodo")
scene:adversary("Shire Elder")

-- The elder forbids hospitality and invokes peace when attacked.
scene:start_session("Village Council")
scene:gm_fear(2)

-- Example: No Hospitality action and There Will Be Peace reaction.
-- Missing DSL: apply hospitality ban, Hope loss, Stress, and unconscious condition.
scene:gm_spend_fear(2):spotlight("Shire Elder")
scene:apply_condition{ target = "Frodo", life_state = "unconscious", source = "there_will_be_peace" }

scene:end_session()

return scene
