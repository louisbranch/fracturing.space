local scn = Scenario.new("interaction_simulated_ai_gm_action_roll")
local dh = scn:system("DAGGERHEART")

-- Set up a simulated AI GM loop where player conversation and mechanics interleave.
scn:campaign{
  name = "Interaction Simulated AI GM Action Roll",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "interaction"
}

-- Create one GM seat and two player characters who will trade spotlight beats.
scn:participant({name = "Guide", role = "GM"})
scn:participant({name = "Rhea"}):character({name = "Aria"})
scn:participant({name = "Bryn"}):character({name = "Corin"})

-- Open the cliffside scene and hand pacing control to the interaction surface.
scn:start_session("Cliffside Rescue")
scn:create_scene{
  name = "Storm Ledge",
  description = "A storm tears at the cliff path while a trapped scout clings to the far ledge.",
  characters = {"Aria", "Corin"}
}
scn:interaction_set_gm_authority({participant = "Guide"})
scn:interaction_set_active_scene({scene = "Storm Ledge"})
scn:interaction_start_player_phase{
  scene = "Storm Ledge",
  frame_text = "The scout is slipping and the cliff path is crumbling under the rain. What do you do?",
  characters = {"Aria", "Corin"}
}

-- Have the first player commit intent and execute a real action roll during the same beat.
scn:interaction_post{
  as = "Rhea",
  summary = "Aria darts for the loose mooring pin before the rope line tears free.",
  characters = {"Aria"}
}
dh:action_roll{
  as = "Rhea",
  actor = "Aria",
  trait = "agility",
  difficulty = 12,
  outcome = "success_fear"
}
scn:interaction_yield({as = "Rhea"})

-- Let the second player respond in-character and close the shared player phase.
scn:interaction_post{
  as = "Bryn",
  summary = "Corin braces the line and shouts directions to the trapped scout.",
  characters = {"Corin"},
  yield = true
}
scn:interaction_expect{
  phase_status = "GM_REVIEW",
  gm_authority = "Guide",
  slots = {
    {participant = "Bryn", summary = "Corin braces the line and shouts directions to the trapped scout.", characters = {"Corin"}, yielded = true, review_status = "UNDER_REVIEW"},
    {participant = "Rhea", summary = "Aria darts for the loose mooring pin before the rope line tears free.", characters = {"Aria"}, yielded = true, review_status = "UNDER_REVIEW"}
  }
}
scn:interaction_accept_player_phase({as = "Guide"})

-- Simulate the AI GM reacting to the result by narrowing the next beat to one player.
scn:interaction_start_player_phase{
  scene = "Storm Ledge",
  frame_text = "Aria has the line, but the scout is panicking. Corin, what do you say to keep them moving?",
  characters = {"Corin"}
}
scn:interaction_expect{
  phase_status = "PLAYERS",
  acting_characters = {"Corin"},
  acting_participants = {"Bryn"},
  frame_text = "Aria has the line, but the scout is panicking. Corin, what do you say to keep them moving?"
}
scn:interaction_post{
  as = "Bryn",
  summary = "Corin calls out a steady cadence and points the scout toward Aria's line.",
  characters = {"Corin"},
  yield = true
}
scn:interaction_expect{
  phase_status = "GM_REVIEW",
  slots = {
    {participant = "Bryn", summary = "Corin calls out a steady cadence and points the scout toward Aria's line.", characters = {"Corin"}, yielded = true, review_status = "UNDER_REVIEW"}
  },
  gm_authority = "Guide"
}

scn:end_session()

return scn
