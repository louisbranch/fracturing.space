-- Session conclusion: GM delivers final interaction via interaction_conclude_session.
-- Models a starter campaign wrap-up where the adventure arc completes.
--
-- Uses the conclude session tool to commit the final interaction, store
-- the recap, end all scenes, and close the session in one authoritative call.
local scn = Scenario.new("session_conclusion_flow")

scn:campaign{
  name = "Session Conclusion Flow",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "starter lifecycle"
}

scn:participant({name = "Guide", role = "GM"})
scn:participant({name = "Player"}):character({name = "Kael"})

scn:start_session("Opening Night")

scn:create_scene{
  name = "Lantern Room",
  description = "The top of the Ivory Beacon. Broken glass and the ignition crystal.",
  characters = {"Kael"}
}
scn:interaction_set_session_gm_authority({participant = "Guide"})

-- GM sets the climactic scene and opens player phase.
scn:interaction_open_scene_player_phase{
  as = "Guide",
  scene = "Lantern Room",
  interaction = {
    title = "Showdown",
    beats = {
      {type = "fiction", text = "Voss holds the crystal over the railing. The wind howls through broken panes."},
      {type = "prompt", text = "Kael, Voss is cornered. What do you do?"},
    },
  },
  characters = {"Kael"},
}

-- Player submits climactic action.
scn:interaction_submit_scene_player_action{
  as = "Player",
  scene = "Lantern Room",
  summary = "I tell Voss that Lira is safe and offer him a chance to surrender.",
  characters = {"Kael"},
  yield = true
}

-- GM concludes the session with the conclusion tool.
scn:interaction_conclude_session{
  as = "Guide",
  conclusion = "Voss lowers the crystal. His shoulders sag. He hands it over without a word. Kael slots the crystal into the mechanism. The Ivory Beacon blazes to life, cutting through the fog over the Shattered Straits.",
  summary = [[## Key Events
- Kael confronted Voss in the lantern room
- Voss surrendered the ignition crystal
- The Ivory Beacon was relit

## NPCs Met
- Voss "Blacktide" Corran: smuggler antagonist, surrendered
- Lira Corran: Voss's daughter, safe

## Decisions Made
- Kael chose diplomacy over force

## Unresolved Threads
- Voss's smuggling network may still operate without him

## Next Session Hooks
- Brinewall celebrates; what comes next for Kael?]],
  end_campaign = true,
  epilogue = "With the Ivory Beacon blazing again, the trade fleet navigates safely through the Shattered Straits. Brinewall celebrates with a bonfire feast on the beach. Kael stands at the lighthouse railing, watching the ships pass, knowing the coast is safe — for now."
}

return scn
