local scene = Scenario.new("social_merchant_haggling")

-- Model social mechanics for haggling with a merchant.
scene:campaign{
  name = "Social Bilbo Haggling",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "social"
}

scene:pc("Frodo")
scene:adversary("Bree Merchant")

-- The merchant rewards success and penalizes poor rolls.
scene:start_session("Haggling")

-- Example: success grants discounts, failure adds stress and disadvantage.
-- Partial mapping: discount pressure and runaround consequences are explicit by branch.
-- Missing DSL: first-class pricing modifiers and direct stress mark operations.
scene:action_roll{ actor = "Frodo", trait = "presence", difficulty = 12, outcome = "success_fear" }
scene:apply_roll_outcome{
  on_success = {
    {kind = "adversary_update", target = "Bree Merchant", notes = "preferential_treatment_discount"},
  },
  on_failure = {
    {kind = "apply_condition", target = "Frodo", add = {"VULNERABLE"}, source = "merchant_price_penalty"},
  },
  on_fear = {
    {kind = "apply_condition", target = "Frodo", add = {"VULNERABLE"}, source = "merchant_runaround"},
  },
}

scene:end_session()

return scene
