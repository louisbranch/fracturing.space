package maintenance

import (
	"context"
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// fakeEventStore implements storage.EventStore with canned events.
type fakeEventStore struct {
	events  map[string][]event.Event // keyed by campaignID
	listErr error
}

func (f *fakeEventStore) AppendEvent(_ context.Context, _ event.Event) (event.Event, error) {
	return event.Event{}, fmt.Errorf("not implemented")
}

func (f *fakeEventStore) GetEventByHash(_ context.Context, _ string) (event.Event, error) {
	return event.Event{}, fmt.Errorf("not implemented")
}

func (f *fakeEventStore) GetEventBySeq(_ context.Context, _ string, _ uint64) (event.Event, error) {
	return event.Event{}, fmt.Errorf("not implemented")
}

func (f *fakeEventStore) ListEvents(_ context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	all := f.events[campaignID]
	var result []event.Event
	for _, evt := range all {
		if evt.Seq > afterSeq {
			result = append(result, evt)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (f *fakeEventStore) ListEventsBySession(_ context.Context, _, _ string, _ uint64, _ int) ([]event.Event, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeEventStore) GetLatestEventSeq(_ context.Context, _ string) (uint64, error) {
	return 0, fmt.Errorf("not implemented")
}

func (f *fakeEventStore) ListEventsPage(_ context.Context, _ storage.ListEventsPageRequest) (storage.ListEventsPageResult, error) {
	return storage.ListEventsPageResult{}, fmt.Errorf("not implemented")
}

// fakeClosableEventStore wraps fakeEventStore with a closable interface.
type fakeClosableEventStore struct {
	fakeEventStore
	closeErr error
	closed   bool
}

func (f *fakeClosableEventStore) Close() error {
	f.closed = true
	return f.closeErr
}

// fakeProjectionStore satisfies storage.ProjectionStore with injectable function
// fields for methods exercised by tests. Methods without an injectable field
// return "not implemented".
type fakeProjectionStore struct {
	// Injectable function fields for methods used in replay/integrity paths.
	get                        func(ctx context.Context, id string) (campaign.Campaign, error)
	put                        func(ctx context.Context, c campaign.Campaign) error
	listCharacters             func(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.CharacterPage, error)
	getDaggerheartSnapshot     func(ctx context.Context, campaignID string) (storage.DaggerheartSnapshot, error)
	putDaggerheartSnapshot     func(ctx context.Context, snap storage.DaggerheartSnapshot) error
	getDaggerheartCharState    func(ctx context.Context, campaignID, characterID string) (storage.DaggerheartCharacterState, error)
	putDaggerheartCharState    func(ctx context.Context, state storage.DaggerheartCharacterState) error
	putDaggerheartCharProfile  func(ctx context.Context, profile storage.DaggerheartCharacterProfile) error
	putDaggerheartCountdown    func(ctx context.Context, countdown storage.DaggerheartCountdown) error
	getDaggerheartCountdown    func(ctx context.Context, campaignID, countdownID string) (storage.DaggerheartCountdown, error)
	deleteDaggerheartCountdown func(ctx context.Context, campaignID, countdownID string) error
	putDaggerheartAdversary    func(ctx context.Context, adversary storage.DaggerheartAdversary) error
	getDaggerheartAdversary    func(ctx context.Context, campaignID, adversaryID string) (storage.DaggerheartAdversary, error)
	deleteDaggerheartAdversary func(ctx context.Context, campaignID, adversaryID string) error
}

func (f *fakeProjectionStore) Put(ctx context.Context, c campaign.Campaign) error {
	if f.put != nil {
		return f.put(ctx, c)
	}
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) Get(ctx context.Context, id string) (campaign.Campaign, error) {
	if f.get != nil {
		return f.get(ctx, id)
	}
	return campaign.Campaign{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) List(context.Context, int, string) (storage.CampaignPage, error) {
	return storage.CampaignPage{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) PutParticipant(context.Context, participant.Participant) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetParticipant(context.Context, string, string) (participant.Participant, error) {
	return participant.Participant{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) DeleteParticipant(context.Context, string, string) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) ListParticipantsByCampaign(context.Context, string) ([]participant.Participant, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) ListParticipants(context.Context, string, int, string) (storage.ParticipantPage, error) {
	return storage.ParticipantPage{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) PutParticipantClaim(context.Context, string, string, string, time.Time) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetParticipantClaim(context.Context, string, string) (storage.ParticipantClaim, error) {
	return storage.ParticipantClaim{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) DeleteParticipantClaim(context.Context, string, string) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) PutInvite(context.Context, invite.Invite) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetInvite(context.Context, string) (invite.Invite, error) {
	return invite.Invite{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) ListInvites(context.Context, string, string, invite.Status, int, string) (storage.InvitePage, error) {
	return storage.InvitePage{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) ListPendingInvites(context.Context, string, int, string) (storage.InvitePage, error) {
	return storage.InvitePage{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) ListPendingInvitesForRecipient(context.Context, string, int, string) (storage.InvitePage, error) {
	return storage.InvitePage{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) UpdateInviteStatus(context.Context, string, invite.Status, time.Time) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) PutCharacter(context.Context, character.Character) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetCharacter(context.Context, string, string) (character.Character, error) {
	return character.Character{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) DeleteCharacter(context.Context, string, string) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) ListCharacters(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.CharacterPage, error) {
	if f.listCharacters != nil {
		return f.listCharacters(ctx, campaignID, pageSize, pageToken)
	}
	return storage.CharacterPage{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) PutDaggerheartCharacterProfile(ctx context.Context, profile storage.DaggerheartCharacterProfile) error {
	if f.putDaggerheartCharProfile != nil {
		return f.putDaggerheartCharProfile(ctx, profile)
	}
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetDaggerheartCharacterProfile(context.Context, string, string) (storage.DaggerheartCharacterProfile, error) {
	return storage.DaggerheartCharacterProfile{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) PutDaggerheartCharacterState(ctx context.Context, state storage.DaggerheartCharacterState) error {
	if f.putDaggerheartCharState != nil {
		return f.putDaggerheartCharState(ctx, state)
	}
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetDaggerheartCharacterState(ctx context.Context, campaignID, characterID string) (storage.DaggerheartCharacterState, error) {
	if f.getDaggerheartCharState != nil {
		return f.getDaggerheartCharState(ctx, campaignID, characterID)
	}
	return storage.DaggerheartCharacterState{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) PutDaggerheartSnapshot(ctx context.Context, snap storage.DaggerheartSnapshot) error {
	if f.putDaggerheartSnapshot != nil {
		return f.putDaggerheartSnapshot(ctx, snap)
	}
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetDaggerheartSnapshot(ctx context.Context, campaignID string) (storage.DaggerheartSnapshot, error) {
	if f.getDaggerheartSnapshot != nil {
		return f.getDaggerheartSnapshot(ctx, campaignID)
	}
	return storage.DaggerheartSnapshot{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) PutDaggerheartCountdown(ctx context.Context, countdown storage.DaggerheartCountdown) error {
	if f.putDaggerheartCountdown != nil {
		return f.putDaggerheartCountdown(ctx, countdown)
	}
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetDaggerheartCountdown(ctx context.Context, campaignID, countdownID string) (storage.DaggerheartCountdown, error) {
	if f.getDaggerheartCountdown != nil {
		return f.getDaggerheartCountdown(ctx, campaignID, countdownID)
	}
	return storage.DaggerheartCountdown{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) ListDaggerheartCountdowns(context.Context, string) ([]storage.DaggerheartCountdown, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) DeleteDaggerheartCountdown(ctx context.Context, campaignID, countdownID string) error {
	if f.deleteDaggerheartCountdown != nil {
		return f.deleteDaggerheartCountdown(ctx, campaignID, countdownID)
	}
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) PutDaggerheartAdversary(ctx context.Context, adversary storage.DaggerheartAdversary) error {
	if f.putDaggerheartAdversary != nil {
		return f.putDaggerheartAdversary(ctx, adversary)
	}
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetDaggerheartAdversary(ctx context.Context, campaignID, adversaryID string) (storage.DaggerheartAdversary, error) {
	if f.getDaggerheartAdversary != nil {
		return f.getDaggerheartAdversary(ctx, campaignID, adversaryID)
	}
	return storage.DaggerheartAdversary{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) ListDaggerheartAdversaries(context.Context, string, string) ([]storage.DaggerheartAdversary, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) DeleteDaggerheartAdversary(ctx context.Context, campaignID, adversaryID string) error {
	if f.deleteDaggerheartAdversary != nil {
		return f.deleteDaggerheartAdversary(ctx, campaignID, adversaryID)
	}
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) PutSession(context.Context, session.Session) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) EndSession(context.Context, string, string, time.Time) (session.Session, bool, error) {
	return session.Session{}, false, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetSession(context.Context, string, string) (session.Session, error) {
	return session.Session{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetActiveSession(context.Context, string) (session.Session, error) {
	return session.Session{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) ListSessions(context.Context, string, int, string) (storage.SessionPage, error) {
	return storage.SessionPage{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) PutSnapshot(context.Context, storage.Snapshot) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetSnapshot(context.Context, string, string) (storage.Snapshot, error) {
	return storage.Snapshot{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetLatestSnapshot(context.Context, string) (storage.Snapshot, error) {
	return storage.Snapshot{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) ListSnapshots(context.Context, string, int) ([]storage.Snapshot, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetCampaignForkMetadata(context.Context, string) (storage.ForkMetadata, error) {
	return storage.ForkMetadata{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) SetCampaignForkMetadata(context.Context, string, storage.ForkMetadata) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetGameStatistics(context.Context, *time.Time) (storage.GameStatistics, error) {
	return storage.GameStatistics{}, fmt.Errorf("not implemented")
}

// fakeClosableProjectionStore wraps fakeProjectionStore with a closable interface.
type fakeClosableProjectionStore struct {
	fakeProjectionStore
	closeErr error
	closed   bool
}

func (f *fakeClosableProjectionStore) Close() error {
	f.closed = true
	return f.closeErr
}
