local scn = Scenario.new("interaction_multi_character_owner")

-- One participant owns multiple active characters and submits one combined
-- post that references both.
scn:campaign{
  name = "Interaction Multi Character Owner",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "interaction"
}

scn:participant({name = "Guide", role = "GM"})
local rhea = scn:participant({name = "Rhea"})
rhea:character({name = "Aria"})
rhea:character({name = "Sable"})

-- Both characters share one active scene and one participant-owned post slot.
scn:start_session("Courtyard")
scn:create_scene{
  name = "Moonlit Courtyard",
  description = "Aria and Sable confer beside the fountain while the keep sleeps.",
  characters = {"Aria", "Sable"}
}
scn:interaction_set_gm_authority({participant = "Guide"})
scn:interaction_set_active_scene({scene = "Moonlit Courtyard"})
scn:interaction_start_player_phase{
  scene = "Moonlit Courtyard",
  frame_text = "The fountain masks your voices for now. How do the two of you move next?",
  characters = {"Aria", "Sable"}
}
scn:interaction_expect{
  phase_status = "PLAYERS",
  acting_characters = {"Aria", "Sable"},
  acting_participants = {"Rhea"}
}
scn:interaction_post{
  as = "Rhea",
  summary = "Aria keeps watch at the archway while Sable slips to the fountain to inspect the satchel.",
  characters = {"Aria", "Sable"}
}
scn:interaction_expect{
  phase_status = "PLAYERS",
  slots = {
    {participant = "Rhea", summary = "Aria keeps watch at the archway while Sable slips to the fountain to inspect the satchel.", characters = {"Aria", "Sable"}}
  }
}
scn:interaction_yield({as = "Rhea", scene = "Moonlit Courtyard"})
scn:interaction_expect{
  phase_status = "GM_REVIEW",
  slots = {
    {participant = "Rhea", summary = "Aria keeps watch at the archway while Sable slips to the fountain to inspect the satchel.", characters = {"Aria", "Sable"}, yielded = true, review_status = "UNDER_REVIEW"}
  },
  gm_authority = "Guide"
}
scn:interaction_accept_player_phase({as = "Guide"})
scn:interaction_expect{
  phase_status = "GM",
  slots = {},
  gm_authority = "Guide"
}

scn:end_session()

return scn
