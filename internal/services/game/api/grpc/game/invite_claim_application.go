package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/joingrant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a inviteApplication) ClaimInvite(ctx context.Context, campaignID string, in *campaignv1.ClaimInviteRequest) (storage.InviteRecord, storage.ParticipantRecord, error) {
	inviteID, err := validate.RequiredID(in.GetInviteId(), "invite id")
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}
	if _, err := validate.RequiredID(in.GetJoinGrant(), "join grant"); err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}
	userID, err := validate.RequiredID(grpcmeta.UserIDFromContext(ctx), "user id")
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}

	inv, err := a.stores.Invite.GetInvite(ctx, inviteID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}
	if inv.CampaignID != campaignID {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.InvalidArgument, "invite campaign does not match")
	}
	if recipient := strings.TrimSpace(inv.RecipientUserID); recipient != "" && recipient != userID {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.PermissionDenied, "invite recipient does not match")
	}
	campaignRecord, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}

	claims, err := a.joinGrantVerifier.Validate(in.GetJoinGrant(), joingrant.Expectation{
		CampaignID: campaignID,
		InviteID:   inv.ID,
		UserID:     userID,
	})
	if err != nil {
		if errors.Is(err, joingrant.ErrVerifierNotConfigured) {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("join grant validation is not configured", err)
		}
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, err
		}
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("validate join grant", err)
	}
	if a.stores.ClaimIndex != nil {
		claim, err := a.stores.ClaimIndex.GetParticipantClaim(ctx, campaignID, userID)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("load participant claim", err)
		}
		if err == nil && claim.ParticipantID != inv.ParticipantID {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, apperrors.WithMetadata(
				apperrors.CodeParticipantUserAlreadyClaimed,
				"participant user already claimed",
				map[string]string{
					"CampaignID": campaignID,
					"UserID":     userID,
				},
			)
		}
	}
	claimEvent, err := findInviteClaimByJTI(ctx, a.stores.Event, campaignID, claims.JWTID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("lookup join grant", err)
	}
	if claimEvent != nil {
		var payload invite.ClaimPayload
		if err := json.Unmarshal(claimEvent.PayloadJSON, &payload); err != nil {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("decode prior claim", err)
		}
		if payload.InviteID != ids.InviteID(inv.ID) || payload.UserID != ids.UserID(userID) {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, apperrors.New(apperrors.CodeInviteJoinGrantUsed, "join grant already used")
		}
		updatedInvite, err := a.stores.Invite.GetInvite(ctx, inv.ID)
		if err != nil {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("load invite", err)
		}
		updatedParticipant, err := a.stores.Participant.GetParticipant(ctx, campaignID, inv.ParticipantID)
		if err != nil {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("load participant", err)
		}
		return updatedInvite, updatedParticipant, nil
	}
	if inv.Status == invite.StatusClaimed {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, "invite already claimed")
	}
	if inv.Status == invite.StatusDeclined {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, "invite already declined")
	}
	if inv.Status == invite.StatusRevoked {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, "invite already revoked")
	}

	seat, err := a.stores.Participant.GetParticipant(ctx, campaignID, inv.ParticipantID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}
	inviteState, err := loadInviteReplayState(ctx, a.stores.Event, campaignID, inv.ID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("load invite state", err)
	}
	if !inviteState.Created {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.NotFound, "invite not found")
	}
	if inviteState.Status == string(invite.StatusClaimed) {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, "invite already claimed")
	}
	if inviteState.Status == string(invite.StatusDeclined) {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, "invite already declined")
	}
	if inviteState.Status == string(invite.StatusRevoked) {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, "invite already revoked")
	}

	participantStates, err := loadCampaignParticipantReplayStates(ctx, a.stores.Event, campaignID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("load participant bindings", err)
	}
	if claimedParticipantID, ok := findClaimedParticipantForUser(participantStates, userID); ok && claimedParticipantID != inv.ParticipantID {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, apperrors.WithMetadata(
			apperrors.CodeParticipantUserAlreadyClaimed,
			"participant user already claimed",
			map[string]string{
				"CampaignID": campaignID,
				"UserID":     userID,
			},
		)
	}

	seatState, err := loadParticipantReplayState(ctx, a.stores.Event, campaignID, inv.ParticipantID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("load participant state", err)
	}
	if participantStateHasActiveUserBinding(seatState) {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, "participant already claimed")
	}

	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	payload := participant.BindPayload{
		ParticipantID: ids.ParticipantID(seat.ID),
		UserID:        ids.UserID(userID),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("encode participant payload", err)
	}

	actorID, actorType := resolveCommandActor(ctx)
	bindCmd := commandbuild.Core(commandbuild.CoreInput{
		CampaignID:   campaignID,
		Type:         commandTypeParticipantBind,
		ActorType:    actorType,
		ActorID:      actorID,
		RequestID:    requestID,
		InvocationID: invocationID,
		EntityType:   "participant",
		EntityID:     seat.ID,
		PayloadJSON:  payloadJSON,
	})
	bindEvents, err := decideParticipantBindEvents(seatState, bindCmd, a.clock)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}

	claimPayload := invite.ClaimPayload{
		InviteID:      ids.InviteID(inv.ID),
		ParticipantID: ids.ParticipantID(inv.ParticipantID),
		UserID:        ids.UserID(userID),
		JWTID:         claims.JWTID,
	}
	claimJSON, err := json.Marshal(claimPayload)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("encode invite payload", err)
	}
	inviteActorType := command.ActorTypeSystem
	if actorID != "" {
		inviteActorType = command.ActorTypeParticipant
	}
	claimCmd := commandbuild.Core(commandbuild.CoreInput{
		CampaignID:   campaignID,
		Type:         commandTypeInviteClaim,
		ActorType:    inviteActorType,
		ActorID:      actorID,
		RequestID:    requestID,
		InvocationID: invocationID,
		EntityType:   "invite",
		EntityID:     inv.ID,
		PayloadJSON:  claimJSON,
	})
	claimEvents, err := decideInviteClaimEvents(inviteState, claimCmd, a.clock)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}
	if _, err := appendAndApplyInviteClaimEvents(ctx, a.stores.Event, a.applier, append(bindEvents, claimEvents...)); err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}

	updatedInvite, err := a.stores.Invite.GetInvite(ctx, inv.ID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("load invite", err)
	}
	a.applyInviteClaimProfileSnapshot(ctx, campaignID, seat.ID, userID, requestID, invocationID, actorID, inviteActorType)
	updatedParticipant, err := a.stores.Participant.GetParticipant(ctx, campaignID, seat.ID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("load participant", err)
	}

	return updatedInvite, updatedParticipant, nil
}

