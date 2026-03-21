local scn = Scenario.new("interaction_simulated_ai_gm_ooc_resume")
local dh = scn:system("DAGGERHEART")

-- Exercise the OOC overlay, then resume scene play and take a real mechanic action.
scn:campaign{
  name = "Interaction Simulated AI GM OOC Resume",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "interaction"
}

-- Create the GM seat plus two players who can pause and resume the same scene.
scn:participant({name = "Guide", role = "GM"})
scn:participant({name = "Rhea"}):character({name = "Aria"})
scn:participant({name = "Bryn"}):character({name = "Corin"})

-- Open the vault scene and start a shared player phase.
scn:start_session("Vault Run")
scn:create_scene{
  name = "Sealed Vault",
  description = "An old vault door hums with warding magic and a narrow seam of light.",
  characters = {"Aria", "Corin"}
}
scn:interaction_set_gm_authority({participant = "Guide"})
scn:interaction_set_active_scene({scene = "Sealed Vault"})
scn:interaction_start_player_phase{
  scene = "Sealed Vault",
  interaction = {
    title = "Vault Seam",
    beats = {
      {type = "prompt", text = "The ward flickers when either of you approaches the seam. What do you do?"},
    },
  },
  characters = {"Aria", "Corin"}
}

-- Pause for an out-of-character clarification and make the overlay authoritative.
scn:interaction_pause_ooc({as = "Rhea", reason = "Clarify how the ward reacts to touch."})
scn:interaction_post_ooc({as = "Rhea", body = "Does the ward flare if Aria uses a tool instead of bare hands?"})
scn:interaction_post_ooc({as = "Guide", body = "The ward reacts to contact with the seam, not to sight or sound."})
scn:interaction_post_ooc({as = "Bryn", body = "Then Corin can coach from a safe distance."})
scn:interaction_ready_ooc({as = "Rhea"})
scn:interaction_ready_ooc({as = "Bryn"})
scn:interaction_expect{
  phase_status = "PLAYERS",
  ooc_open = true,
  ooc_requested_by = "Rhea",
  ooc_interrupted_scene = "Sealed Vault",
  ooc_interrupted_phase_status = "PLAYERS",
  ooc_ready = {"Rhea", "Bryn"},
  ooc_posts = {
    {participant = "Rhea", body = "Does the ward flare if Aria uses a tool instead of bare hands?"},
    {participant = "Guide", body = "The ward reacts to contact with the seam, not to sight or sound."},
    {participant = "Bryn", body = "Then Corin can coach from a safe distance."}
  }
}

-- Resume the scene, let the GM reframe the beat, and take a real roll after the pause.
scn:interaction_resume_ooc()
scn:interaction_expect{
  phase_status = "PLAYERS",
  ooc_open = false,
  ooc_resolution_pending = true,
  gm_authority = "Guide"
}
scn:interaction_resolve_interrupted_phase{
  as = "Guide",
  interaction = {
    title = "Understand The Trigger",
    beats = {
      {type = "fiction", text = "The ward's pulse sharpens once you understand the seam is the real trigger."},
      {type = "prompt", text = "Aria, now that you know the seam is the trigger, how do you pry it open?"},
    },
  },
  characters = {"Aria"}
}
scn:interaction_post{
  as = "Rhea",
  summary = "Aria wedges a hooked tool into the seam and eases the ward apart without touching it.",
  characters = {"Aria"}
}
dh:action_roll{
  as = "Rhea",
  actor = "Aria",
  trait = "instinct",
  difficulty = 14,
  outcome = "hope"
}
scn:interaction_yield({as = "Rhea"})
scn:interaction_expect{
  phase_status = "GM_REVIEW",
  slots = {
    {participant = "Rhea", summary = "Aria wedges a hooked tool into the seam and eases the ward apart without touching it.", characters = {"Aria"}, yielded = true, review_status = "UNDER_REVIEW"}
  },
  gm_authority = "Guide",
  ooc_open = false,
  ooc_resolution_pending = false
}

scn:end_session()

return scn
