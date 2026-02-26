package web

import (
	"context"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/campaign"
)

func buildCampaignFeatureClientDependencies(h *handler, d *campaignfeature.AppCampaignDependencies) {
	d.ListCampaigns = func(ctx context.Context, req *statev1.ListCampaignsRequest) (*statev1.ListCampaignsResponse, error) {
		return h.campaignClient.ListCampaigns(ctx, req)
	}
	d.CreateCampaign = func(ctx context.Context, req *statev1.CreateCampaignRequest) (*statev1.CreateCampaignResponse, error) {
		return h.campaignClient.CreateCampaign(ctx, req)
	}
	d.GetCampaign = func(ctx context.Context, req *statev1.GetCampaignRequest) (*statev1.GetCampaignResponse, error) {
		return h.campaignClient.GetCampaign(ctx, req)
	}
	d.ListSessions = func(ctx context.Context, req *statev1.ListSessionsRequest) (*statev1.ListSessionsResponse, error) {
		return h.sessionClient.ListSessions(ctx, req)
	}
	d.GetSession = func(ctx context.Context, req *statev1.GetSessionRequest) (*statev1.GetSessionResponse, error) {
		return h.sessionClient.GetSession(ctx, req)
	}
	d.StartSession = func(ctx context.Context, req *statev1.StartSessionRequest) (*statev1.StartSessionResponse, error) {
		return h.sessionClient.StartSession(ctx, req)
	}
	d.EndSession = func(ctx context.Context, req *statev1.EndSessionRequest) (*statev1.EndSessionResponse, error) {
		return h.sessionClient.EndSession(ctx, req)
	}
	d.ListParticipants = func(ctx context.Context, req *statev1.ListParticipantsRequest) (*statev1.ListParticipantsResponse, error) {
		return h.participantClient.ListParticipants(ctx, req)
	}
	d.UpdateParticipant = func(ctx context.Context, req *statev1.UpdateParticipantRequest) (*statev1.UpdateParticipantResponse, error) {
		return h.participantClient.UpdateParticipant(ctx, req)
	}
	d.ListCharacters = func(ctx context.Context, req *statev1.ListCharactersRequest) (*statev1.ListCharactersResponse, error) {
		return h.characterClient.ListCharacters(ctx, req)
	}
	d.GetCharacterSheet = func(ctx context.Context, req *statev1.GetCharacterSheetRequest) (*statev1.GetCharacterSheetResponse, error) {
		return h.characterClient.GetCharacterSheet(ctx, req)
	}
	d.CreateCharacter = func(ctx context.Context, req *statev1.CreateCharacterRequest) (*statev1.CreateCharacterResponse, error) {
		return h.characterClient.CreateCharacter(ctx, req)
	}
	d.UpdateCharacter = func(ctx context.Context, req *statev1.UpdateCharacterRequest) (*statev1.UpdateCharacterResponse, error) {
		return h.characterClient.UpdateCharacter(ctx, req)
	}
	d.SetDefaultControl = func(ctx context.Context, req *statev1.SetDefaultControlRequest) (*statev1.SetDefaultControlResponse, error) {
		return h.characterClient.SetDefaultControl(ctx, req)
	}
	d.ListInvites = func(ctx context.Context, req *statev1.ListInvitesRequest) (*statev1.ListInvitesResponse, error) {
		return h.inviteClient.ListInvites(ctx, req)
	}
	d.CreateInvite = func(ctx context.Context, req *statev1.CreateInviteRequest) (*statev1.CreateInviteResponse, error) {
		return h.inviteClient.CreateInvite(ctx, req)
	}
	d.RevokeInvite = func(ctx context.Context, req *statev1.RevokeInviteRequest) (*statev1.RevokeInviteResponse, error) {
		return h.inviteClient.RevokeInvite(ctx, req)
	}
}