func (a inviteApplication) applyInviteClaimProfileSnapshot(
	ctx context.Context,
	campaignID string,
	participantID string,
	userID string,
	requestID string,
	invocationID string,
	actorID string,
	actorType command.ActorType,
) {
	snapshot := loadSocialProfileSnapshot(ctx, a.stores.Social, userID)
	fields := map[string]string{}
	if snapshot.Name != "" {
		fields["name"] = snapshot.Name
	}
	if snapshot.Pronouns != "" {
		fields["pronouns"] = snapshot.Pronouns
	}
	if snapshot.AvatarSetID != "" {
		fields["avatar_set_id"] = snapshot.AvatarSetID
	}
	if snapshot.AvatarAssetID != "" {
		fields["avatar_asset_id"] = snapshot.AvatarAssetID
	}
	if len(fields) == 0 {
		return
	}

	payloadJSON, err := json.Marshal(participant.UpdatePayload{
		ParticipantID: ids.ParticipantID(participantID),
		Fields:        fields,
	})
	if err != nil {
		return
	}

	_, _ = executeAndApplyDomainCommand(
		ctx,
		a.write,
		a.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeParticipantUpdate,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    requestID,
			InvocationID: invocationID,
			EntityType:   "participant",
			EntityID:     participantID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.Options{
			ApplyErr: domainApplyErrorWithCodePreserve("apply participant event"),
		},
	)
}

