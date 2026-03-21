package maintenance

import (
	"context"
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// fakeEventStore implements storage.EventStore with canned events.
type fakeEventStore struct {
	events     map[string][]event.Event // keyed by campaignID
	latestSeqs map[string]uint64        // keyed by campaignID
	listErr    error
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

func (f *fakeEventStore) GetLatestEventSeq(_ context.Context, campaignID string) (uint64, error) {
	if f.latestSeqs != nil {
		seq, ok := f.latestSeqs[campaignID]
		if ok {
			return seq, nil
		}
	}
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
	get                          func(ctx context.Context, id string) (storage.CampaignRecord, error)
	put                          func(ctx context.Context, c storage.CampaignRecord) error
	listCharacters               func(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.CharacterPage, error)
	listCharactersByOwner        func(ctx context.Context, campaignID, participantID string) ([]storage.CharacterRecord, error)
	getDaggerheartSnapshot       func(ctx context.Context, campaignID string) (projectionstore.DaggerheartSnapshot, error)
	putDaggerheartSnapshot       func(ctx context.Context, snap projectionstore.DaggerheartSnapshot) error
	getDaggerheartCharState      func(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error)
	putDaggerheartCharState      func(ctx context.Context, state projectionstore.DaggerheartCharacterState) error
	putDaggerheartCharProfile    func(ctx context.Context, profile projectionstore.DaggerheartCharacterProfile) error
	putDaggerheartCountdown      func(ctx context.Context, countdown projectionstore.DaggerheartCountdown) error
	getDaggerheartCountdown      func(ctx context.Context, campaignID, countdownID string) (projectionstore.DaggerheartCountdown, error)
	deleteDaggerheartCountdown   func(ctx context.Context, campaignID, countdownID string) error
	putDaggerheartAdversary      func(ctx context.Context, adversary projectionstore.DaggerheartAdversary) error
	getDaggerheartAdversary      func(ctx context.Context, campaignID, adversaryID string) (projectionstore.DaggerheartAdversary, error)
	deleteDaggerheartAdversary   func(ctx context.Context, campaignID, adversaryID string) error
	putDaggerheartEnvironment    func(ctx context.Context, environmentEntity projectionstore.DaggerheartEnvironmentEntity) error
	getDaggerheartEnvironment    func(ctx context.Context, campaignID, environmentEntityID string) (projectionstore.DaggerheartEnvironmentEntity, error)
	deleteDaggerheartEnvironment func(ctx context.Context, campaignID, environmentEntityID string) error
	listProjectionWatermarks     func(ctx context.Context) ([]storage.ProjectionWatermark, error)
}

func (f *fakeProjectionStore) Put(ctx context.Context, c storage.CampaignRecord) error {
	if f.put != nil {
		return f.put(ctx, c)
	}
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) Get(ctx context.Context, id string) (storage.CampaignRecord, error) {
	if f.get != nil {
		return f.get(ctx, id)
	}
	return storage.CampaignRecord{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) List(context.Context, int, string) (storage.CampaignPage, error) {
	return storage.CampaignPage{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) PutParticipant(context.Context, storage.ParticipantRecord) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetParticipant(context.Context, string, string) (storage.ParticipantRecord, error) {
	return storage.ParticipantRecord{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) DeleteParticipant(context.Context, string, string) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) CountParticipants(context.Context, string) (int, error) {
	return 0, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) ListParticipantsByCampaign(context.Context, string) ([]storage.ParticipantRecord, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) ListCampaignIDsByUser(context.Context, string) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) ListCampaignIDsByParticipant(context.Context, string) ([]string, error) {
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

func (f *fakeProjectionStore) PutCharacter(context.Context, storage.CharacterRecord) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetCharacter(context.Context, string, string) (storage.CharacterRecord, error) {
	return storage.CharacterRecord{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) DeleteCharacter(context.Context, string, string) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) CountCharacters(context.Context, string) (int, error) {
	return 0, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) ListCharacters(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.CharacterPage, error) {
	if f.listCharacters != nil {
		return f.listCharacters(ctx, campaignID, pageSize, pageToken)
	}
	return storage.CharacterPage{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) ListCharactersByOwnerParticipant(ctx context.Context, campaignID, participantID string) ([]storage.CharacterRecord, error) {
	if f.listCharactersByOwner != nil {
		return f.listCharactersByOwner(ctx, campaignID, participantID)
	}
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) ListCharactersByControllerParticipant(context.Context, string, string) ([]storage.CharacterRecord, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) PutDaggerheartCharacterProfile(ctx context.Context, profile projectionstore.DaggerheartCharacterProfile) error {
	if f.putDaggerheartCharProfile != nil {
		return f.putDaggerheartCharProfile(ctx, profile)
	}
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetDaggerheartCharacterProfile(context.Context, string, string) (projectionstore.DaggerheartCharacterProfile, error) {
	return projectionstore.DaggerheartCharacterProfile{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) ListDaggerheartCharacterProfiles(context.Context, string, int, string) (projectionstore.DaggerheartCharacterProfilePage, error) {
	return projectionstore.DaggerheartCharacterProfilePage{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) DeleteDaggerheartCharacterProfile(context.Context, string, string) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) PutDaggerheartCharacterState(ctx context.Context, state projectionstore.DaggerheartCharacterState) error {
	if f.putDaggerheartCharState != nil {
		return f.putDaggerheartCharState(ctx, state)
	}
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetDaggerheartCharacterState(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error) {
	if f.getDaggerheartCharState != nil {
		return f.getDaggerheartCharState(ctx, campaignID, characterID)
	}
	return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) PutDaggerheartSnapshot(ctx context.Context, snap projectionstore.DaggerheartSnapshot) error {
	if f.putDaggerheartSnapshot != nil {
		return f.putDaggerheartSnapshot(ctx, snap)
	}
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetDaggerheartSnapshot(ctx context.Context, campaignID string) (projectionstore.DaggerheartSnapshot, error) {
	if f.getDaggerheartSnapshot != nil {
		return f.getDaggerheartSnapshot(ctx, campaignID)
	}
	return projectionstore.DaggerheartSnapshot{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) PutDaggerheartCountdown(ctx context.Context, countdown projectionstore.DaggerheartCountdown) error {
	if f.putDaggerheartCountdown != nil {
		return f.putDaggerheartCountdown(ctx, countdown)
	}
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetDaggerheartCountdown(ctx context.Context, campaignID, countdownID string) (projectionstore.DaggerheartCountdown, error) {
	if f.getDaggerheartCountdown != nil {
		return f.getDaggerheartCountdown(ctx, campaignID, countdownID)
	}
	return projectionstore.DaggerheartCountdown{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) ListDaggerheartCountdowns(context.Context, string) ([]projectionstore.DaggerheartCountdown, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) DeleteDaggerheartCountdown(ctx context.Context, campaignID, countdownID string) error {
	if f.deleteDaggerheartCountdown != nil {
		return f.deleteDaggerheartCountdown(ctx, campaignID, countdownID)
	}
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) PutDaggerheartAdversary(ctx context.Context, adversary projectionstore.DaggerheartAdversary) error {
	if f.putDaggerheartAdversary != nil {
		return f.putDaggerheartAdversary(ctx, adversary)
	}
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetDaggerheartAdversary(ctx context.Context, campaignID, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
	if f.getDaggerheartAdversary != nil {
		return f.getDaggerheartAdversary(ctx, campaignID, adversaryID)
	}
	return projectionstore.DaggerheartAdversary{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) ListDaggerheartAdversaries(context.Context, string, string) ([]projectionstore.DaggerheartAdversary, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) DeleteDaggerheartAdversary(ctx context.Context, campaignID, adversaryID string) error {
	if f.deleteDaggerheartAdversary != nil {
		return f.deleteDaggerheartAdversary(ctx, campaignID, adversaryID)
	}
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) PutDaggerheartEnvironmentEntity(ctx context.Context, environmentEntity projectionstore.DaggerheartEnvironmentEntity) error {
	if f.putDaggerheartEnvironment != nil {
		return f.putDaggerheartEnvironment(ctx, environmentEntity)
	}
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetDaggerheartEnvironmentEntity(ctx context.Context, campaignID, environmentEntityID string) (projectionstore.DaggerheartEnvironmentEntity, error) {
	if f.getDaggerheartEnvironment != nil {
		return f.getDaggerheartEnvironment(ctx, campaignID, environmentEntityID)
	}
	return projectionstore.DaggerheartEnvironmentEntity{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) ListDaggerheartEnvironmentEntities(context.Context, string, string, string) ([]projectionstore.DaggerheartEnvironmentEntity, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) DeleteDaggerheartEnvironmentEntity(ctx context.Context, campaignID, environmentEntityID string) error {
	if f.deleteDaggerheartEnvironment != nil {
		return f.deleteDaggerheartEnvironment(ctx, campaignID, environmentEntityID)
	}
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) PutSession(context.Context, storage.SessionRecord) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) EndSession(context.Context, string, string, time.Time) (storage.SessionRecord, bool, error) {
	return storage.SessionRecord{}, false, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetSession(context.Context, string, string) (storage.SessionRecord, error) {
	return storage.SessionRecord{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetActiveSession(context.Context, string) (storage.SessionRecord, error) {
	return storage.SessionRecord{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) CountSessions(context.Context, string) (int, error) {
	return 0, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) ListSessions(context.Context, string, int, string) (storage.SessionPage, error) {
	return storage.SessionPage{}, fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) PutSessionInteraction(context.Context, storage.SessionInteraction) error {
	return fmt.Errorf("not implemented")
}

func (f *fakeProjectionStore) GetSessionInteraction(context.Context, string, string) (storage.SessionInteraction, error) {
	return storage.SessionInteraction{}, fmt.Errorf("not implemented")
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

func (f *fakeProjectionStore) GetProjectionWatermark(context.Context, string) (storage.ProjectionWatermark, error) {
	return storage.ProjectionWatermark{}, storage.ErrNotFound
}

func (f *fakeProjectionStore) SaveProjectionWatermark(context.Context, storage.ProjectionWatermark) error {
	return nil
}

func (f *fakeProjectionStore) ListProjectionWatermarks(ctx context.Context) ([]storage.ProjectionWatermark, error) {
	if f.listProjectionWatermarks != nil {
		return f.listProjectionWatermarks(ctx)
	}
	return nil, nil
}

// SessionGateStore methods.
func (f *fakeProjectionStore) PutSessionGate(context.Context, storage.SessionGate) error {
	return fmt.Errorf("not implemented")
}
func (f *fakeProjectionStore) GetSessionGate(context.Context, string, string, string) (storage.SessionGate, error) {
	return storage.SessionGate{}, fmt.Errorf("not implemented")
}
func (f *fakeProjectionStore) GetOpenSessionGate(context.Context, string, string) (storage.SessionGate, error) {
	return storage.SessionGate{}, fmt.Errorf("not implemented")
}

// SessionSpotlightStore methods.
func (f *fakeProjectionStore) PutSessionSpotlight(context.Context, storage.SessionSpotlight) error {
	return fmt.Errorf("not implemented")
}
func (f *fakeProjectionStore) GetSessionSpotlight(context.Context, string, string) (storage.SessionSpotlight, error) {
	return storage.SessionSpotlight{}, fmt.Errorf("not implemented")
}
func (f *fakeProjectionStore) ClearSessionSpotlight(context.Context, string, string) error {
	return fmt.Errorf("not implemented")
}

// SceneStore methods.
func (f *fakeProjectionStore) PutScene(context.Context, storage.SceneRecord) error {
	return fmt.Errorf("not implemented")
}
func (f *fakeProjectionStore) EndScene(context.Context, string, string, time.Time) error {
	return fmt.Errorf("not implemented")
}
func (f *fakeProjectionStore) GetScene(context.Context, string, string) (storage.SceneRecord, error) {
	return storage.SceneRecord{}, fmt.Errorf("not implemented")
}
func (f *fakeProjectionStore) ListScenes(context.Context, string, string, int, string) (storage.ScenePage, error) {
	return storage.ScenePage{}, fmt.Errorf("not implemented")
}
func (f *fakeProjectionStore) ListOpenScenes(context.Context, string) ([]storage.SceneRecord, error) {
	return nil, fmt.Errorf("not implemented")
}
func (f *fakeProjectionStore) ListVisibleOpenScenesForCharacters(context.Context, string, string, []string) ([]storage.SceneRecord, error) {
	return nil, fmt.Errorf("not implemented")
}

// SceneCharacterStore methods.
func (f *fakeProjectionStore) PutSceneCharacter(context.Context, storage.SceneCharacterRecord) error {
	return fmt.Errorf("not implemented")
}
func (f *fakeProjectionStore) DeleteSceneCharacter(context.Context, string, string, string) error {
	return fmt.Errorf("not implemented")
}
func (f *fakeProjectionStore) ListSceneCharacters(context.Context, string, string) ([]storage.SceneCharacterRecord, error) {
	return nil, fmt.Errorf("not implemented")
}

// SceneGateStore methods.
func (f *fakeProjectionStore) PutSceneGate(context.Context, storage.SceneGate) error {
	return fmt.Errorf("not implemented")
}
func (f *fakeProjectionStore) GetSceneGate(context.Context, string, string, string) (storage.SceneGate, error) {
	return storage.SceneGate{}, fmt.Errorf("not implemented")
}
func (f *fakeProjectionStore) GetOpenSceneGate(context.Context, string, string) (storage.SceneGate, error) {
	return storage.SceneGate{}, fmt.Errorf("not implemented")
}

// SceneSpotlightStore methods.
func (f *fakeProjectionStore) PutSceneSpotlight(context.Context, storage.SceneSpotlight) error {
	return fmt.Errorf("not implemented")
}
func (f *fakeProjectionStore) GetSceneSpotlight(context.Context, string, string) (storage.SceneSpotlight, error) {
	return storage.SceneSpotlight{}, fmt.Errorf("not implemented")
}
func (f *fakeProjectionStore) ClearSceneSpotlight(context.Context, string, string) error {
	return fmt.Errorf("not implemented")
}

// SceneInteractionStore methods.
func (f *fakeProjectionStore) PutSceneInteraction(context.Context, storage.SceneInteraction) error {
	return fmt.Errorf("not implemented")
}
func (f *fakeProjectionStore) GetSceneInteraction(context.Context, string, string) (storage.SceneInteraction, error) {
	return storage.SceneInteraction{}, fmt.Errorf("not implemented")
}

// SceneGMInteractionStore methods.
func (f *fakeProjectionStore) PutSceneGMInteraction(context.Context, storage.SceneGMInteraction) error {
	return fmt.Errorf("not implemented")
}
func (f *fakeProjectionStore) ListSceneGMInteractions(context.Context, string, string) ([]storage.SceneGMInteraction, error) {
	return nil, fmt.Errorf("not implemented")
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
