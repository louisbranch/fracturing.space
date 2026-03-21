local scn = Scenario.new("adversary_box_in_focus_target")
local dh = scn:system("DAGGERHEART")

-- Model the Vault Guardian Sentinel's Box In feature that applies
-- disadvantage against the focused target.
scn:campaign{
  name = "Adversary Box In Focus Target",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary"
}

scn:pc("Frodo")
scn:pc("Samwise")
dh:adversary("Vault Guardian Sentinel")

-- Stage the Box In feature against Frodo.
scn:start_session("Box In Focus Target")

dh:adversary_feature{
  actor = "Vault Guardian Sentinel",
  feature_id = "adversary-feature.vault-guardian-sentinel-box-in",
  target = "Frodo"
}

-- Attack Frodo — the focused target gets disadvantage applied by the
-- runtime FocusTargetDisadvantage rule.
dh:adversary_attack{
  actor = "Vault Guardian Sentinel",
  target = "Frodo",
  feature_id = "adversary-feature.vault-guardian-sentinel-box-in",
  difficulty = 17,
  damage_type = "physical"
}

-- Attack Samwise — not the focused target, so no extra disadvantage.
dh:adversary_attack{
  actor = "Vault Guardian Sentinel",
  target = "Samwise",
  feature_id = "adversary-feature.vault-guardian-sentinel-box-in",
  difficulty = 17,
  damage_type = "physical"
}

scn:end_session()

return scn