// loadInviteReplayState replays the invite aggregate from the event journal so
// claim validation does not depend on potentially stale invite projections.
func loadInviteReplayState(ctx context.Context, store storage.EventStore, campaignID string, inviteID string) (invite.State, error) {
	return replayEntityState(
		ctx,
		store,
		campaignID,
		"invite",
		inviteID,
		invite.State{},
		invite.Fold,
	)
}

// loadParticipantReplayState replays one participant aggregate so occupancy
// checks can rely on authoritative history instead of projection lag windows.
func loadParticipantReplayState(ctx context.Context, store storage.EventStore, campaignID string, participantID string) (participant.State, error) {
	return replayEntityState(
		ctx,
		store,
		campaignID,
		"participant",
		participantID,
		participant.State{},
		participant.Fold,
	)
}

// loadCampaignParticipantReplayStates folds all participant events for a
// campaign so claim-time user-binding checks stay authoritative even if the
// claim index projection is missing or behind.
func loadCampaignParticipantReplayStates(ctx context.Context, store storage.EventStore, campaignID string) (map[string]participant.State, error) {
	if store == nil {
		return nil, fmt.Errorf("event store is not configured")
	}
	states := make(map[string]participant.State)
	var cursor uint64
	for {
		page, err := store.ListEventsPage(ctx, storage.ListEventsPageRequest{
			CampaignID: campaignID,
			PageSize:   200,
			CursorSeq:  cursor,
			CursorDir:  "fwd",
			Filter: storage.EventQueryFilter{
				EntityType: "participant",
			},
		})
		if err != nil {
			return nil, err
		}
		for _, evt := range page.Events {
			state := states[evt.EntityID]
			state, err = participant.Fold(state, evt)
			if err != nil {
				return nil, err
			}
			states[evt.EntityID] = state
		}
		if !page.HasNextPage || len(page.Events) == 0 {
			return states, nil
		}
		cursor = page.Events[len(page.Events)-1].Seq
	}
}

// findClaimedParticipantForUser scans replayed participant state to answer the
// conflict question directly from authoritative campaign history.
func findClaimedParticipantForUser(states map[string]participant.State, userID string) (string, bool) {
	normalizedUserID := strings.TrimSpace(userID)
	if normalizedUserID == "" {
		return "", false
	}
	for participantID, state := range states {
		if !participantStateHasActiveUserBinding(state) {
			continue
		}
		if strings.TrimSpace(state.UserID.String()) != normalizedUserID {
			continue
		}
		return participantID, true
	}
	return "", false
}

// participantStateHasActiveUserBinding narrows replayed participant history to
// active seat ownership so past leaves and explicit unbinds do not block claim.
func participantStateHasActiveUserBinding(state participant.State) bool {
	if !state.Joined || state.Left {
		return false
	}
	return strings.TrimSpace(state.UserID.String()) != ""
}

// replayEntityState pages the event journal with entity filters and folds the
// matching events into domain state for write-path preflight checks.
func replayEntityState[T any](
	ctx context.Context,
	store storage.EventStore,
	campaignID string,
	entityType string,
	entityID string,
	state T,
	fold func(T, event.Event) (T, error),
) (T, error) {
	if store == nil {
		return state, fmt.Errorf("event store is not configured")
	}
	var cursor uint64
	for {
		page, err := store.ListEventsPage(ctx, storage.ListEventsPageRequest{
			CampaignID: campaignID,
			PageSize:   200,
			CursorSeq:  cursor,
			CursorDir:  "fwd",
			Filter: storage.EventQueryFilter{
				EntityType: entityType,
				EntityID:   entityID,
			},
		})
		if err != nil {
			return state, err
		}
		for _, evt := range page.Events {
			state, err = fold(state, evt)
			if err != nil {
				return state, err
			}
		}
		if !page.HasNextPage || len(page.Events) == 0 {
			return state, nil
		}
		cursor = page.Events[len(page.Events)-1].Seq
	}
}

