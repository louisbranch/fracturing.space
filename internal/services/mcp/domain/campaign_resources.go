package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ParticipantListResourceHandler(client statev1.ParticipantServiceClient) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if client == nil {
			return nil, fmt.Errorf("participant list client is not configured")
		}

		if req == nil || req.Params == nil || req.Params.URI == "" {
			return nil, fmt.Errorf("campaign ID is required; use URI format campaign://{campaign_id}/participants")
		}
		uri := req.Params.URI

		// Parse campaign_id from URI: expected format is campaign://{campaign_id}/participants.
		campaignID, err := parseCampaignIDFromURI(uri)
		if err != nil {
			return nil, fmt.Errorf("parse campaign ID from URI: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		callCtx, _, err := NewOutgoingContext(runCtx, "")
		if err != nil {
			return nil, fmt.Errorf("create request metadata: %w", err)
		}

		payload := ParticipantListPayload{}
		response, err := client.ListParticipants(callCtx, &statev1.ListParticipantsRequest{
			CampaignId: campaignID,
			PageSize:   10,
		})
		if err != nil {
			return nil, fmt.Errorf("participant list failed: %w", err)
		}
		if response == nil {
			return nil, fmt.Errorf("participant list response is missing")
		}

		for _, participant := range response.GetParticipants() {
			payload.Participants = append(payload.Participants, ParticipantListEntry{
				ID:         participant.GetId(),
				CampaignID: participant.GetCampaignId(),
				Name:       participant.GetName(),
				Role:       participantRoleToString(participant.GetRole()),
				Controller: controllerToString(participant.GetController()),
				CreatedAt:  formatTimestamp(participant.GetCreatedAt()),
				UpdatedAt:  formatTimestamp(participant.GetUpdatedAt()),
			})
		}

		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal participant list: %w", err)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      uri,
					MIMEType: "application/json",
					Text:     string(data),
				},
			},
		}, nil
	}
}

// parseCampaignIDFromURI extracts the campaign ID from a URI of the form campaign://{campaign_id}/participants.
// It parses URIs of the expected format but requires an actual campaign ID.
func parseCampaignIDFromURI(uri string) (string, error) {
	return parseCampaignIDFromResourceURI(uri, "participants")
}

// CharacterListResourceHandler returns a readable character listing resource.
func CharacterListResourceHandler(client statev1.CharacterServiceClient) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if client == nil {
			return nil, fmt.Errorf("character list client is not configured")
		}

		if req == nil || req.Params == nil || req.Params.URI == "" {
			return nil, fmt.Errorf("campaign ID is required; use URI format campaign://{campaign_id}/characters")
		}
		uri := req.Params.URI

		// Parse campaign_id from URI: expected format is campaign://{campaign_id}/characters.
		campaignID, err := parseCampaignIDFromCharacterURI(uri)
		if err != nil {
			return nil, fmt.Errorf("parse campaign ID from URI: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		callCtx, _, err := NewOutgoingContext(runCtx, "")
		if err != nil {
			return nil, fmt.Errorf("create request metadata: %w", err)
		}

		payload := CharacterListPayload{}
		response, err := client.ListCharacters(callCtx, &statev1.ListCharactersRequest{
			CampaignId: campaignID,
			PageSize:   10,
		})
		if err != nil {
			return nil, fmt.Errorf("character list failed: %w", err)
		}
		if response == nil {
			return nil, fmt.Errorf("character list response is missing")
		}

		for _, character := range response.GetCharacters() {
			payload.Characters = append(payload.Characters, CharacterListEntry{
				ID:         character.GetId(),
				CampaignID: character.GetCampaignId(),
				Name:       character.GetName(),
				Kind:       characterKindToString(character.GetKind()),
				Notes:      character.GetNotes(),
				CreatedAt:  formatTimestamp(character.GetCreatedAt()),
				UpdatedAt:  formatTimestamp(character.GetUpdatedAt()),
			})
		}

		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal character list: %w", err)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      uri,
					MIMEType: "application/json",
					Text:     string(data),
				},
			},
		}, nil
	}
}

