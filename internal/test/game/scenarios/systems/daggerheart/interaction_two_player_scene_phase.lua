local scn = Scenario.new("interaction_two_player_scene_phase")

-- Frame a shared scene, let both players commit actions, and return authority
-- to the GM on the final yield.
scn:campaign{
  name = "Interaction Two Player Scene Phase",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "interaction"
}

-- Add the GM seat plus two player-controlled characters in one shared scene.
scn:participant({name = "Guide", role = "GM"})
scn:participant({name = "Rhea"}):character({name = "Aria"})
scn:participant({name = "Bryn"}):character({name = "Corin"})

-- Open the scene through the interaction surface.
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
  frame_text = "The bridge jerks under your boots and the far ropes groan. What do you do?",
  characters = {"Aria", "Corin"}
}
scn:interaction_expect{
  session = "Crossing",
  active_scene = "The Bridge",
  phase_status = "PLAYERS",
  acting_characters = {"Aria", "Corin"},
  acting_participants = {"Rhea", "Bryn"},
  gm_authority = "Guide"
}

-- Let each acting participant commit a post before the final yield returns to GM control.
scn:interaction_post{
  as = "Rhea",
  summary = "Aria drops low and grabs the near rope to steady the bridge.",
  characters = {"Aria"}
}
scn:interaction_post{
  as = "Bryn",
  summary = "Corin shields the lantern and inches toward the midpoint.",
  characters = {"Corin"}
}
scn:interaction_expect{
  phase_status = "PLAYERS",
  slots = {
    {participant = "Bryn", summary = "Corin shields the lantern and inches toward the midpoint.", characters = {"Corin"}},
    {participant = "Rhea", summary = "Aria drops low and grabs the near rope to steady the bridge.", characters = {"Aria"}}
  }
}
scn:interaction_yield({as = "Bryn", scene = "The Bridge"})
scn:interaction_expect{
  phase_status = "PLAYERS",
  slots = {
    {participant = "Bryn", summary = "Corin shields the lantern and inches toward the midpoint.", characters = {"Corin"}, yielded = true},
    {participant = "Rhea", summary = "Aria drops low and grabs the near rope to steady the bridge.", characters = {"Aria"}}
  }
}
scn:interaction_yield({as = "Rhea", scene = "The Bridge"})
scn:interaction_expect{
  phase_status = "GM_REVIEW",
  slots = {
    {participant = "Bryn", summary = "Corin shields the lantern and inches toward the midpoint.", characters = {"Corin"}, yielded = true, review_status = "UNDER_REVIEW"},
    {participant = "Rhea", summary = "Aria drops low and grabs the near rope to steady the bridge.", characters = {"Aria"}, yielded = true, review_status = "UNDER_REVIEW"}
  },
  gm_authority = "Guide"
}

scn:end_session()

return scn
