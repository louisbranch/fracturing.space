local scn = Scenario.new("adversary_horde_bloodied")
local dh = scn:system("DAGGERHEART")

scn:campaign{
  name = "Adversary Horde Bloodied",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary_rules"
}

scn:pc("Frodo", { armor = 0 })
scn:pc("Sam", { armor = 1 })
dh:adversary("Swarm of Rats", { adversary_entry_id = "adversary.swarm-of-rats" })

scn:start_session("Horde Bloodied")

dh:adversary_attack{
  actor = "Swarm of Rats",
  target = "Frodo",
  difficulty = 0,
  seed = 17,
  damage_type = "physical",
  expect_hp_delta = -3,
  expect_armor_delta = 0
}

dh:combined_damage{
  target = "Swarm of Rats",
  damage_type = "physical",
  sources = {
    { character = "Frodo", amount = 1 }
  },
  expect_adversary_hp_delta = -1
}

dh:adversary_attack{
  actor = "Swarm of Rats",
  target = "Sam",
  difficulty = 0,
  seed = 17,
  damage_type = "physical",
  expect_hp_delta = -1,
  expect_armor_delta = -1
}

scn:end_session()

return scn