// decideParticipantBindEvents derives the seat-binding event before any write
// so claim orchestration can append both accepted events atomically.
func decideParticipantBindEvents(state participant.State, cmd command.Command, now func() time.Time) ([]event.Event, error) {
	decision := participant.Decide(state, cmd, now)
	if len(decision.Rejections) > 0 {
		rejection := decision.Rejections[0]
		return nil, domainDecisionError(rejection.Code, rejection.Message)
	}
	if len(decision.Events) == 0 {
		return nil, status.Error(codes.Internal, "participant bind emitted no events")
	}
	return decision.Events, nil
}

// decideInviteClaimEvents derives the invite-claim event alongside the bind
// event so the journal sees one atomic claim transition.
func decideInviteClaimEvents(state invite.State, cmd command.Command, now func() time.Time) ([]event.Event, error) {
	decision := invite.Decide(state, cmd, now)
	if len(decision.Rejections) > 0 {
		rejection := decision.Rejections[0]
		return nil, domainDecisionError(rejection.Code, rejection.Message)
	}
	if len(decision.Events) == 0 {
		return nil, status.Error(codes.Internal, "invite claim emitted no events")
	}
	return decision.Events, nil
}

// appendAndApplyInviteClaimEvents persists the accepted claim batch atomically,
// then updates inline projections in the stored event order.
func appendAndApplyInviteClaimEvents(
	ctx context.Context,
	store storage.EventStore,
	applier projection.Applier,
	events []event.Event,
) ([]event.Event, error) {
	stored, err := NewJournalAdapter(store).BatchAppend(ctx, events)
	if err != nil {
		return nil, grpcerror.Internal("append invite claim events", err)
	}
	applyErr := domainApplyErrorWithCodePreserve("apply invite claim event")
	for _, evt := range stored {
		if err := applier.Apply(ctx, evt); err != nil {
			return nil, applyErr(err)
		}
	}
	return stored, nil
}

// domainDecisionError keeps manual decider flows aligned with the standard
// write-path rejection mapping used by ExecuteAndApply.
func domainDecisionError(code string, message string) error {
	options := domainwrite.Options{}
	grpcerror.NormalizeDomainWriteOptions(&options, grpcerror.NormalizeDomainWriteOptionsConfig{})
	return options.RejectErr(code, message)
}

func findInviteClaimByJTI(ctx context.Context, store storage.EventStore, campaignID, jti string) (*event.Event, error) {
	if strings.TrimSpace(jti) == "" {
		return nil, nil
	}
	if store == nil {
		return nil, fmt.Errorf("event store is not configured")
	}

	var cursor uint64
	for {
		page, err := store.ListEventsPage(ctx, storage.ListEventsPageRequest{
			CampaignID: campaignID,
			PageSize:   200,
			CursorSeq:  cursor,
			CursorDir:  "fwd",
			Descending: false,
			Filter: storage.EventQueryFilter{
				EventType: string(eventTypeInviteClaimed),
			},
		})
		if err != nil {
			return nil, err
		}
		for i := range page.Events {
			evt := page.Events[i]
			if evt.Type != eventTypeInviteClaimed {
				continue
			}
			var payload invite.ClaimPayload
			if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
				return nil, err
			}
			if payload.JWTID == jti {
				return &evt, nil
			}
		}
		if !page.HasNextPage || len(page.Events) == 0 {
			return nil, nil
		}
		cursor = page.Events[len(page.Events)-1].Seq
	}
}
