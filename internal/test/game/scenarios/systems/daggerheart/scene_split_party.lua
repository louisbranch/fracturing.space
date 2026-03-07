-- Split party: two concurrent scenes with character transfer.
local scn = Scenario.new("scene_split_party")

scn:campaign{
  name = "Split Party Campaign",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "basics"
}

scn:pc("Frodo")
scn:pc("Sam")
scn:pc("Aragorn")

scn:start_session("Session")

-- Two concurrent scenes.
scn:create_scene{name = "Mines", characters = {"Frodo", "Sam"}}
scn:create_scene{name = "Forest", characters = {"Aragorn"}}

-- Sam moves from Mines to Forest.
scn:scene_transfer_character{from_scene = "Mines", to_scene = "Forest", character = "Sam"}

scn:end_scene{name = "Mines"}
scn:end_scene{name = "Forest"}
scn:end_session()

return scn
