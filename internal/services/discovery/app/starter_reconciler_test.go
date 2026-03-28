package server

import (
	"context"
	"errors"
	"testing"

	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/discovery/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/discovery/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestStarterThemePromptPrefersCampaignTheme(t *testing.T) {
	t.Parallel()

	entry := storage.DiscoveryEntry{
		Kind:          discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_CAMPAIGN_STARTER,
		Description:   "Short description",
		CampaignTheme: "Longer public theme",
	}

	if got := starterThemePrompt(entry); got != "Longer public theme" {
		t.Fatalf("starterThemePrompt() = %q, want %q", got, "Longer public theme")
	}
}

func TestStarterThemePromptFallsBackToDescription(t *testing.T) {
	t.Parallel()

	entry := storage.DiscoveryEntry{
		Kind:        discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_CAMPAIGN_STARTER,
		Description: "Short description",
	}

	if got := starterThemePrompt(entry); got != "Short description" {
		t.Fatalf("starterThemePrompt() = %q, want %q", got, "Short description")
	}
}

func TestTemplateCampaignExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		id     string
		client gamev1.CampaignServiceClient
		want   bool
	}{
		{name: "empty id", want: false},
		{name: "nil client", id: "camp-1", want: false},
		{
			name: "campaign found",
			id:   "camp-1",
			client: &fakeStarterCampaignClient{
				getCampaign: func(_ context.Context, req *gamev1.GetCampaignRequest, _ ...grpc.CallOption) (*gamev1.GetCampaignResponse, error) {
					if req.GetCampaignId() != "camp-1" {
						t.Fatalf("GetCampaign() campaign id = %q, want %q", req.GetCampaignId(), "camp-1")
					}
					return &gamev1.GetCampaignResponse{Campaign: &gamev1.Campaign{Id: " camp-1 "}}, nil
				},
			},
			want: true,
		},
		{
			name: "campaign missing",
			id:   "camp-1",
			client: &fakeStarterCampaignClient{
				getCampaign: func(context.Context, *gamev1.GetCampaignRequest, ...grpc.CallOption) (*gamev1.GetCampaignResponse, error) {
					return nil, status.Error(codes.NotFound, "missing")
				},
			},
			want: false,
		},
		{
			name: "empty response campaign id",
			id:   "camp-1",
			client: &fakeStarterCampaignClient{
				getCampaign: func(context.Context, *gamev1.GetCampaignRequest, ...grpc.CallOption) (*gamev1.GetCampaignResponse, error) {
					return &gamev1.GetCampaignResponse{Campaign: &gamev1.Campaign{}}, nil
				},
			},
			want: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := templateCampaignExists(context.Background(), tc.client, tc.id); got != tc.want {
				t.Fatalf("templateCampaignExists() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCreateStarterTemplateCampaign_Success(t *testing.T) {
	t.Parallel()

	var createReq *gamev1.CreateCampaignRequest
	var createCharacterReq *gamev1.CreateCharacterRequest
	var workflowReq *gamev1.ApplyCharacterCreationWorkflowRequest

	campaignClient := &fakeStarterCampaignClient{
		createCampaign: func(_ context.Context, req *gamev1.CreateCampaignRequest, _ ...grpc.CallOption) (*gamev1.CreateCampaignResponse, error) {
			createReq = req
			return &gamev1.CreateCampaignResponse{
				Campaign:         &gamev1.Campaign{Id: "camp-42"},
				OwnerParticipant: &gamev1.Participant{Id: "participant-7"},
			}, nil
		},
	}
	characterClient := &fakeStarterCharacterClient{
		createCharacter: func(ctx context.Context, req *gamev1.CreateCharacterRequest, _ ...grpc.CallOption) (*gamev1.CreateCharacterResponse, error) {
			assertOutgoingParticipantID(t, ctx, "participant-7")
			createCharacterReq = req
			return &gamev1.CreateCharacterResponse{Character: &gamev1.Character{Id: "char-9"}}, nil
		},
		applyWorkflow: func(ctx context.Context, req *gamev1.ApplyCharacterCreationWorkflowRequest, _ ...grpc.CallOption) (*gamev1.ApplyCharacterCreationWorkflowResponse, error) {
			assertOutgoingParticipantID(t, ctx, "participant-7")
			workflowReq = req
			return &gamev1.ApplyCharacterCreationWorkflowResponse{}, nil
		},
	}

	campaignID, err := createStarterTemplateCampaign(context.Background(), campaignClient, characterClient, testStarterDefinition())
	if err != nil {
		t.Fatalf("createStarterTemplateCampaign() error = %v", err)
	}
	if campaignID != "camp-42" {
		t.Fatalf("campaign id = %q, want %q", campaignID, "camp-42")
	}
	if createReq == nil {
		t.Fatal("expected CreateCampaign request")
	}
	if createReq.GetThemePrompt() != "Longer public theme" {
		t.Fatalf("CreateCampaign() theme prompt = %q, want %q", createReq.GetThemePrompt(), "Longer public theme")
	}
	if createCharacterReq == nil {
		t.Fatal("expected CreateCharacter request")
	}
	if createCharacterReq.GetCampaignId() != "camp-42" {
		t.Fatalf("CreateCharacter() campaign id = %q, want %q", createCharacterReq.GetCampaignId(), "camp-42")
	}
	if createCharacterReq.GetName() != "Vera Flint" {
		t.Fatalf("CreateCharacter() name = %q, want %q", createCharacterReq.GetName(), "Vera Flint")
	}
	if workflowReq == nil {
		t.Fatal("expected ApplyCharacterCreationWorkflow request")
	}
	if workflowReq.GetCampaignId() != "camp-42" {
		t.Fatalf("ApplyCharacterCreationWorkflow() campaign id = %q, want %q", workflowReq.GetCampaignId(), "camp-42")
	}
	if workflowReq.GetCharacterId() != "char-9" {
		t.Fatalf("ApplyCharacterCreationWorkflow() character id = %q, want %q", workflowReq.GetCharacterId(), "char-9")
	}
	daggerheart := workflowReq.GetDaggerheart()
	if daggerheart == nil {
		t.Fatal("expected daggerheart workflow input")
	}
	if got := daggerheart.GetExperiencesInput().GetExperiences(); len(got) != 1 {
		t.Fatalf("workflow experiences len = %d, want %d", len(got), 1)
	}
	if got := daggerheart.GetExperiencesInput().GetExperiences()[0]; got.GetName() != "Smuggler routes" || got.GetModifier() != 2 {
		t.Fatalf("workflow experience = %#v, want Smuggler routes/+2", got)
	}
	if got := daggerheart.GetConnectionsInput().GetConnections(); got != "I owe the harbormaster a favor." {
		t.Fatalf("workflow connections = %q, want %q", got, "I owe the harbormaster a favor.")
	}
}

func TestCreateStarterTemplateCampaign_ArchivesCampaignOnWorkflowFailure(t *testing.T) {
	t.Parallel()

	var archivedCampaignID string

	campaignClient := &fakeStarterCampaignClient{
		createCampaign: func(_ context.Context, _ *gamev1.CreateCampaignRequest, _ ...grpc.CallOption) (*gamev1.CreateCampaignResponse, error) {
			return &gamev1.CreateCampaignResponse{
				Campaign:         &gamev1.Campaign{Id: "camp-42"},
				OwnerParticipant: &gamev1.Participant{Id: "participant-7"},
			}, nil
		},
		archiveCampaign: func(ctx context.Context, req *gamev1.ArchiveCampaignRequest, _ ...grpc.CallOption) (*gamev1.ArchiveCampaignResponse, error) {
			assertOutgoingParticipantID(t, ctx, "participant-7")
			archivedCampaignID = req.GetCampaignId()
			return &gamev1.ArchiveCampaignResponse{}, nil
		},
	}
	characterClient := &fakeStarterCharacterClient{
		createCharacter: func(ctx context.Context, _ *gamev1.CreateCharacterRequest, _ ...grpc.CallOption) (*gamev1.CreateCharacterResponse, error) {
			assertOutgoingParticipantID(t, ctx, "participant-7")
			return &gamev1.CreateCharacterResponse{Character: &gamev1.Character{Id: "char-9"}}, nil
		},
		applyWorkflow: func(ctx context.Context, _ *gamev1.ApplyCharacterCreationWorkflowRequest, _ ...grpc.CallOption) (*gamev1.ApplyCharacterCreationWorkflowResponse, error) {
			assertOutgoingParticipantID(t, ctx, "participant-7")
			return nil, errors.New("workflow boom")
		},
	}

	_, err := createStarterTemplateCampaign(context.Background(), campaignClient, characterClient, testStarterDefinition())
	if err == nil {
		t.Fatal("expected workflow failure")
	}
	if archivedCampaignID != "camp-42" {
		t.Fatalf("archived campaign id = %q, want %q", archivedCampaignID, "camp-42")
	}
}

func TestIsRetryableStarterReconciliationError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "plain error", err: errors.New("boom"), want: false},
		{name: "unavailable", err: status.Error(codes.Unavailable, "try later"), want: true},
		{name: "deadline", err: status.Error(codes.DeadlineExceeded, "timeout"), want: true},
		{name: "aborted", err: status.Error(codes.Aborted, "retry"), want: true},
		{name: "invalid argument missing catalog dependency", err: status.Error(codes.InvalidArgument, "community xyz is not found"), want: true},
		{name: "invalid argument permanent", err: status.Error(codes.InvalidArgument, "bad request"), want: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := isRetryableStarterReconciliationError(tc.err); got != tc.want {
				t.Fatalf("isRetryableStarterReconciliationError() = %v, want %v", got, tc.want)
			}
		})
	}
}

