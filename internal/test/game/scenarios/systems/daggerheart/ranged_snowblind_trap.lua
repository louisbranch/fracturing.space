local scn = Scenario.new("ranged_snowblind_trap")
local dh = scn:system("DAGGERHEART")

-- Capture the Ranger of the North's snowblind trap action.
scn:campaign{
  name = "Ranged Snowblind Trap",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary"
}

scn:pc("Frodo")
scn:pc("Sam")
dh:adversary("Ranger of the North")

-- The hunter spends Fear to trap a group and apply Vulnerable.
scn:start_session("Snowblind Trap")
dh:gm_fear(1)

-- Example: targets fail Agility and become Vulnerable until a Strength/Finesse roll.
-- Missing DSL: apply group reaction rolls and Vulnerable condition.
dh:gm_spend_fear(1):spotlight("Ranger of the North")
dh:apply_condition{ target = "Frodo", add = { "VULNERABLE" }, source = "snowblind_trap" }
dh:apply_condition{ target = "Sam", add = { "VULNERABLE" }, source = "snowblind_trap" }

scn:end_session()

return scn
