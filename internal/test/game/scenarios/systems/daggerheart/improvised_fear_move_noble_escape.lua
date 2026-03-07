local scn = Scenario.new("improvised_fear_move_noble_escape")
local dh = scn:system("DAGGERHEART")

-- Model the improvised fear move that lets a villain escape.
scn:campaign{
  name = "Improvised Fear Move Noble Escape",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "gm_fear"
}

scn:pc("Sam")
dh:adversary("Corrupt Steward")

-- The GM spends Fear to remove a near-certain victory.
scn:start_session("Noble Escape")
dh:gm_fear(1)

-- Example: the noble reveals a surprise escape to avoid defeat.
-- Partial mapping: fear spend, concrete consequence, and spotlight handoff are explicit.
-- Missing DSL: adversary-focused spotlight state without routing through the GM spotlight.
dh:gm_spend_fear(1):spotlight("Corrupt Steward", { description = "noble_escape_through_secret_passage" })
dh:countdown_create{ name = "Seal the Escape Route", kind = "progress", current = 0, max = 4, direction = "increase" }
dh:countdown_update{ name = "Seal the Escape Route", delta = 1, reason = "steward_breaks_contact" }
dh:apply_condition{ target = "Sam", add = { "VULNERABLE" }, source = "noble_escape_distraction" }
scn:set_spotlight{ target = "Sam" }

-- Close the session after the escape move.
scn:end_session()

return scn
