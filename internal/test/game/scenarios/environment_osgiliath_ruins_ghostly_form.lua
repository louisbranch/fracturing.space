local scene = Scenario.new("environment_osgiliath_ruins_ghostly_form")

-- Capture ghostly adversaries' resistance and phasing.
scene:campaign{
  name = "Environment Osgiliath Ruins Ghostly Form",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
scene:adversary("Barrow Wight")

-- Ghosts resist physical damage and phase through walls by marking Stress.
scene:start_session("Ghostly Form")

-- Physical resistance modeling remains unresolved in this fixture.
scene:adversary_update{ target = "Barrow Wight", stress_delta = 1, notes = "phase_through_walls" }
scene:adversary_attack{ actor = "Barrow Wight", target = "Frodo", difficulty = 0, damage_type = "physical" }

scene:end_session()

return scene