type fakeStarterCampaignClient struct {
	gamev1.CampaignServiceClient

	createCampaign  func(context.Context, *gamev1.CreateCampaignRequest, ...grpc.CallOption) (*gamev1.CreateCampaignResponse, error)
	getCampaign     func(context.Context, *gamev1.GetCampaignRequest, ...grpc.CallOption) (*gamev1.GetCampaignResponse, error)
	archiveCampaign func(context.Context, *gamev1.ArchiveCampaignRequest, ...grpc.CallOption) (*gamev1.ArchiveCampaignResponse, error)
}

func (f *fakeStarterCampaignClient) CreateCampaign(ctx context.Context, req *gamev1.CreateCampaignRequest, opts ...grpc.CallOption) (*gamev1.CreateCampaignResponse, error) {
	if f.createCampaign == nil {
		return nil, errors.New("CreateCampaign not implemented")
	}
	return f.createCampaign(ctx, req, opts...)
}

func (f *fakeStarterCampaignClient) GetCampaign(ctx context.Context, req *gamev1.GetCampaignRequest, opts ...grpc.CallOption) (*gamev1.GetCampaignResponse, error) {
	if f.getCampaign == nil {
		return nil, errors.New("GetCampaign not implemented")
	}
	return f.getCampaign(ctx, req, opts...)
}

