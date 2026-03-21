local scn = Scenario.new("interaction_active_scene_switch")

-- Switching the active scene interrupts the open player phase on the prior
-- scene and makes the new scene the only actionable one.
scn:campaign{
  name = "Interaction Active Scene Switch",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "interaction"
}

scn:participant({name = "Guide", role = "GM"})
scn:participant({name = "Rhea"}):character({name = "Aria"})
scn:participant({name = "Bryn"}):character({name = "Corin"})

-- Create two simultaneous scenes for a split-party cutaway.
scn:start_session("Split Party")
scn:create_scene{
  name = "North Gate",
  description = "Aria waits on the wall above the city gate.",
  characters = {"Aria"}
}
scn:create_scene{
  name = "South Tunnel",
  description = "Corin crouches in the drainage tunnel beneath the keep.",
  characters = {"Corin"}
}
scn:interaction_set_gm_authority({participant = "Guide"})

-- Open a player phase on the first active scene.
scn:interaction_set_active_scene({scene = "North Gate"})
scn:interaction_start_player_phase{
  scene = "North Gate",
  interaction = {
    title = "Changing Watch",
    beats = {
      {type = "prompt", text = "The gate guards are changing watch. Aria, what do you do?"},
    },
  },
  characters = {"Aria"}
}
scn:interaction_post{
  as = "Rhea",
  summary = "Aria counts the guard rotation and marks the safest blind angle.",
  characters = {"Aria"}
}
scn:interaction_expect{
  active_scene = "North Gate",
  phase_status = "PLAYERS",
  slots = {
    {participant = "Rhea", summary = "Aria counts the guard rotation and marks the safest blind angle.", characters = {"Aria"}}
  }
}

-- Cut away to the second active scene and verify the first phase is interrupted.
scn:interaction_set_active_scene({scene = "South Tunnel"})
scn:interaction_expect{
  active_scene = "South Tunnel",
  phase_status = "GM",
  acting_participants = {},
  acting_characters = {},
  slots = {},
  gm_authority = "Guide"
}

-- The new active scene is now the only one that can open a player phase.
scn:interaction_start_player_phase{
  scene = "South Tunnel",
  interaction = {
    title = "Tunnel Opening",
    beats = {
      {type = "prompt", text = "Corin, the tunnel opens beneath the keep. What do you do next?"},
    },
  },
  characters = {"Corin"}
}
scn:interaction_post{
  as = "Bryn",
  summary = "Corin slides into the runoff and listens for footsteps above the grate.",
  characters = {"Corin"},
  yield = true
}
scn:interaction_expect{
  active_scene = "South Tunnel",
  phase_status = "GM_REVIEW",
  slots = {
    {participant = "Bryn", summary = "Corin slides into the runoff and listens for footsteps above the grate.", characters = {"Corin"}, yielded = true, review_status = "UNDER_REVIEW"}
  },
  gm_authority = "Guide"
}

scn:end_session()

return scn
