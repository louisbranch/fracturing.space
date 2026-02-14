package daggerheart

import "testing"

func TestStoresApplier(t *testing.T) {
	s := Stores{
		Campaign:         &fakeCampaignStore{},
		Character:        &fakeCharacterStore{},
		Session:          &fakeSessionStore{},
		SessionGate:      &fakeSessionGateStore{},
		SessionSpotlight: &fakeSessionSpotlightStore{},
		Daggerheart:      &fakeDaggerheartStore{},
		Event:            &fakeEventStore{},
	}
	applier := s.Applier()

	if applier.Campaign == nil {
		t.Error("expected Campaign to be set")
	}
	if applier.Character == nil {
		t.Error("expected Character to be set")
	}
	if applier.Session == nil {
		t.Error("expected Session to be set")
	}
	if applier.SessionGate == nil {
		t.Error("expected SessionGate to be set")
	}
	if applier.SessionSpotlight == nil {
		t.Error("expected SessionSpotlight to be set")
	}
	if applier.Daggerheart == nil {
		t.Error("expected Daggerheart to be set")
	}
}
