local scn = Scenario.new("domain_card_acquire")
local dh = scn:system("DAGGERHEART")

-- Verify domain card acquisition into vault and loadout.
scn:campaign{
  name = "Domain Card Acquire",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "inventory"
}

scn:pc("Frodo")

-- Acquire a domain card into vault.
scn:start_session("Card Acquire")
dh:acquire_domain_card{
  target = "Frodo",
  card_id = "blade_domain_strike",
  card_level = 1,
  destination = "vault",
}

-- Acquire another card directly to loadout.
dh:acquire_domain_card{
  target = "Frodo",
  card_id = "sage_domain_insight",
  card_level = 2,
  destination = "loadout",
}

scn:end_session()

return scn
