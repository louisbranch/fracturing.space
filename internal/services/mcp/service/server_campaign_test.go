package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestCampaignCreateHandlerReturnsClientError ensures gRPC errors are returned as tool errors.
func TestCampaignCreateHandlerReturnsClientError(t *testing.T) {
	client := &fakeCampaignClient{err: errors.New("boom")}
	handler := domain.CampaignCreateHandler(client, nil)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.CampaignCreateInput{
		Name:   "New Campaign",
		GmMode: "HUMAN",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

func TestCampaignCreateHandlerRequiresOwnerParticipant(t *testing.T) {
	campaign := &statev1.Campaign{Id: "camp-123"}
	cases := []struct {
		name     string
		response *statev1.CreateCampaignResponse
	}{
		{
			name: "missing owner participant",
			response: &statev1.CreateCampaignResponse{
				Campaign: campaign,
			},
		},
		{
			name: "empty owner participant id",
			response: &statev1.CreateCampaignResponse{
				Campaign: campaign,
				OwnerParticipant: &statev1.Participant{
					Id: "",
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := &fakeCampaignClient{response: tc.response}
			handler := domain.CampaignCreateHandler(client, nil)

			result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.CampaignCreateInput{
				Name:   "New Campaign",
				GmMode: "HUMAN",
			})
			if err == nil {
				t.Fatal("expected error")
			}
			if result != nil {
				t.Fatal("expected nil result on error")
			}
		})
	}
}

// TestCampaignCreateHandlerMapsRequestAndResponse ensures inputs and outputs map consistently.
func TestCampaignCreateHandlerMapsRequestAndResponse(t *testing.T) {
	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	client := &fakeCampaignClient{response: &statev1.CreateCampaignResponse{
		Campaign: &statev1.Campaign{
			Id:               "camp-123",
			Name:             "Snowbound",
			GmMode:           statev1.GmMode_AI,
			ParticipantCount: 5,
			CharacterCount:   3,
			ThemePrompt:      "ice and steel",
			CreatedAt:        timestamppb.New(now),
			UpdatedAt:        timestamppb.New(now),
		},
		OwnerParticipant: &statev1.Participant{
			Id: "part-owner",
		},
	}}
	result, output, err := domain.CampaignCreateHandler(client, nil)(
		context.Background(),
		&mcp.CallToolRequest{},
		domain.CampaignCreateInput{
			Name:        "Snowbound",
			GmMode:      "HUMAN",
			ThemePrompt: "ice and steel",
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	requireToolMetadata(t, result)
	if client.lastRequest == nil {
		t.Fatal("expected gRPC request")
	}
	if client.lastRequest.GetGmMode() != statev1.GmMode_HUMAN {
		t.Fatalf("expected gm mode HUMAN, got %v", client.lastRequest.GetGmMode())
	}
	if output.ID != "camp-123" {
		t.Fatalf("expected id camp-123, got %q", output.ID)
	}
	if output.OwnerParticipantID != "part-owner" {
		t.Fatalf("expected owner participant id part-owner, got %q", output.OwnerParticipantID)
	}
	if output.GmMode != "AI" {
		t.Fatalf("expected gm mode AI, got %q", output.GmMode)
	}
	if output.ParticipantCount != 5 {
		t.Fatalf("expected participant count 5, got %d", output.ParticipantCount)
	}
	if output.CharacterCount != 3 {
		t.Fatalf("expected character count 3, got %d", output.CharacterCount)
	}
}

// TestCampaignEndHandlerMapsRequestAndResponse ensures inputs and outputs map consistently.
func TestCampaignEndHandlerMapsRequestAndResponse(t *testing.T) {
	now := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	client := &fakeCampaignClient{endCampaignResponse: &statev1.EndCampaignResponse{
		Campaign: &statev1.Campaign{
			Id:          "camp-123",
			Name:        "Finale",
			Status:      statev1.CampaignStatus_COMPLETED,
			GmMode:      statev1.GmMode_HUMAN,
			CreatedAt:   timestamppb.New(now.Add(-2 * time.Hour)),
			UpdatedAt:   timestamppb.New(now),
			CompletedAt: timestamppb.New(now),
		},
	}}

	result, output, err := domain.CampaignEndHandler(client, nil, nil)(
		context.Background(),
		&mcp.CallToolRequest{},
		domain.CampaignStatusChangeInput{CampaignID: "camp-123"},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	requireToolMetadata(t, result)
	if client.lastEndCampaignRequest == nil {
		t.Fatal("expected gRPC request")
	}
	if client.lastEndCampaignRequest.GetCampaignId() != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", client.lastEndCampaignRequest.GetCampaignId())
	}
	if output.ID != "camp-123" {
		t.Fatalf("expected id camp-123, got %q", output.ID)
	}
	if output.Status != "COMPLETED" {
		t.Fatalf("expected status COMPLETED, got %q", output.Status)
	}
	if output.CompletedAt != now.Format(time.RFC3339) {
		t.Fatalf("expected completed_at %q, got %q", now.Format(time.RFC3339), output.CompletedAt)
	}
	if output.ArchivedAt != "" {
		t.Fatalf("expected empty archived_at, got %q", output.ArchivedAt)
	}
}

// TestCampaignEndHandlerUsesContextDefaults ensures campaign_id defaults from context.
func TestCampaignEndHandlerUsesContextDefaults(t *testing.T) {
	client := &fakeCampaignClient{endCampaignResponse: &statev1.EndCampaignResponse{
		Campaign: &statev1.Campaign{Id: "camp-123"},
	}}
	getContext := func() domain.Context {
		return domain.Context{CampaignID: "camp-123"}
	}

	result, _, err := domain.CampaignEndHandler(client, getContext, nil)(
		context.Background(),
		&mcp.CallToolRequest{},
		domain.CampaignStatusChangeInput{},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	requireToolMetadata(t, result)
	if client.lastEndCampaignRequest == nil {
		t.Fatal("expected gRPC request")
	}
	if client.lastEndCampaignRequest.GetCampaignId() != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", client.lastEndCampaignRequest.GetCampaignId())
	}
}

// TestCampaignEndHandlerRejectsMissingCampaign ensures campaign_id is required.
func TestCampaignEndHandlerRejectsMissingCampaign(t *testing.T) {
	client := &fakeCampaignClient{}
	handler := domain.CampaignEndHandler(client, func() domain.Context { return domain.Context{} }, nil)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.CampaignStatusChangeInput{})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestCampaignEndHandlerReturnsClientError ensures gRPC errors are returned as tool errors.
func TestCampaignEndHandlerReturnsClientError(t *testing.T) {
	client := &fakeCampaignClient{endCampaignErr: errors.New("boom")}
	handler := domain.CampaignEndHandler(client, nil, nil)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.CampaignStatusChangeInput{CampaignID: "camp-123"})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestCampaignEndHandlerRejectsEmptyResponse ensures nil responses are rejected.
func TestCampaignEndHandlerRejectsEmptyResponse(t *testing.T) {
	client := &fakeCampaignClient{endCampaignResponse: &statev1.EndCampaignResponse{}}
	handler := domain.CampaignEndHandler(client, nil, nil)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.CampaignStatusChangeInput{CampaignID: "camp-123"})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestCampaignArchiveHandlerMapsRequestAndResponse ensures inputs and outputs map consistently.
func TestCampaignArchiveHandlerMapsRequestAndResponse(t *testing.T) {
	now := time.Date(2026, 2, 1, 11, 0, 0, 0, time.UTC)
	client := &fakeCampaignClient{archiveCampaignResponse: &statev1.ArchiveCampaignResponse{
		Campaign: &statev1.Campaign{
			Id:         "camp-123",
			Name:       "Finale",
			Status:     statev1.CampaignStatus_ARCHIVED,
			GmMode:     statev1.GmMode_AI,
			CreatedAt:  timestamppb.New(now.Add(-2 * time.Hour)),
			UpdatedAt:  timestamppb.New(now),
			ArchivedAt: timestamppb.New(now),
		},
	}}

	result, output, err := domain.CampaignArchiveHandler(client, nil, nil)(
		context.Background(),
		&mcp.CallToolRequest{},
		domain.CampaignStatusChangeInput{CampaignID: "camp-123"},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	requireToolMetadata(t, result)
	if client.lastArchiveCampaignRequest == nil {
		t.Fatal("expected gRPC request")
	}
	if client.lastArchiveCampaignRequest.GetCampaignId() != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", client.lastArchiveCampaignRequest.GetCampaignId())
	}
	if output.ID != "camp-123" {
		t.Fatalf("expected id camp-123, got %q", output.ID)
	}
	if output.Status != "ARCHIVED" {
		t.Fatalf("expected status ARCHIVED, got %q", output.Status)
	}
	if output.ArchivedAt != now.Format(time.RFC3339) {
		t.Fatalf("expected archived_at %q, got %q", now.Format(time.RFC3339), output.ArchivedAt)
	}
}

// TestCampaignArchiveHandlerUsesContextDefaults ensures campaign_id defaults from context.
func TestCampaignArchiveHandlerUsesContextDefaults(t *testing.T) {
	client := &fakeCampaignClient{archiveCampaignResponse: &statev1.ArchiveCampaignResponse{
		Campaign: &statev1.Campaign{Id: "camp-123"},
	}}
	getContext := func() domain.Context {
		return domain.Context{CampaignID: "camp-123"}
	}

	result, _, err := domain.CampaignArchiveHandler(client, getContext, nil)(
		context.Background(),
		&mcp.CallToolRequest{},
		domain.CampaignStatusChangeInput{},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	requireToolMetadata(t, result)
	if client.lastArchiveCampaignRequest == nil {
		t.Fatal("expected gRPC request")
	}
	if client.lastArchiveCampaignRequest.GetCampaignId() != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", client.lastArchiveCampaignRequest.GetCampaignId())
	}
}

// TestCampaignArchiveHandlerRejectsMissingCampaign ensures campaign_id is required.
func TestCampaignArchiveHandlerRejectsMissingCampaign(t *testing.T) {
	client := &fakeCampaignClient{}
	handler := domain.CampaignArchiveHandler(client, func() domain.Context { return domain.Context{} }, nil)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.CampaignStatusChangeInput{})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestCampaignArchiveHandlerReturnsClientError ensures gRPC errors are returned as tool errors.
func TestCampaignArchiveHandlerReturnsClientError(t *testing.T) {
	client := &fakeCampaignClient{archiveCampaignErr: errors.New("boom")}
	handler := domain.CampaignArchiveHandler(client, nil, nil)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.CampaignStatusChangeInput{CampaignID: "camp-123"})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestCampaignArchiveHandlerRejectsEmptyResponse ensures nil responses are rejected.
func TestCampaignArchiveHandlerRejectsEmptyResponse(t *testing.T) {
	client := &fakeCampaignClient{archiveCampaignResponse: &statev1.ArchiveCampaignResponse{}}
	handler := domain.CampaignArchiveHandler(client, nil, nil)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.CampaignStatusChangeInput{CampaignID: "camp-123"})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestCampaignRestoreHandlerMapsRequestAndResponse ensures inputs and outputs map consistently.
func TestCampaignRestoreHandlerMapsRequestAndResponse(t *testing.T) {
	now := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	client := &fakeCampaignClient{restoreCampaignResponse: &statev1.RestoreCampaignResponse{
		Campaign: &statev1.Campaign{
			Id:        "camp-123",
			Name:      "Finale",
			Status:    statev1.CampaignStatus_DRAFT,
			GmMode:    statev1.GmMode_HYBRID,
			CreatedAt: timestamppb.New(now.Add(-2 * time.Hour)),
			UpdatedAt: timestamppb.New(now),
		},
	}}

	result, output, err := domain.CampaignRestoreHandler(client, nil, nil)(
		context.Background(),
		&mcp.CallToolRequest{},
		domain.CampaignStatusChangeInput{CampaignID: "camp-123"},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	requireToolMetadata(t, result)
	if client.lastRestoreCampaignRequest == nil {
		t.Fatal("expected gRPC request")
	}
	if client.lastRestoreCampaignRequest.GetCampaignId() != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", client.lastRestoreCampaignRequest.GetCampaignId())
	}
	if output.ID != "camp-123" {
		t.Fatalf("expected id camp-123, got %q", output.ID)
	}
	if output.Status != "DRAFT" {
		t.Fatalf("expected status DRAFT, got %q", output.Status)
	}
	if output.CompletedAt != "" {
		t.Fatalf("expected empty completed_at, got %q", output.CompletedAt)
	}
	if output.ArchivedAt != "" {
		t.Fatalf("expected empty archived_at, got %q", output.ArchivedAt)
	}
}

// TestCampaignRestoreHandlerUsesContextDefaults ensures campaign_id defaults from context.
func TestCampaignRestoreHandlerUsesContextDefaults(t *testing.T) {
	client := &fakeCampaignClient{restoreCampaignResponse: &statev1.RestoreCampaignResponse{
		Campaign: &statev1.Campaign{Id: "camp-123"},
	}}
	getContext := func() domain.Context {
		return domain.Context{CampaignID: "camp-123"}
	}

	result, _, err := domain.CampaignRestoreHandler(client, getContext, nil)(
		context.Background(),
		&mcp.CallToolRequest{},
		domain.CampaignStatusChangeInput{},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	requireToolMetadata(t, result)
	if client.lastRestoreCampaignRequest == nil {
		t.Fatal("expected gRPC request")
	}
	if client.lastRestoreCampaignRequest.GetCampaignId() != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", client.lastRestoreCampaignRequest.GetCampaignId())
	}
}

// TestCampaignRestoreHandlerRejectsMissingCampaign ensures campaign_id is required.
func TestCampaignRestoreHandlerRejectsMissingCampaign(t *testing.T) {
	client := &fakeCampaignClient{}
	handler := domain.CampaignRestoreHandler(client, func() domain.Context { return domain.Context{} }, nil)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.CampaignStatusChangeInput{})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestCampaignRestoreHandlerReturnsClientError ensures gRPC errors are returned as tool errors.
func TestCampaignRestoreHandlerReturnsClientError(t *testing.T) {
	client := &fakeCampaignClient{restoreCampaignErr: errors.New("boom")}
	handler := domain.CampaignRestoreHandler(client, nil, nil)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.CampaignStatusChangeInput{CampaignID: "camp-123"})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestCampaignRestoreHandlerRejectsEmptyResponse ensures nil responses are rejected.
func TestCampaignRestoreHandlerRejectsEmptyResponse(t *testing.T) {
	client := &fakeCampaignClient{restoreCampaignResponse: &statev1.RestoreCampaignResponse{}}
	handler := domain.CampaignRestoreHandler(client, nil, nil)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.CampaignStatusChangeInput{CampaignID: "camp-123"})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestCampaignListResourceHandlerReturnsClientError ensures list errors are returned.
func TestCampaignListResourceHandlerReturnsClientError(t *testing.T) {
	client := &fakeCampaignClient{listErr: errors.New("boom")}
	handler := domain.CampaignListResourceHandler(client)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if client.listCalls != 1 {
		t.Fatalf("expected 1 list call, got %d", client.listCalls)
	}
}

// TestCampaignListResourceHandlerRejectsEmptyResponse ensures nil responses are rejected.
func TestCampaignListResourceHandlerRejectsEmptyResponse(t *testing.T) {
	client := &fakeCampaignClient{}
	handler := domain.CampaignListResourceHandler(client)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestCampaignListResourceHandlerMapsResponse ensures JSON payload is formatted.
func TestCampaignListResourceHandlerMapsResponse(t *testing.T) {
	now := time.Date(2026, 1, 23, 13, 0, 0, 0, time.UTC)
	client := &fakeCampaignClient{listResponse: &statev1.ListCampaignsResponse{
		Campaigns: []*statev1.Campaign{{
			Id:               "camp-1",
			Name:             "Red Sands",
			GmMode:           statev1.GmMode_HUMAN,
			ParticipantCount: 4,
			CharacterCount:   2,
			CanStartSession:  true,
			ThemePrompt:      "desert skies",
			CreatedAt:        timestamppb.New(now),
			UpdatedAt:        timestamppb.New(now.Add(time.Hour)),
		}},
		NextPageToken: "next",
	}}

	handler := domain.CampaignListResourceHandler(client)
	result, err := handler(context.Background(), &mcp.ReadResourceRequest{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil || len(result.Contents) != 1 {
		t.Fatalf("expected 1 content item, got %v", result)
	}
	if client.lastListRequest == nil {
		t.Fatal("expected list request")
	}
	if client.lastListRequest.GetPageSize() != 10 {
		t.Fatalf("expected page size 10, got %d", client.lastListRequest.GetPageSize())
	}
	if client.lastListRequest.GetPageToken() != "" {
		t.Fatalf("expected empty page token, got %q", client.lastListRequest.GetPageToken())
	}
	if client.listCalls != 1 {
		t.Fatalf("expected 1 list call, got %d", client.listCalls)
	}

	var payload struct {
		Campaigns []struct {
			ID               string `json:"id"`
			Name             string `json:"name"`
			GmMode           string `json:"gm_mode"`
			ParticipantCount int    `json:"participant_count"`
			CharacterCount   int    `json:"character_count"`
			CanStartSession  bool   `json:"can_start_session"`
			ThemePrompt      string `json:"theme_prompt"`
			CreatedAt        string `json:"created_at"`
			UpdatedAt        string `json:"updated_at"`
		} `json:"campaigns"`
	}
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if len(payload.Campaigns) != 1 {
		t.Fatalf("expected 1 campaign, got %d", len(payload.Campaigns))
	}
	if payload.Campaigns[0].ID != "camp-1" {
		t.Fatalf("expected id camp-1, got %q", payload.Campaigns[0].ID)
	}
	if payload.Campaigns[0].GmMode != "HUMAN" {
		t.Fatalf("expected gm mode HUMAN, got %q", payload.Campaigns[0].GmMode)
	}
	if !payload.Campaigns[0].CanStartSession {
		t.Fatalf("expected can_start_session true, got %t", payload.Campaigns[0].CanStartSession)
	}
	if payload.Campaigns[0].CreatedAt != now.Format(time.RFC3339) {
		t.Fatalf("expected created_at %q, got %q", now.Format(time.RFC3339), payload.Campaigns[0].CreatedAt)
	}
	if payload.Campaigns[0].UpdatedAt != now.Add(time.Hour).Format(time.RFC3339) {
		t.Fatalf("expected updated_at %q, got %q", now.Add(time.Hour).Format(time.RFC3339), payload.Campaigns[0].UpdatedAt)
	}
}

// TestCampaignResourceHandlerMapsResponse ensures JSON payload is formatted.
func TestCampaignResourceHandlerMapsResponse(t *testing.T) {
	now := time.Date(2026, 1, 23, 13, 0, 0, 0, time.UTC)
	client := &fakeCampaignClient{getCampaignResponse: &statev1.GetCampaignResponse{
		Campaign: &statev1.Campaign{
			Id:               "camp-1",
			Name:             "Red Sands",
			GmMode:           statev1.GmMode_HUMAN,
			ParticipantCount: 4,
			CharacterCount:   2,
			CanStartSession:  true,
			ThemePrompt:      "desert skies",
			CreatedAt:        timestamppb.New(now),
			UpdatedAt:        timestamppb.New(now.Add(time.Hour)),
		},
	}}

	handler := domain.CampaignResourceHandler(client)
	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "campaign://camp-1",
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil || len(result.Contents) != 1 {
		t.Fatalf("expected 1 content item, got %v", result)
	}
	if client.lastGetCampaignRequest == nil {
		t.Fatal("expected get campaign request")
	}
	if client.lastGetCampaignRequest.GetCampaignId() != "camp-1" {
		t.Fatalf("expected campaign id camp-1, got %q", client.lastGetCampaignRequest.GetCampaignId())
	}

	var payload struct {
		Campaign struct {
			ID               string `json:"id"`
			Name             string `json:"name"`
			GmMode           string `json:"gm_mode"`
			ParticipantCount int    `json:"participant_count"`
			CharacterCount   int    `json:"character_count"`
			CanStartSession  bool   `json:"can_start_session"`
			ThemePrompt      string `json:"theme_prompt"`
			CreatedAt        string `json:"created_at"`
			UpdatedAt        string `json:"updated_at"`
		} `json:"campaign"`
	}
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Campaign.ID != "camp-1" {
		t.Fatalf("expected id camp-1, got %q", payload.Campaign.ID)
	}
	if payload.Campaign.Name != "Red Sands" {
		t.Fatalf("expected name Red Sands, got %q", payload.Campaign.Name)
	}
	if payload.Campaign.GmMode != "HUMAN" {
		t.Fatalf("expected gm mode HUMAN, got %q", payload.Campaign.GmMode)
	}
	if payload.Campaign.ParticipantCount != 4 {
		t.Fatalf("expected participant_count 4, got %d", payload.Campaign.ParticipantCount)
	}
	if payload.Campaign.CharacterCount != 2 {
		t.Fatalf("expected character_count 2, got %d", payload.Campaign.CharacterCount)
	}
	if !payload.Campaign.CanStartSession {
		t.Fatalf("expected can_start_session true, got %t", payload.Campaign.CanStartSession)
	}
	if payload.Campaign.CreatedAt != now.Format(time.RFC3339) {
		t.Fatalf("expected created_at %q, got %q", now.Format(time.RFC3339), payload.Campaign.CreatedAt)
	}
	if payload.Campaign.UpdatedAt != now.Add(time.Hour).Format(time.RFC3339) {
		t.Fatalf("expected updated_at %q, got %q", now.Add(time.Hour).Format(time.RFC3339), payload.Campaign.UpdatedAt)
	}
}

// TestCampaignResourceHandlerReturnsNotFound ensures NotFound errors are returned.
func TestCampaignResourceHandlerReturnsNotFound(t *testing.T) {
	client := &fakeCampaignClient{getCampaignErr: status.Error(codes.NotFound, "campaign not found")}
	handler := domain.CampaignResourceHandler(client)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "campaign://camp-999",
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if !strings.Contains(err.Error(), "campaign not found") {
		t.Fatalf("expected 'campaign not found' in error, got %q", err.Error())
	}
}

// TestCampaignResourceHandlerReturnsInvalidArgument ensures InvalidArgument errors are returned.
func TestCampaignResourceHandlerReturnsInvalidArgument(t *testing.T) {
	client := &fakeCampaignClient{getCampaignErr: status.Error(codes.InvalidArgument, "campaign id is required")}
	handler := domain.CampaignResourceHandler(client)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "campaign://invalid-id",
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if !strings.Contains(err.Error(), "invalid campaign_id") {
		t.Fatalf("expected 'invalid campaign_id' in error, got %q", err.Error())
	}
}

// TestCampaignResourceHandlerRejectsMissingURI ensures missing URI is rejected.
func TestCampaignResourceHandlerRejectsMissingURI(t *testing.T) {
	client := &fakeCampaignClient{}
	handler := domain.CampaignResourceHandler(client)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "",
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if !strings.Contains(err.Error(), "campaign ID is required") {
		t.Fatalf("expected 'campaign ID is required' in error, got %q", err.Error())
	}
}

// TestCampaignResourceHandlerRejectsEmptyID ensures empty campaign ID is rejected.
func TestCampaignResourceHandlerRejectsEmptyID(t *testing.T) {
	client := &fakeCampaignClient{}
	handler := domain.CampaignResourceHandler(client)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "campaign://",
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if !strings.Contains(err.Error(), "campaign ID is required") {
		t.Fatalf("expected 'campaign ID is required' in error, got %q", err.Error())
	}
}

// TestCampaignResourceHandlerRejectsSuffixedURI ensures URIs with path segments are rejected.
func TestCampaignResourceHandlerRejectsSuffixedURI(t *testing.T) {
	client := &fakeCampaignClient{}
	handler := domain.CampaignResourceHandler(client)

	testCases := []struct {
		name string
		uri  string
	}{
		{"path segment", "campaign://camp-1/participants"},
		{"query parameter", "campaign://camp-1?foo=bar"},
		{"fragment", "campaign://camp-1#section"},
		{"path and query", "campaign://camp-1/participants?foo=bar"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := handler(context.Background(), &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: tc.uri,
				},
			})
			if err == nil {
				t.Fatal("expected error")
			}
			if result != nil {
				t.Fatal("expected nil result on error")
			}
			if !strings.Contains(err.Error(), "path segments") && !strings.Contains(err.Error(), "query parameters") && !strings.Contains(err.Error(), "fragments") {
				t.Fatalf("expected error about path segments/query parameters/fragments, got %q", err.Error())
			}
		})
	}
}
