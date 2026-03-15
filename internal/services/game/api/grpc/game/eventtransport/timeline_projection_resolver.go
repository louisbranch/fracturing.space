package eventtransport

import (
	"context"
	"errors"
	"fmt"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// timelineProjectionResolver caches projection reads to keep timeline responses consistent.
type timelineProjectionResolver struct {
	campaignStore    storage.CampaignStore
	participantStore storage.ParticipantStore
	characterStore   storage.CharacterStore
	sessionStore     storage.SessionStore

	campaignCache    map[string]storage.CampaignRecord
	participantCache map[string]storage.ParticipantRecord
	characterCache   map[string]storage.CharacterRecord
	sessionCache     map[string]storage.SessionRecord
}

type timelineProjectionStores struct {
	Campaign    storage.CampaignStore
	Participant storage.ParticipantStore
	Character   storage.CharacterStore
	Session     storage.SessionStore
}

// newTimelineProjectionResolver prepares a resolver for projection lookups.
func newTimelineProjectionResolver(stores timelineProjectionStores) *timelineProjectionResolver {
	return &timelineProjectionResolver{
		campaignStore:    stores.Campaign,
		participantStore: stores.Participant,
		characterStore:   stores.Character,
		sessionStore:     stores.Session,
		campaignCache:    make(map[string]storage.CampaignRecord),
		participantCache: make(map[string]storage.ParticipantRecord),
		characterCache:   make(map[string]storage.CharacterRecord),
		sessionCache:     make(map[string]storage.SessionRecord),
	}
}

// resolve maps an event to the icon and projection display used in the timeline.
func (r *timelineProjectionResolver) resolve(ctx context.Context, evt event.Event) (commonv1.IconId, *campaignv1.ProjectionDisplay, error) {
	domain := strings.TrimSpace(evt.EntityType)
	if domain == "" {
		domain = strings.TrimSpace(eventDomainFromType(evt.Type))
	}
	domain = strings.ToLower(domain)

	entityID := strings.TrimSpace(evt.EntityID)
	if entityID == "" {
		switch domain {
		case "campaign":
			entityID = strings.TrimSpace(string(evt.CampaignID))
		case "session":
			entityID = strings.TrimSpace(evt.SessionID.String())
		}
	}

	switch domain {
	case "campaign":
		entry, ok, err := r.lookupCampaign(ctx, entityID)
		if err != nil {
			return commonv1.IconId_ICON_ID_CAMPAIGN, nil, err
		}
		if !ok {
			return commonv1.IconId_ICON_ID_CAMPAIGN, nil, nil
		}
		return commonv1.IconId_ICON_ID_CAMPAIGN, campaignProjectionDisplay(entry), nil
	case "participant":
		entry, ok, err := r.lookupParticipant(ctx, string(evt.CampaignID), entityID)
		if err != nil {
			return commonv1.IconId_ICON_ID_PARTICIPANT, nil, err
		}
		if !ok {
			return commonv1.IconId_ICON_ID_PARTICIPANT, nil, nil
		}
		return commonv1.IconId_ICON_ID_PARTICIPANT, participantProjectionDisplay(entry), nil
	case "character":
		entry, ok, err := r.lookupCharacter(ctx, string(evt.CampaignID), entityID)
		if err != nil {
			return commonv1.IconId_ICON_ID_CHARACTER, nil, err
		}
		if !ok {
			return commonv1.IconId_ICON_ID_CHARACTER, nil, nil
		}
		return commonv1.IconId_ICON_ID_CHARACTER, characterProjectionDisplay(entry), nil
	case "session":
		entry, ok, err := r.lookupSession(ctx, string(evt.CampaignID), entityID)
		if err != nil {
			return commonv1.IconId_ICON_ID_SESSION, nil, err
		}
		if !ok {
			return commonv1.IconId_ICON_ID_SESSION, nil, nil
		}
		return commonv1.IconId_ICON_ID_SESSION, sessionProjectionDisplay(entry), nil
	default:
		return commonv1.IconId_ICON_ID_GENERIC, nil, nil
	}
}

// lookupCampaign fetches campaign projection data, using cache when available.
func (r *timelineProjectionResolver) lookupCampaign(ctx context.Context, campaignID string) (storage.CampaignRecord, bool, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return storage.CampaignRecord{}, false, nil
	}
	if r.campaignStore == nil {
		return storage.CampaignRecord{}, false, fmt.Errorf("campaign store is not configured")
	}
	if cached, ok := r.campaignCache[campaignID]; ok {
		return cached, true, nil
	}
	entry, err := r.campaignStore.Get(ctx, campaignID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return storage.CampaignRecord{}, false, nil
		}
		return storage.CampaignRecord{}, false, err
	}
	r.campaignCache[campaignID] = entry
	return entry, true, nil
}

// lookupParticipant fetches participant projection data, using cache when available.
func (r *timelineProjectionResolver) lookupParticipant(ctx context.Context, campaignID, participantID string) (storage.ParticipantRecord, bool, error) {
	campaignID = strings.TrimSpace(campaignID)
	participantID = strings.TrimSpace(participantID)
	if campaignID == "" || participantID == "" {
		return storage.ParticipantRecord{}, false, nil
	}
	if r.participantStore == nil {
		return storage.ParticipantRecord{}, false, fmt.Errorf("participant store is not configured")
	}
	key := campaignID + ":" + participantID
	if cached, ok := r.participantCache[key]; ok {
		return cached, true, nil
	}
	entry, err := r.participantStore.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return storage.ParticipantRecord{}, false, nil
		}
		return storage.ParticipantRecord{}, false, err
	}
	r.participantCache[key] = entry
	return entry, true, nil
}

// lookupCharacter fetches character projection data, using cache when available.
func (r *timelineProjectionResolver) lookupCharacter(ctx context.Context, campaignID, characterID string) (storage.CharacterRecord, bool, error) {
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return storage.CharacterRecord{}, false, nil
	}
	if r.characterStore == nil {
		return storage.CharacterRecord{}, false, fmt.Errorf("character store is not configured")
	}
	key := campaignID + ":" + characterID
	if cached, ok := r.characterCache[key]; ok {
		return cached, true, nil
	}
	entry, err := r.characterStore.GetCharacter(ctx, campaignID, characterID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return storage.CharacterRecord{}, false, nil
		}
		return storage.CharacterRecord{}, false, err
	}
	r.characterCache[key] = entry
	return entry, true, nil
}

// lookupSession fetches session projection data, using cache when available.
func (r *timelineProjectionResolver) lookupSession(ctx context.Context, campaignID, sessionID string) (storage.SessionRecord, bool, error) {
	campaignID = strings.TrimSpace(campaignID)
	sessionID = strings.TrimSpace(sessionID)
	if campaignID == "" || sessionID == "" {
		return storage.SessionRecord{}, false, nil
	}
	if r.sessionStore == nil {
		return storage.SessionRecord{}, false, fmt.Errorf("session store is not configured")
	}
	key := campaignID + ":" + sessionID
	if cached, ok := r.sessionCache[key]; ok {
		return cached, true, nil
	}
	entry, err := r.sessionStore.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return storage.SessionRecord{}, false, nil
		}
		return storage.SessionRecord{}, false, err
	}
	r.sessionCache[key] = entry
	return entry, true, nil
}
