local scn = Scenario.new("interaction_simulated_ai_gm_scene_loop")

-- Setup a human-controlled GM participant that stands in for the future AI GM.
scn:campaign{
  name = "Interaction Simulated AI GM Scene Loop",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "interaction"
}

-- Add the GM seat plus two player-controlled characters in one shared scene.
scn:participant({name = "Guide", role = "GM"})
scn:participant({name = "Rhea"}):character({name = "Aria"})
scn:participant({name = "Bryn"}):character({name = "Corin"})

-- Open the session and frame the first scene through the interaction surface.
scn:start_session("Crossing")
scn:create_scene{
  name = "The Bridge",
  description = "Rain lashes the ropes and the planks sway over the gorge.",
  characters = {"Aria", "Corin"}
}
scn:interaction_set_gm_authority({participant = "Guide"})
scn:interaction_set_active_scene({scene = "The Bridge"})
scn:interaction_start_player_phase{
  scene = "The Bridge",
  frame_text = "Rain lashes the ropes and the bridge dips hard under your boots. What do you do next?",
  characters = {"Aria", "Corin"}
}
scn:interaction_expect{
  session = "Crossing",
  active_scene = "The Bridge",
  phase_status = "PLAYERS",
  frame_text = "Rain lashes the ropes and the bridge dips hard under your boots. What do you do next?",
  acting_characters = {"Aria", "Corin"},
  acting_participants = {"Rhea", "Bryn"},
  gm_authority = "Guide"
}

-- Let the first player commit an action summary without yielding the phase.
scn:interaction_post{
  as = "Rhea",
  summary = "Aria drops low and grabs the near rope to steady the span for everyone else.",
  characters = {"Aria"}
}
scn:interaction_expect{
  phase_status = "PLAYERS",
  acting_participants = {"Rhea", "Bryn"},
  slots = {
    {participant = "Rhea", summary = "Aria drops low and grabs the near rope to steady the span for everyone else.", characters = {"Aria"}},
    {participant = "Bryn"}
  }
}

-- Let the second player post and yield while the first player still owns the beat.
scn:interaction_post{
  as = "Bryn",
  summary = "Corin cups the lantern and edges toward the midpoint before the wind can snuff it out.",
  characters = {"Corin"},
  yield = true
}
scn:interaction_expect{
  phase_status = "PLAYERS",
  slots = {
    {participant = "Bryn", summary = "Corin cups the lantern and edges toward the midpoint before the wind can snuff it out.", characters = {"Corin"}, yielded = true},
    {participant = "Rhea", summary = "Aria drops low and grabs the near rope to steady the span for everyone else.", characters = {"Aria"}}
  }
}

-- Yield the final acting participant and hand the beat into GM review.
scn:interaction_yield({as = "Rhea"})
scn:interaction_expect{
  phase_status = "GM_REVIEW",
  slots = {
    {participant = "Bryn", summary = "Corin cups the lantern and edges toward the midpoint before the wind can snuff it out.", characters = {"Corin"}, yielded = true, review_status = "UNDER_REVIEW"},
    {participant = "Rhea", summary = "Aria drops low and grabs the near rope to steady the span for everyone else.", characters = {"Aria"}, yielded = true, review_status = "UNDER_REVIEW"}
  },
  gm_authority = "Guide"
}
scn:interaction_accept_player_phase({as = "Guide"})

-- Simulate the AI GM by having the GM participant immediately frame the next beat.
scn:interaction_start_player_phase{
  scene = "The Bridge",
  frame_text = "The far anchor snaps loose and the lantern swings over the gorge. Who catches it before it drops?",
  characters = {"Aria", "Corin"}
}
scn:interaction_expect{
  phase_status = "PLAYERS",
  frame_text = "The far anchor snaps loose and the lantern swings over the gorge. Who catches it before it drops?",
  acting_characters = {"Aria", "Corin"},
  acting_participants = {"Rhea", "Bryn"},
  slots = {
    {participant = "Bryn"},
    {participant = "Rhea"}
  }
}

scn:end_session()

return scn
