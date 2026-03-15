local scn = Scenario.new("interaction_edit_post_and_unyield")

-- A participant submits, yields, gets a GM revision request, keeps the prior
-- slot text visible, revises, and returns the phase for final review.
scn:campaign{
  name = "Interaction Edit Post And Unyield",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "interaction"
}

scn:participant({name = "Guide", role = "GM"})
scn:participant({name = "Rhea"}):character({name = "Aria"})
scn:participant({name = "Bryn"}):character({name = "Corin"})

scn:start_session("Flooded Archive")
scn:create_scene{
  name = "Flooded Archive",
  description = "Water laps around collapsed shelves and a ledger glints beneath the drift.",
  characters = {"Aria", "Corin"}
}
scn:interaction_set_gm_authority({participant = "Guide"})
scn:interaction_set_active_scene({scene = "Flooded Archive"})
scn:interaction_start_player_phase{
  scene = "Flooded Archive",
  frame_text = "The archive is flooding fast. What do you do before the ledger is lost?",
  characters = {"Aria", "Corin"}
}

-- Both players submit and yield so the beat moves into GM review.
scn:interaction_post{
  as = "Rhea",
  summary = "Aria hangs back at the doorway and studies the safest route through the water.",
  characters = {"Aria"}
}
scn:interaction_post{
  as = "Bryn",
  summary = "Corin braces a fallen shelf against the current and clears a path back to the door.",
  characters = {"Corin"},
  yield = true
}
scn:interaction_yield({as = "Rhea", scene = "Flooded Archive"})
scn:interaction_expect{
  phase_status = "GM_REVIEW",
  slots = {
    {participant = "Bryn", summary = "Corin braces a fallen shelf against the current and clears a path back to the door.", characters = {"Corin"}, yielded = true, review_status = "UNDER_REVIEW"},
    {participant = "Rhea", summary = "Aria hangs back at the doorway and studies the safest route through the water.", characters = {"Aria"}, yielded = true, review_status = "UNDER_REVIEW"}
  },
  gm_authority = "Guide"
}

-- The GM requests one slot revision; the original text stays visible.
scn:interaction_request_revisions{
  as = "Guide",
  scene = "Flooded Archive",
  revisions = {
    {participant = "Rhea", reason = "Commit to the route through the floodwater.", characters = {"Aria"}}
  }
}
scn:interaction_expect{
  phase_status = "PLAYERS",
  slots = {
    {participant = "Bryn", summary = "Corin braces a fallen shelf against the current and clears a path back to the door.", characters = {"Corin"}, yielded = true, review_status = "ACCEPTED"},
    {participant = "Rhea", summary = "Aria hangs back at the doorway and studies the safest route through the water.", characters = {"Aria"}, review_status = "CHANGES_REQUESTED", review_reason = "Commit to the route through the floodwater.", review_characters = {"Aria"}}
  }
}

-- Repost the updated plan, then yield again so the beat re-enters GM review.
scn:interaction_post{
  as = "Rhea",
  summary = "Aria wades in, hooks the ledger free, and signals for Corin to cover the retreat.",
  characters = {"Aria"}
}
scn:interaction_expect{
  phase_status = "PLAYERS",
  slots = {
    {participant = "Bryn", summary = "Corin braces a fallen shelf against the current and clears a path back to the door.", characters = {"Corin"}, yielded = true, review_status = "ACCEPTED"},
    {participant = "Rhea", summary = "Aria wades in, hooks the ledger free, and signals for Corin to cover the retreat.", characters = {"Aria"}}
  }
}
scn:interaction_yield({as = "Rhea", scene = "Flooded Archive"})
scn:interaction_expect{
  phase_status = "GM_REVIEW",
  slots = {
    {participant = "Bryn", summary = "Corin braces a fallen shelf against the current and clears a path back to the door.", characters = {"Corin"}, yielded = true, review_status = "UNDER_REVIEW"},
    {participant = "Rhea", summary = "Aria wades in, hooks the ledger free, and signals for Corin to cover the retreat.", characters = {"Aria"}, yielded = true, review_status = "UNDER_REVIEW"}
  }
}
scn:interaction_accept_player_phase({as = "Guide", scene = "Flooded Archive"})
scn:interaction_expect{
  phase_status = "GM",
  slots = {},
  gm_authority = "Guide"
}

scn:end_session()

return scn