// parseCampaignIDFromCharacterURI extracts the campaign ID from a URI of the form campaign://{campaign_id}/characters.
// It parses URIs of the expected format but requires an actual campaign ID.
func parseCampaignIDFromCharacterURI(uri string) (string, error) {
	return parseCampaignIDFromResourceURI(uri, "characters")
}

// parseCampaignIDFromCampaignURI extracts the campaign ID from a URI of the form campaign://{campaign_id}.
// It parses URIs of the expected format but requires an actual campaign ID.
// It also rejects URIs with additional path segments, query parameters, or fragments (e.g., campaign://id/participants).
func parseCampaignIDFromCampaignURI(uri string) (string, error) {
	prefix := "campaign://"

	if !strings.HasPrefix(uri, prefix) {
		return "", fmt.Errorf("URI must start with %q", prefix)
	}

	campaignID := strings.TrimPrefix(uri, prefix)
	campaignID = strings.TrimSpace(campaignID)

	if campaignID == "" {
		return "", fmt.Errorf("campaign ID is required in URI")
	}

	// Reject the placeholder value - actual campaign IDs must be provided
	if campaignID == "_" {
		return "", fmt.Errorf("campaign ID placeholder '_' is not a valid campaign ID")
	}

	// Reject URIs with additional path segments, query parameters, or fragments
	// These should be handled by other resource handlers (e.g., campaign://id/participants)
	if strings.ContainsAny(campaignID, "/?#") {
		return "", fmt.Errorf("URI must not contain path segments, query parameters, or fragments after campaign ID")
	}

	return campaignID, nil
}

// CampaignResourceHandler returns a readable single campaign resource.
func CampaignResourceHandler(client statev1.CampaignServiceClient) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if client == nil {
			return nil, fmt.Errorf("campaign client is not configured")
		}

		if req == nil || req.Params == nil || req.Params.URI == "" {
			return nil, fmt.Errorf("campaign ID is required; use URI format campaign://{campaign_id}")
		}
		uri := req.Params.URI

		// Parse campaign_id from URI: expected format is campaign://{campaign_id}.
		campaignID, err := parseCampaignIDFromCampaignURI(uri)
		if err != nil {
			return nil, fmt.Errorf("parse campaign ID from URI: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		callCtx, _, err := NewOutgoingContext(runCtx, "")
		if err != nil {
			return nil, fmt.Errorf("create request metadata: %w", err)
		}

		response, err := client.GetCampaign(callCtx, &statev1.GetCampaignRequest{
			CampaignId: campaignID,
		})
		if err != nil {
			if s, ok := status.FromError(err); ok {
				if s.Code() == codes.NotFound {
					return nil, fmt.Errorf("campaign not found")
				}
				if s.Code() == codes.InvalidArgument {
					return nil, fmt.Errorf("invalid campaign_id: %s", s.Message())
				}
			}
			return nil, fmt.Errorf("get campaign failed: %w", err)
		}
		if response == nil || response.Campaign == nil {
			return nil, fmt.Errorf("campaign response is missing")
		}

		campaign := response.Campaign
		payload := CampaignPayload{
			Campaign: CampaignListEntry{
				ID:               campaign.GetId(),
				Name:             campaign.GetName(),
				Status:           campaignStatusToString(campaign.GetStatus()),
				GmMode:           gmModeToString(campaign.GetGmMode()),
				ParticipantCount: int(campaign.GetParticipantCount()),
				CharacterCount:   int(campaign.GetCharacterCount()),
				GmFear:           0, // GM Fear is now in Snapshot, not Campaign
				ThemePrompt:      campaign.GetThemePrompt(),
				CreatedAt:        formatTimestamp(campaign.GetCreatedAt()),
				UpdatedAt:        formatTimestamp(campaign.GetUpdatedAt()),
				CompletedAt:      formatTimestamp(campaign.GetCompletedAt()),
				ArchivedAt:       formatTimestamp(campaign.GetArchivedAt()),
			},
		}

		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal campaign: %w", err)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      uri,
					MIMEType: "application/json",
					Text:     string(data),
				},
			},
		}, nil
	}
}