func (f *fakeStarterCampaignClient) ArchiveCampaign(ctx context.Context, req *gamev1.ArchiveCampaignRequest, opts ...grpc.CallOption) (*gamev1.ArchiveCampaignResponse, error) {
	if f.archiveCampaign == nil {
		return nil, errors.New("ArchiveCampaign not implemented")
	}
	return f.archiveCampaign(ctx, req, opts...)
}

type fakeStarterCharacterClient struct {
	gamev1.CharacterServiceClient

	createCharacter func(context.Context, *gamev1.CreateCharacterRequest, ...grpc.CallOption) (*gamev1.CreateCharacterResponse, error)
	applyWorkflow   func(context.Context, *gamev1.ApplyCharacterCreationWorkflowRequest, ...grpc.CallOption) (*gamev1.ApplyCharacterCreationWorkflowResponse, error)
}

func (f *fakeStarterCharacterClient) CreateCharacter(ctx context.Context, req *gamev1.CreateCharacterRequest, opts ...grpc.CallOption) (*gamev1.CreateCharacterResponse, error) {
	if f.createCharacter == nil {
		return nil, errors.New("CreateCharacter not implemented")
	}
	return f.createCharacter(ctx, req, opts...)
}

func (f *fakeStarterCharacterClient) ApplyCharacterCreationWorkflow(ctx context.Context, req *gamev1.ApplyCharacterCreationWorkflowRequest, opts ...grpc.CallOption) (*gamev1.ApplyCharacterCreationWorkflowResponse, error) {
	if f.applyWorkflow == nil {
		return nil, errors.New("ApplyCharacterCreationWorkflow not implemented")
	}
	return f.applyWorkflow(ctx, req, opts...)
}

func assertOutgoingParticipantID(t *testing.T, ctx context.Context, want string) {
	t.Helper()

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("expected outgoing metadata")
	}
	got := md.Get(grpcmeta.ParticipantIDHeader)
	if len(got) != 1 || got[0] != want {
		t.Fatalf("outgoing participant id = %v, want [%q]", got, want)
	}
}

func testStarterDefinition() catalog.StarterDefinition {
	return catalog.StarterDefinition{
		Entry: storage.DiscoveryEntry{
			Kind:          discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_CAMPAIGN_STARTER,
			Title:         "Ash Harbor",
			Description:   "Short description",
			CampaignTheme: "Longer public theme",
		},
		Character: catalog.StarterCharacterDefinition{
			Name:         "Vera Flint",
			Pronouns:     "she/her",
			Summary:      "A quick-handed smuggler with unfinished business.",
			ClassID:      "rogue",
			SubclassID:   "nightwalker",
			AncestryID:   "human",
			CommunityID:  "ridgeborne",
			WeaponIDs:    []string{"blade"},
			ArmorID:      "leather",
			PotionItemID: "minor-health",
			Description:  "Lean, alert, and always watching exits.",
			Background:   "Harbor runner",
			Connections:  "I owe the harbormaster a favor.",
			DomainCardIDs: []string{
				"midnight-card",
			},
			Traits: catalog.StarterTraitDefinition{
				Agility:   2,
				Strength:  0,
				Finesse:   1,
				Instinct:  1,
				Presence:  0,
				Knowledge: -1,
			},
			Experiences: []catalog.StarterExperienceDefinition{
				{Name: "Smuggler routes", Modifier: 2},
				{Name: "   ", Modifier: 1},
			},
		},
	}
}
