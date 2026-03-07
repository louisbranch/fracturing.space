package game

import (
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/manifest"
)

// testStoresBuilder reduces repeated store construction in tests.
// Default stores (campaign, participant, event) are created eagerly
// since nearly every test needs them. Optional stores are added via
// with* methods.
//
// Usage:
//
//	ts := newTestStores()                              // campaign + participant + event
//	ts := newTestStores().withCharacter()               // + character + daggerheart
//	ts := newTestStores().withDomain(domain)            // + domain engine + write runtime
//	svc := NewCampaignService(ts.build())
type testStoresBuilder struct {
	Campaign    *fakeCampaignStore
	Participant *fakeParticipantStore
	Event       *fakeEventStore
	Character   *fakeCharacterStore
	Daggerheart *fakeDaggerheartStore
	Session     *fakeSessionStore
	SessionGate *fakeSessionGateStore
	Spotlight   *fakeSessionSpotlightStore
	Invite      *fakeInviteStore
	Fork        *fakeCampaignForkStore

	domain       Domain
	writeRuntime bool
}

// newTestStores creates a builder with the three stores that nearly every test
// needs: campaign, participant, and event.
func newTestStores() *testStoresBuilder {
	return &testStoresBuilder{
		Campaign:    newFakeCampaignStore(),
		Participant: newFakeParticipantStore(),
		Event:       newFakeEventStore(),
	}
}

func (b *testStoresBuilder) withCharacter() *testStoresBuilder {
	b.Character = newFakeCharacterStore()
	b.Daggerheart = newFakeDaggerheartStore()
	return b
}

func (b *testStoresBuilder) withSession() *testStoresBuilder {
	b.Session = newFakeSessionStore()
	return b
}

func (b *testStoresBuilder) withSessionGate() *testStoresBuilder {
	b.SessionGate = newFakeSessionGateStore()
	return b
}

func (b *testStoresBuilder) withSpotlight() *testStoresBuilder {
	b.Spotlight = newFakeSessionSpotlightStore()
	return b
}

func (b *testStoresBuilder) withInvite() *testStoresBuilder {
	b.Invite = newFakeInviteStore()
	return b
}

func (b *testStoresBuilder) withFork() *testStoresBuilder {
	b.Fork = newFakeCampaignForkStore()
	return b
}

func (b *testStoresBuilder) withDomain(d Domain) *testStoresBuilder {
	b.domain = d
	b.writeRuntime = true
	return b
}

func (b *testStoresBuilder) build() Stores {
	s := Stores{
		Campaign:    b.Campaign,
		Participant: b.Participant,
		Event:       b.Event,
	}
	if b.Character != nil {
		s.Character = b.Character
	}
	if b.Daggerheart != nil {
		s.SystemStores = systemmanifest.ProjectionStores{Daggerheart: b.Daggerheart}
	}
	if b.Session != nil {
		s.Session = b.Session
	}
	if b.SessionGate != nil {
		s.SessionGate = b.SessionGate
	}
	if b.Spotlight != nil {
		s.SessionSpotlight = b.Spotlight
	}
	if b.Invite != nil {
		s.Invite = b.Invite
	}
	if b.Fork != nil {
		s.CampaignFork = b.Fork
	}
	if b.domain != nil {
		s.Domain = b.domain
	}
	if b.writeRuntime {
		s.WriteRuntime = testRuntime
	}
	return s
}
