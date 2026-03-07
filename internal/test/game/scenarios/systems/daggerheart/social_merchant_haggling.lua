local scn = Scenario.new("social_merchant_haggling")
local dh = scn:system("DAGGERHEART")

-- Model social mechanics for haggling with a merchant.
scn:campaign{
  name = "Social Bilbo Haggling",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "social"
}

scn:pc("Frodo")
dh:adversary("Bree Merchant")

-- The merchant rewards success and penalizes poor rolls.
scn:start_session("Haggling")

-- Example: success grants discounts, failure adds stress and disadvantage.
-- Partial mapping: discount pressure and runaround consequences are explicit by branch.
-- Missing DSL: first-class pricing modifiers and direct stress mark operations.
dh:action_roll{ actor = "Frodo", trait = "presence", difficulty = 12, outcome = "success_fear" }
dh:apply_roll_outcome{
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

scn:end_session()

return scn
