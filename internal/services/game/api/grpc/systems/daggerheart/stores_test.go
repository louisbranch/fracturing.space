package daggerheart

import "testing"

func TestStoresApplier(t *testing.T) {
	s := Stores{
		Campaign:         &fakeCampaignStore{},
		Character:        &fakeCharacterStore{},
		Domain:           &fakeDomainEngine{},
		Session:          &fakeSessionStore{},
		SessionGate:      &fakeSessionGateStore{},
		SessionSpotlight: &fakeSessionSpotlightStore{},
		Daggerheart:      &fakeDaggerheartStore{},
		Event:            &fakeEventStore{},
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("validate stores: %v", err)
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
	if applier.Adapters == nil {
		t.Error("expected Adapters to be set")
	}
}
