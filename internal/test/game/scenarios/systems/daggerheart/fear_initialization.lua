local scn = Scenario.new("fear_initialization")
local dh = scn:system("DAGGERHEART")

-- Set up a first-session start with two PCs and one NPC.
scn:campaign{
  name = "Fear Initialization",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "fear"
}

-- Add two PCs and one NPC so the seed count can exclude NPCs.
scn:pc("Frodo")
scn:pc("Sam")
scn:npc("Guide")

-- First-session Daggerheart fear should seed from PC count only.
scn:start_session("Fear Seed")
dh:expect_gm_fear(2)
scn:end_session()

return scn
