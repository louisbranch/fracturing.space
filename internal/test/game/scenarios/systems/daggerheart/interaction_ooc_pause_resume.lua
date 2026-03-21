local scn = Scenario.new("interaction_ooc_pause_resume")

-- Players pause the scene for OOC discussion, coordinate with the GM, then
-- resume the same scene and continue the beat.
scn:campaign{
  name = "Interaction OOC Pause Resume",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "interaction"
}

scn:participant({name = "Guide", role = "GM"})
scn:participant({name = "Rhea"}):character({name = "Aria"})
scn:participant({name = "Bryn"}):character({name = "Corin"})

-- Open the shared scene and pause it for a ruling discussion.
scn:start_session("Vault")
scn:create_scene{
  name = "Sealed Vault",
  description = "The vault door hums with old warding magic.",
  characters = {"Aria", "Corin"}
}
scn:interaction_set_gm_authority({participant = "Guide"})
scn:interaction_set_active_scene({scene = "Sealed Vault"})
scn:interaction_start_player_phase{
  scene = "Sealed Vault",
  frame_text = "The ward crackles when either of you nears the seam. What do you do?",
  characters = {"Aria", "Corin"}
}
scn:interaction_pause_ooc({as = "Rhea", reason = "Clarify how the ward reacts to tools."})
scn:interaction_post_ooc({as = "Rhea", body = "Does the ward react to metal touching the seam or only skin?"})
scn:interaction_post_ooc({as = "Guide", body = "The ward reacts to any contact with the seam itself."})
scn:interaction_post_ooc({as = "Bryn", body = "Then Corin should guide from a step back."})
scn:interaction_ready_ooc({as = "Rhea"})
scn:interaction_ready_ooc({as = "Bryn"})
scn:interaction_expect{
  active_scene = "Sealed Vault",
  phase_status = "PLAYERS",
  ooc_open = true,
  ooc_requested_by = "Rhea",
  ooc_interrupted_scene = "Sealed Vault",
  ooc_interrupted_phase_status = "PLAYERS",
  ooc_ready = {"Rhea", "Bryn"},
  ooc_posts = {
    {participant = "Rhea", body = "Does the ward react to metal touching the seam or only skin?"},
    {participant = "Guide", body = "The ward reacts to any contact with the seam itself."},
    {participant = "Bryn", body = "Then Corin should guide from a step back."}
  }
}

-- Resume the same scene, reopen the player phase, and complete the beat.
scn:interaction_resume_ooc()
scn:interaction_expect{
  active_scene = "Sealed Vault",
  phase_status = "PLAYERS",
  ooc_open = false,
  ooc_resolution_pending = true,
  ooc_ready = {},
  gm_authority = "Guide"
}
scn:interaction_resolve_interrupted_phase{
  as = "Guide",
  gm_output_text = "The ward's crackle changes once you understand the seam is the trigger.",
  frame_text = "Aria, now that you know the seam is the trigger, how do you pry the vault open?",
  characters = {"Aria", "Corin"}
}
scn:interaction_post{
  as = "Rhea",
  summary = "Aria wedges a hooked tool into the seam without touching it.",
  characters = {"Aria"},
  yield = true
}
scn:interaction_post{
  as = "Bryn",
  summary = "Corin keeps clear of the ward and talks Aria through the leverage point.",
  characters = {"Corin"},
  yield = true
}
scn:interaction_expect{
  phase_status = "GM_REVIEW",
  ooc_open = false,
  slots = {
    {participant = "Bryn", summary = "Corin keeps clear of the ward and talks Aria through the leverage point.", characters = {"Corin"}, yielded = true, review_status = "UNDER_REVIEW"},
    {participant = "Rhea", summary = "Aria wedges a hooked tool into the seam without touching it.", characters = {"Aria"}, yielded = true, review_status = "UNDER_REVIEW"}
  },
  gm_authority = "Guide",
  ooc_resolution_pending = false
}

scn:end_session()

return scn
