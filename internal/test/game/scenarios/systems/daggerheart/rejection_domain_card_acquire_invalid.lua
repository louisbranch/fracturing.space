local scn = Scenario.new("rejection_domain_card_acquire_invalid")
local dh = scn:system("DAGGERHEART")

-- Acquiring a domain card to an invalid destination should be rejected.
scn:campaign{
  name = "Rejection Domain Card Acquire Invalid",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "rejection"
}

scn:pc("Frodo")

scn:start_session("Rejection")

-- Attempt to acquire to an invalid slot (not vault or loadout).
dh:acquire_domain_card{
  target = "Frodo",
  card_id = "domain.valor.1",
  destination = "invalid_slot",
  expect_error = {code = "INTERNAL"}
}

scn:end_session()
return scn
