package scenario

import (
	"context"
	"log"
	"os"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func unimplemented(method string) error {
	return status.Errorf(codes.Unimplemented, "%s not implemented", method)
}

type fakeCampaignClient struct {
	create func(context.Context, *gamev1.CreateCampaignRequest, ...grpc.CallOption) (*gamev1.CreateCampaignResponse, error)
}

func (f *fakeCampaignClient) CreateCampaign(ctx context.Context, in *gamev1.CreateCampaignRequest, opts ...grpc.CallOption) (*gamev1.CreateCampaignResponse, error) {
	if f.create != nil {
		return f.create(ctx, in, opts...)
	}
	return nil, unimplemented("CreateCampaign")
}

func (f *fakeCampaignClient) ListCampaigns(context.Context, *gamev1.ListCampaignsRequest, ...grpc.CallOption) (*gamev1.ListCampaignsResponse, error) {
	return nil, unimplemented("ListCampaigns")
}

func (f *fakeCampaignClient) GetCampaign(context.Context, *gamev1.GetCampaignRequest, ...grpc.CallOption) (*gamev1.GetCampaignResponse, error) {
	return nil, unimplemented("GetCampaign")
}

func (f *fakeCampaignClient) EndCampaign(context.Context, *gamev1.EndCampaignRequest, ...grpc.CallOption) (*gamev1.EndCampaignResponse, error) {
	return nil, unimplemented("EndCampaign")
}

func (f *fakeCampaignClient) ArchiveCampaign(context.Context, *gamev1.ArchiveCampaignRequest, ...grpc.CallOption) (*gamev1.ArchiveCampaignResponse, error) {
	return nil, unimplemented("ArchiveCampaign")
}

func (f *fakeCampaignClient) RestoreCampaign(context.Context, *gamev1.RestoreCampaignRequest, ...grpc.CallOption) (*gamev1.RestoreCampaignResponse, error) {
	return nil, unimplemented("RestoreCampaign")
}

type fakeParticipantClient struct {
	create func(context.Context, *gamev1.CreateParticipantRequest, ...grpc.CallOption) (*gamev1.CreateParticipantResponse, error)
}

func (f *fakeParticipantClient) CreateParticipant(ctx context.Context, in *gamev1.CreateParticipantRequest, opts ...grpc.CallOption) (*gamev1.CreateParticipantResponse, error) {
	if f.create != nil {
		return f.create(ctx, in, opts...)
	}
	return nil, unimplemented("CreateParticipant")
}

func (f *fakeParticipantClient) UpdateParticipant(context.Context, *gamev1.UpdateParticipantRequest, ...grpc.CallOption) (*gamev1.UpdateParticipantResponse, error) {
	return nil, unimplemented("UpdateParticipant")
}

func (f *fakeParticipantClient) DeleteParticipant(context.Context, *gamev1.DeleteParticipantRequest, ...grpc.CallOption) (*gamev1.DeleteParticipantResponse, error) {
	return nil, unimplemented("DeleteParticipant")
}

func (f *fakeParticipantClient) ListParticipants(context.Context, *gamev1.ListParticipantsRequest, ...grpc.CallOption) (*gamev1.ListParticipantsResponse, error) {
	return nil, unimplemented("ListParticipants")
}

func (f *fakeParticipantClient) GetParticipant(context.Context, *gamev1.GetParticipantRequest, ...grpc.CallOption) (*gamev1.GetParticipantResponse, error) {
	return nil, unimplemented("GetParticipant")
}

type fakeCharacterClient struct {
	create            func(context.Context, *gamev1.CreateCharacterRequest, ...grpc.CallOption) (*gamev1.CreateCharacterResponse, error)
	setDefaultControl func(context.Context, *gamev1.SetDefaultControlRequest, ...grpc.CallOption) (*gamev1.SetDefaultControlResponse, error)
	patchProfile      func(context.Context, *gamev1.PatchCharacterProfileRequest, ...grpc.CallOption) (*gamev1.PatchCharacterProfileResponse, error)
	patchState        func(context.Context, *gamev1.PatchCharacterStateRequest, ...grpc.CallOption) (*gamev1.PatchCharacterStateResponse, error)
	getSheet          func(context.Context, *gamev1.GetCharacterSheetRequest, ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error)
}

func (f *fakeCharacterClient) CreateCharacter(ctx context.Context, in *gamev1.CreateCharacterRequest, opts ...grpc.CallOption) (*gamev1.CreateCharacterResponse, error) {
	if f.create != nil {
		return f.create(ctx, in, opts...)
	}
	return nil, unimplemented("CreateCharacter")
}

func (f *fakeCharacterClient) UpdateCharacter(context.Context, *gamev1.UpdateCharacterRequest, ...grpc.CallOption) (*gamev1.UpdateCharacterResponse, error) {
	return nil, unimplemented("UpdateCharacter")
}

func (f *fakeCharacterClient) DeleteCharacter(context.Context, *gamev1.DeleteCharacterRequest, ...grpc.CallOption) (*gamev1.DeleteCharacterResponse, error) {
	return nil, unimplemented("DeleteCharacter")
}

func (f *fakeCharacterClient) ListCharacters(context.Context, *gamev1.ListCharactersRequest, ...grpc.CallOption) (*gamev1.ListCharactersResponse, error) {
	return nil, unimplemented("ListCharacters")
}

func (f *fakeCharacterClient) SetDefaultControl(ctx context.Context, in *gamev1.SetDefaultControlRequest, opts ...grpc.CallOption) (*gamev1.SetDefaultControlResponse, error) {
	if f.setDefaultControl != nil {
		return f.setDefaultControl(ctx, in, opts...)
	}
	return nil, unimplemented("SetDefaultControl")
}

func (f *fakeCharacterClient) GetCharacterSheet(ctx context.Context, in *gamev1.GetCharacterSheetRequest, opts ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
	if f.getSheet != nil {
		return f.getSheet(ctx, in, opts...)
	}
	return nil, unimplemented("GetCharacterSheet")
}

func (f *fakeCharacterClient) PatchCharacterProfile(ctx context.Context, in *gamev1.PatchCharacterProfileRequest, opts ...grpc.CallOption) (*gamev1.PatchCharacterProfileResponse, error) {
	if f.patchProfile != nil {
		return f.patchProfile(ctx, in, opts...)
	}
	return nil, unimplemented("PatchCharacterProfile")
}

func (f *fakeCharacterClient) PatchCharacterState(ctx context.Context, in *gamev1.PatchCharacterStateRequest, opts ...grpc.CallOption) (*gamev1.PatchCharacterStateResponse, error) {
	if f.patchState != nil {
		return f.patchState(ctx, in, opts...)
	}
	return nil, unimplemented("PatchCharacterState")
}

type fakeEventClient struct {
	seq int64
}

func (f *fakeEventClient) AppendEvent(context.Context, *gamev1.AppendEventRequest, ...grpc.CallOption) (*gamev1.AppendEventResponse, error) {
	return nil, unimplemented("AppendEvent")
}

func (f *fakeEventClient) ListEvents(context.Context, *gamev1.ListEventsRequest, ...grpc.CallOption) (*gamev1.ListEventsResponse, error) {
	f.seq++
	return &gamev1.ListEventsResponse{
		Events: []*gamev1.Event{{Seq: uint64(f.seq)}},
	}, nil
}

type fakeSnapshotClient struct {
	patchState     func(context.Context, *gamev1.PatchCharacterStateRequest, ...grpc.CallOption) (*gamev1.PatchCharacterStateResponse, error)
	getSnapshot    func(context.Context, *gamev1.GetSnapshotRequest, ...grpc.CallOption) (*gamev1.GetSnapshotResponse, error)
	updateSnapshot func(context.Context, *gamev1.UpdateSnapshotStateRequest, ...grpc.CallOption) (*gamev1.UpdateSnapshotStateResponse, error)
}

func (f *fakeSnapshotClient) GetSnapshot(ctx context.Context, in *gamev1.GetSnapshotRequest, opts ...grpc.CallOption) (*gamev1.GetSnapshotResponse, error) {
	if f.getSnapshot != nil {
		return f.getSnapshot(ctx, in, opts...)
	}
	return nil, unimplemented("GetSnapshot")
}

func (f *fakeSnapshotClient) PatchCharacterState(ctx context.Context, in *gamev1.PatchCharacterStateRequest, opts ...grpc.CallOption) (*gamev1.PatchCharacterStateResponse, error) {
	if f.patchState != nil {
		return f.patchState(ctx, in, opts...)
	}
	return nil, unimplemented("PatchCharacterState")
}

func (f *fakeSnapshotClient) UpdateSnapshotState(ctx context.Context, in *gamev1.UpdateSnapshotStateRequest, opts ...grpc.CallOption) (*gamev1.UpdateSnapshotStateResponse, error) {
	if f.updateSnapshot != nil {
		return f.updateSnapshot(ctx, in, opts...)
	}
	return nil, unimplemented("UpdateSnapshotState")
}

// fakeSessionClient implements gamev1.SessionServiceClient for testing.
type fakeSessionClient struct {
	startSession   func(context.Context, *gamev1.StartSessionRequest, ...grpc.CallOption) (*gamev1.StartSessionResponse, error)
	endSession     func(context.Context, *gamev1.EndSessionRequest, ...grpc.CallOption) (*gamev1.EndSessionResponse, error)
	resolveGate    func(context.Context, *gamev1.ResolveSessionGateRequest, ...grpc.CallOption) (*gamev1.ResolveSessionGateResponse, error)
	setSpotlight   func(context.Context, *gamev1.SetSessionSpotlightRequest, ...grpc.CallOption) (*gamev1.SetSessionSpotlightResponse, error)
	clearSpotlight func(context.Context, *gamev1.ClearSessionSpotlightRequest, ...grpc.CallOption) (*gamev1.ClearSessionSpotlightResponse, error)
	getSpotlight   func(context.Context, *gamev1.GetSessionSpotlightRequest, ...grpc.CallOption) (*gamev1.GetSessionSpotlightResponse, error)
}

func (f *fakeSessionClient) StartSession(ctx context.Context, in *gamev1.StartSessionRequest, opts ...grpc.CallOption) (*gamev1.StartSessionResponse, error) {
	if f.startSession != nil {
		return f.startSession(ctx, in, opts...)
	}
	return nil, unimplemented("StartSession")
}

func (f *fakeSessionClient) ListSessions(context.Context, *gamev1.ListSessionsRequest, ...grpc.CallOption) (*gamev1.ListSessionsResponse, error) {
	return nil, unimplemented("ListSessions")
}

func (f *fakeSessionClient) GetSession(context.Context, *gamev1.GetSessionRequest, ...grpc.CallOption) (*gamev1.GetSessionResponse, error) {
	return nil, unimplemented("GetSession")
}

func (f *fakeSessionClient) EndSession(ctx context.Context, in *gamev1.EndSessionRequest, opts ...grpc.CallOption) (*gamev1.EndSessionResponse, error) {
	if f.endSession != nil {
		return f.endSession(ctx, in, opts...)
	}
	return nil, unimplemented("EndSession")
}

func (f *fakeSessionClient) OpenSessionGate(context.Context, *gamev1.OpenSessionGateRequest, ...grpc.CallOption) (*gamev1.OpenSessionGateResponse, error) {
	return nil, unimplemented("OpenSessionGate")
}

func (f *fakeSessionClient) ResolveSessionGate(ctx context.Context, in *gamev1.ResolveSessionGateRequest, opts ...grpc.CallOption) (*gamev1.ResolveSessionGateResponse, error) {
	if f.resolveGate != nil {
		return f.resolveGate(ctx, in, opts...)
	}
	return nil, unimplemented("ResolveSessionGate")
}

func (f *fakeSessionClient) AbandonSessionGate(context.Context, *gamev1.AbandonSessionGateRequest, ...grpc.CallOption) (*gamev1.AbandonSessionGateResponse, error) {
	return nil, unimplemented("AbandonSessionGate")
}

func (f *fakeSessionClient) GetSessionSpotlight(ctx context.Context, in *gamev1.GetSessionSpotlightRequest, opts ...grpc.CallOption) (*gamev1.GetSessionSpotlightResponse, error) {
	if f.getSpotlight != nil {
		return f.getSpotlight(ctx, in, opts...)
	}
	return nil, unimplemented("GetSessionSpotlight")
}

func (f *fakeSessionClient) SetSessionSpotlight(ctx context.Context, in *gamev1.SetSessionSpotlightRequest, opts ...grpc.CallOption) (*gamev1.SetSessionSpotlightResponse, error) {
	if f.setSpotlight != nil {
		return f.setSpotlight(ctx, in, opts...)
	}
	return nil, unimplemented("SetSessionSpotlight")
}

func (f *fakeSessionClient) ClearSessionSpotlight(ctx context.Context, in *gamev1.ClearSessionSpotlightRequest, opts ...grpc.CallOption) (*gamev1.ClearSessionSpotlightResponse, error) {
	if f.clearSpotlight != nil {
		return f.clearSpotlight(ctx, in, opts...)
	}
	return nil, unimplemented("ClearSessionSpotlight")
}

// fakeDaggerheartClient implements daggerheartv1.DaggerheartServiceClient for testing.
type fakeDaggerheartClient struct {
	actionRoll                  func(context.Context, *daggerheartv1.ActionRollRequest, ...grpc.CallOption) (*daggerheartv1.ActionRollResponse, error)
	rollDice                    func(context.Context, *daggerheartv1.RollDiceRequest, ...grpc.CallOption) (*daggerheartv1.RollDiceResponse, error)
	applyDamage                 func(context.Context, *daggerheartv1.DaggerheartApplyDamageRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyDamageResponse, error)
	applyAdversaryDamage        func(context.Context, *daggerheartv1.DaggerheartApplyAdversaryDamageRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyAdversaryDamageResponse, error)
	applyRest                   func(context.Context, *daggerheartv1.DaggerheartApplyRestRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyRestResponse, error)
	applyDowntimeMove           func(context.Context, *daggerheartv1.DaggerheartApplyDowntimeMoveRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyDowntimeMoveResponse, error)
	swapLoadout                 func(context.Context, *daggerheartv1.DaggerheartSwapLoadoutRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartSwapLoadoutResponse, error)
	applyDeathMove              func(context.Context, *daggerheartv1.DaggerheartApplyDeathMoveRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyDeathMoveResponse, error)
	applyConditions             func(context.Context, *daggerheartv1.DaggerheartApplyConditionsRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyConditionsResponse, error)
	applyAdversaryConditions    func(context.Context, *daggerheartv1.DaggerheartApplyAdversaryConditionsRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyAdversaryConditionsResponse, error)
	applyGmMove                 func(context.Context, *daggerheartv1.DaggerheartApplyGmMoveRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyGmMoveResponse, error)
	createCountdown             func(context.Context, *daggerheartv1.DaggerheartCreateCountdownRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartCreateCountdownResponse, error)
	updateCountdown             func(context.Context, *daggerheartv1.DaggerheartUpdateCountdownRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartUpdateCountdownResponse, error)
	deleteCountdown             func(context.Context, *daggerheartv1.DaggerheartDeleteCountdownRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartDeleteCountdownResponse, error)
	createAdversary             func(context.Context, *daggerheartv1.DaggerheartCreateAdversaryRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartCreateAdversaryResponse, error)
	updateAdversary             func(context.Context, *daggerheartv1.DaggerheartUpdateAdversaryRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartUpdateAdversaryResponse, error)
	deleteAdversary             func(context.Context, *daggerheartv1.DaggerheartDeleteAdversaryRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartDeleteAdversaryResponse, error)
	getAdversary                func(context.Context, *daggerheartv1.DaggerheartGetAdversaryRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartGetAdversaryResponse, error)
	listAdversaries             func(context.Context, *daggerheartv1.DaggerheartListAdversariesRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartListAdversariesResponse, error)
	resolveBlazeOfGlory         func(context.Context, *daggerheartv1.DaggerheartResolveBlazeOfGloryRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartResolveBlazeOfGloryResponse, error)
	sessionActionRoll           func(context.Context, *daggerheartv1.SessionActionRollRequest, ...grpc.CallOption) (*daggerheartv1.SessionActionRollResponse, error)
	sessionDamageRoll           func(context.Context, *daggerheartv1.SessionDamageRollRequest, ...grpc.CallOption) (*daggerheartv1.SessionDamageRollResponse, error)
	sessionAttackFlow           func(context.Context, *daggerheartv1.SessionAttackFlowRequest, ...grpc.CallOption) (*daggerheartv1.SessionAttackFlowResponse, error)
	sessionReactionFlow         func(context.Context, *daggerheartv1.SessionReactionFlowRequest, ...grpc.CallOption) (*daggerheartv1.SessionReactionFlowResponse, error)
	sessionAdversaryAttackRoll  func(context.Context, *daggerheartv1.SessionAdversaryAttackRollRequest, ...grpc.CallOption) (*daggerheartv1.SessionAdversaryAttackRollResponse, error)
	sessionAdversaryAttackFlow  func(context.Context, *daggerheartv1.SessionAdversaryAttackFlowRequest, ...grpc.CallOption) (*daggerheartv1.SessionAdversaryAttackFlowResponse, error)
	sessionGroupActionFlow      func(context.Context, *daggerheartv1.SessionGroupActionFlowRequest, ...grpc.CallOption) (*daggerheartv1.SessionGroupActionFlowResponse, error)
	sessionTagTeamFlow          func(context.Context, *daggerheartv1.SessionTagTeamFlowRequest, ...grpc.CallOption) (*daggerheartv1.SessionTagTeamFlowResponse, error)
	applyRollOutcome            func(context.Context, *daggerheartv1.ApplyRollOutcomeRequest, ...grpc.CallOption) (*daggerheartv1.ApplyRollOutcomeResponse, error)
	applyAttackOutcome          func(context.Context, *daggerheartv1.DaggerheartApplyAttackOutcomeRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyAttackOutcomeResponse, error)
	applyAdversaryAttackOutcome func(context.Context, *daggerheartv1.DaggerheartApplyAdversaryAttackOutcomeRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyAdversaryAttackOutcomeResponse, error)
	applyReactionOutcome        func(context.Context, *daggerheartv1.DaggerheartApplyReactionOutcomeRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyReactionOutcomeResponse, error)
}

func (f *fakeDaggerheartClient) ActionRoll(ctx context.Context, in *daggerheartv1.ActionRollRequest, opts ...grpc.CallOption) (*daggerheartv1.ActionRollResponse, error) {
	if f.actionRoll != nil {
		return f.actionRoll(ctx, in, opts...)
	}
	return nil, unimplemented("ActionRoll")
}

func (f *fakeDaggerheartClient) DualityOutcome(context.Context, *daggerheartv1.DualityOutcomeRequest, ...grpc.CallOption) (*daggerheartv1.DualityOutcomeResponse, error) {
	return nil, unimplemented("DualityOutcome")
}

func (f *fakeDaggerheartClient) DualityExplain(context.Context, *daggerheartv1.DualityExplainRequest, ...grpc.CallOption) (*daggerheartv1.DualityExplainResponse, error) {
	return nil, unimplemented("DualityExplain")
}

func (f *fakeDaggerheartClient) DualityProbability(context.Context, *daggerheartv1.DualityProbabilityRequest, ...grpc.CallOption) (*daggerheartv1.DualityProbabilityResponse, error) {
	return nil, unimplemented("DualityProbability")
}

func (f *fakeDaggerheartClient) RulesVersion(context.Context, *daggerheartv1.RulesVersionRequest, ...grpc.CallOption) (*daggerheartv1.RulesVersionResponse, error) {
	return nil, unimplemented("RulesVersion")
}

func (f *fakeDaggerheartClient) RollDice(ctx context.Context, in *daggerheartv1.RollDiceRequest, opts ...grpc.CallOption) (*daggerheartv1.RollDiceResponse, error) {
	if f.rollDice != nil {
		return f.rollDice(ctx, in, opts...)
	}
	return nil, unimplemented("RollDice")
}

func (f *fakeDaggerheartClient) ApplyDamage(ctx context.Context, in *daggerheartv1.DaggerheartApplyDamageRequest, opts ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyDamageResponse, error) {
	if f.applyDamage != nil {
		return f.applyDamage(ctx, in, opts...)
	}
	return nil, unimplemented("ApplyDamage")
}

func (f *fakeDaggerheartClient) ApplyAdversaryDamage(ctx context.Context, in *daggerheartv1.DaggerheartApplyAdversaryDamageRequest, opts ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyAdversaryDamageResponse, error) {
	if f.applyAdversaryDamage != nil {
		return f.applyAdversaryDamage(ctx, in, opts...)
	}
	return nil, unimplemented("ApplyAdversaryDamage")
}

func (f *fakeDaggerheartClient) ApplyRest(ctx context.Context, in *daggerheartv1.DaggerheartApplyRestRequest, opts ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyRestResponse, error) {
	if f.applyRest != nil {
		return f.applyRest(ctx, in, opts...)
	}
	return nil, unimplemented("ApplyRest")
}

func (f *fakeDaggerheartClient) ApplyDowntimeMove(ctx context.Context, in *daggerheartv1.DaggerheartApplyDowntimeMoveRequest, opts ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyDowntimeMoveResponse, error) {
	if f.applyDowntimeMove != nil {
		return f.applyDowntimeMove(ctx, in, opts...)
	}
	return nil, unimplemented("ApplyDowntimeMove")
}

func (f *fakeDaggerheartClient) SwapLoadout(ctx context.Context, in *daggerheartv1.DaggerheartSwapLoadoutRequest, opts ...grpc.CallOption) (*daggerheartv1.DaggerheartSwapLoadoutResponse, error) {
	if f.swapLoadout != nil {
		return f.swapLoadout(ctx, in, opts...)
	}
	return nil, unimplemented("SwapLoadout")
}

func (f *fakeDaggerheartClient) ApplyDeathMove(ctx context.Context, in *daggerheartv1.DaggerheartApplyDeathMoveRequest, opts ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyDeathMoveResponse, error) {
	if f.applyDeathMove != nil {
		return f.applyDeathMove(ctx, in, opts...)
	}
	return nil, unimplemented("ApplyDeathMove")
}

func (f *fakeDaggerheartClient) ApplyConditions(ctx context.Context, in *daggerheartv1.DaggerheartApplyConditionsRequest, opts ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyConditionsResponse, error) {
	if f.applyConditions != nil {
		return f.applyConditions(ctx, in, opts...)
	}
	return nil, unimplemented("ApplyConditions")
}

func (f *fakeDaggerheartClient) ApplyAdversaryConditions(ctx context.Context, in *daggerheartv1.DaggerheartApplyAdversaryConditionsRequest, opts ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyAdversaryConditionsResponse, error) {
	if f.applyAdversaryConditions != nil {
		return f.applyAdversaryConditions(ctx, in, opts...)
	}
	return nil, unimplemented("ApplyAdversaryConditions")
}

func (f *fakeDaggerheartClient) ApplyGmMove(ctx context.Context, in *daggerheartv1.DaggerheartApplyGmMoveRequest, opts ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyGmMoveResponse, error) {
	if f.applyGmMove != nil {
		return f.applyGmMove(ctx, in, opts...)
	}
	return nil, unimplemented("ApplyGmMove")
}

func (f *fakeDaggerheartClient) CreateCountdown(ctx context.Context, in *daggerheartv1.DaggerheartCreateCountdownRequest, opts ...grpc.CallOption) (*daggerheartv1.DaggerheartCreateCountdownResponse, error) {
	if f.createCountdown != nil {
		return f.createCountdown(ctx, in, opts...)
	}
	return nil, unimplemented("CreateCountdown")
}

func (f *fakeDaggerheartClient) UpdateCountdown(ctx context.Context, in *daggerheartv1.DaggerheartUpdateCountdownRequest, opts ...grpc.CallOption) (*daggerheartv1.DaggerheartUpdateCountdownResponse, error) {
	if f.updateCountdown != nil {
		return f.updateCountdown(ctx, in, opts...)
	}
	return nil, unimplemented("UpdateCountdown")
}

func (f *fakeDaggerheartClient) DeleteCountdown(ctx context.Context, in *daggerheartv1.DaggerheartDeleteCountdownRequest, opts ...grpc.CallOption) (*daggerheartv1.DaggerheartDeleteCountdownResponse, error) {
	if f.deleteCountdown != nil {
		return f.deleteCountdown(ctx, in, opts...)
	}
	return nil, unimplemented("DeleteCountdown")
}

func (f *fakeDaggerheartClient) CreateAdversary(ctx context.Context, in *daggerheartv1.DaggerheartCreateAdversaryRequest, opts ...grpc.CallOption) (*daggerheartv1.DaggerheartCreateAdversaryResponse, error) {
	if f.createAdversary != nil {
		return f.createAdversary(ctx, in, opts...)
	}
	return nil, unimplemented("CreateAdversary")
}

func (f *fakeDaggerheartClient) UpdateAdversary(ctx context.Context, in *daggerheartv1.DaggerheartUpdateAdversaryRequest, opts ...grpc.CallOption) (*daggerheartv1.DaggerheartUpdateAdversaryResponse, error) {
	if f.updateAdversary != nil {
		return f.updateAdversary(ctx, in, opts...)
	}
	return nil, unimplemented("UpdateAdversary")
}

func (f *fakeDaggerheartClient) DeleteAdversary(ctx context.Context, in *daggerheartv1.DaggerheartDeleteAdversaryRequest, opts ...grpc.CallOption) (*daggerheartv1.DaggerheartDeleteAdversaryResponse, error) {
	if f.deleteAdversary != nil {
		return f.deleteAdversary(ctx, in, opts...)
	}
	return nil, unimplemented("DeleteAdversary")
}

func (f *fakeDaggerheartClient) GetAdversary(ctx context.Context, in *daggerheartv1.DaggerheartGetAdversaryRequest, opts ...grpc.CallOption) (*daggerheartv1.DaggerheartGetAdversaryResponse, error) {
	if f.getAdversary != nil {
		return f.getAdversary(ctx, in, opts...)
	}
	return nil, unimplemented("GetAdversary")
}

func (f *fakeDaggerheartClient) ListAdversaries(ctx context.Context, in *daggerheartv1.DaggerheartListAdversariesRequest, opts ...grpc.CallOption) (*daggerheartv1.DaggerheartListAdversariesResponse, error) {
	if f.listAdversaries != nil {
		return f.listAdversaries(ctx, in, opts...)
	}
	return nil, unimplemented("ListAdversaries")
}

func (f *fakeDaggerheartClient) ResolveBlazeOfGlory(ctx context.Context, in *daggerheartv1.DaggerheartResolveBlazeOfGloryRequest, opts ...grpc.CallOption) (*daggerheartv1.DaggerheartResolveBlazeOfGloryResponse, error) {
	if f.resolveBlazeOfGlory != nil {
		return f.resolveBlazeOfGlory(ctx, in, opts...)
	}
	return nil, unimplemented("ResolveBlazeOfGlory")
}

func (f *fakeDaggerheartClient) SessionActionRoll(ctx context.Context, in *daggerheartv1.SessionActionRollRequest, opts ...grpc.CallOption) (*daggerheartv1.SessionActionRollResponse, error) {
	if f.sessionActionRoll != nil {
		return f.sessionActionRoll(ctx, in, opts...)
	}
	return nil, unimplemented("SessionActionRoll")
}

func (f *fakeDaggerheartClient) SessionDamageRoll(ctx context.Context, in *daggerheartv1.SessionDamageRollRequest, opts ...grpc.CallOption) (*daggerheartv1.SessionDamageRollResponse, error) {
	if f.sessionDamageRoll != nil {
		return f.sessionDamageRoll(ctx, in, opts...)
	}
	return nil, unimplemented("SessionDamageRoll")
}

func (f *fakeDaggerheartClient) SessionAttackFlow(ctx context.Context, in *daggerheartv1.SessionAttackFlowRequest, opts ...grpc.CallOption) (*daggerheartv1.SessionAttackFlowResponse, error) {
	if f.sessionAttackFlow != nil {
		return f.sessionAttackFlow(ctx, in, opts...)
	}
	return nil, unimplemented("SessionAttackFlow")
}

func (f *fakeDaggerheartClient) SessionReactionFlow(ctx context.Context, in *daggerheartv1.SessionReactionFlowRequest, opts ...grpc.CallOption) (*daggerheartv1.SessionReactionFlowResponse, error) {
	if f.sessionReactionFlow != nil {
		return f.sessionReactionFlow(ctx, in, opts...)
	}
	return nil, unimplemented("SessionReactionFlow")
}

func (f *fakeDaggerheartClient) SessionAdversaryAttackRoll(ctx context.Context, in *daggerheartv1.SessionAdversaryAttackRollRequest, opts ...grpc.CallOption) (*daggerheartv1.SessionAdversaryAttackRollResponse, error) {
	if f.sessionAdversaryAttackRoll != nil {
		return f.sessionAdversaryAttackRoll(ctx, in, opts...)
	}
	return nil, unimplemented("SessionAdversaryAttackRoll")
}

func (f *fakeDaggerheartClient) SessionAdversaryActionCheck(context.Context, *daggerheartv1.SessionAdversaryActionCheckRequest, ...grpc.CallOption) (*daggerheartv1.SessionAdversaryActionCheckResponse, error) {
	return nil, unimplemented("SessionAdversaryActionCheck")
}

func (f *fakeDaggerheartClient) SessionAdversaryAttackFlow(ctx context.Context, in *daggerheartv1.SessionAdversaryAttackFlowRequest, opts ...grpc.CallOption) (*daggerheartv1.SessionAdversaryAttackFlowResponse, error) {
	if f.sessionAdversaryAttackFlow != nil {
		return f.sessionAdversaryAttackFlow(ctx, in, opts...)
	}
	return nil, unimplemented("SessionAdversaryAttackFlow")
}

func (f *fakeDaggerheartClient) SessionGroupActionFlow(ctx context.Context, in *daggerheartv1.SessionGroupActionFlowRequest, opts ...grpc.CallOption) (*daggerheartv1.SessionGroupActionFlowResponse, error) {
	if f.sessionGroupActionFlow != nil {
		return f.sessionGroupActionFlow(ctx, in, opts...)
	}
	return nil, unimplemented("SessionGroupActionFlow")
}

func (f *fakeDaggerheartClient) SessionTagTeamFlow(ctx context.Context, in *daggerheartv1.SessionTagTeamFlowRequest, opts ...grpc.CallOption) (*daggerheartv1.SessionTagTeamFlowResponse, error) {
	if f.sessionTagTeamFlow != nil {
		return f.sessionTagTeamFlow(ctx, in, opts...)
	}
	return nil, unimplemented("SessionTagTeamFlow")
}

func (f *fakeDaggerheartClient) ApplyRollOutcome(ctx context.Context, in *daggerheartv1.ApplyRollOutcomeRequest, opts ...grpc.CallOption) (*daggerheartv1.ApplyRollOutcomeResponse, error) {
	if f.applyRollOutcome != nil {
		return f.applyRollOutcome(ctx, in, opts...)
	}
	return nil, unimplemented("ApplyRollOutcome")
}

func (f *fakeDaggerheartClient) ApplyAttackOutcome(ctx context.Context, in *daggerheartv1.DaggerheartApplyAttackOutcomeRequest, opts ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyAttackOutcomeResponse, error) {
	if f.applyAttackOutcome != nil {
		return f.applyAttackOutcome(ctx, in, opts...)
	}
	return nil, unimplemented("ApplyAttackOutcome")
}

func (f *fakeDaggerheartClient) ApplyAdversaryAttackOutcome(ctx context.Context, in *daggerheartv1.DaggerheartApplyAdversaryAttackOutcomeRequest, opts ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyAdversaryAttackOutcomeResponse, error) {
	if f.applyAdversaryAttackOutcome != nil {
		return f.applyAdversaryAttackOutcome(ctx, in, opts...)
	}
	return nil, unimplemented("ApplyAdversaryAttackOutcome")
}

func (f *fakeDaggerheartClient) ApplyReactionOutcome(ctx context.Context, in *daggerheartv1.DaggerheartApplyReactionOutcomeRequest, opts ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyReactionOutcomeResponse, error) {
	if f.applyReactionOutcome != nil {
		return f.applyReactionOutcome(ctx, in, opts...)
	}
	return nil, unimplemented("ApplyReactionOutcome")
}

// newTestRunner creates a Runner with pre-built clients for testing.
// Production code should use NewRunner which dials the real gRPC server.
func newTestRunner(env scenarioEnv, opts ...func(*Runner)) *Runner {
	r := &Runner{
		env:        env,
		assertions: Assertions{Mode: AssertionStrict},
		logger:     log.New(os.Stderr, "", 0),
		timeout:    5 * time.Second,
		auth:       NewMockAuth(),
	}
	r.userID = r.auth.CreateUser("Test Runner")
	for _, opt := range opts {
		opt(r)
	}
	return r
}
