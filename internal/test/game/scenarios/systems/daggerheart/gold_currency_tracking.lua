local scn = Scenario.new("gold_currency_tracking")
local dh = scn:system("DAGGERHEART")

-- Verify gold denomination updates and tracking.
scn:campaign{
  name = "Gold Currency Tracking",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "inventory"
}

scn:pc("Frodo")

-- Update gold: gain a handful from quest reward.
scn:start_session("Gold Tracking")
dh:update_gold{
  target = "Frodo",
  handfuls_before = 0,
  handfuls_after = 3,
  bags_before = 0,
  bags_after = 0,
  chests_before = 0,
  chests_after = 0,
  reason = "quest_reward",
}

-- Spend gold: buy supplies.
dh:update_gold{
  target = "Frodo",
  handfuls_before = 3,
  handfuls_after = 1,
  bags_before = 0,
  bags_after = 0,
  chests_before = 0,
  chests_after = 0,
  reason = "purchase_supplies",
}

-- Gain a bag of gold from treasure.
dh:update_gold{
  target = "Frodo",
  handfuls_before = 1,
  handfuls_after = 1,
  bags_before = 0,
  bags_after = 1,
  chests_before = 0,
  chests_after = 0,
  reason = "treasure_find",
}

scn:end_session()

return scn
