package game

import (
	"context"
	"errors"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultListCampaignsPageSize = 10
	maxListCampaignsPageSize     = 10
)

// CampaignService implements the game.v1.CampaignService gRPC API.
type CampaignService struct {
	campaignv1.UnimplementedCampaignServiceServer
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
	authClient  authv1.AuthServiceClient
}

// NewCampaignService creates a CampaignService with default dependencies.
func NewCampaignService(stores Stores) *CampaignService {
	return &CampaignService{
		stores:      stores,
		clock:       time.Now,
		idGenerator: id.NewID,
	}
}

// NewCampaignServiceWithAuth creates a CampaignService with an auth client.
func NewCampaignServiceWithAuth(stores Stores, authClient authv1.AuthServiceClient) *CampaignService {
	service := NewCampaignService(stores)
	service.authClient = authClient
	return service
}

// CreateCampaign creates a new campaign metadata record.
func (s *CampaignService) CreateCampaign(ctx context.Context, in *campaignv1.CreateCampaignRequest) (*campaignv1.CreateCampaignResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create campaign request is required")
	}

	created, owner, err := newCampaignApplication(s).CreateCampaign(ctx, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.CreateCampaignResponse{
		Campaign:         campaignToProto(created),
		OwnerParticipant: participantToProto(owner),
	}, nil
}

// ListCampaigns returns a page of campaign metadata records.
// The web path needs a participant-scoped view quickly, so we filter by caller
// membership before sending a response and keep policy/intent checks explicit in
// later security work that is shared with MCP and future clients.
// The current behavior prefers caller participant_id when present and falls back to
// user_id for backward compatibility.
func (s *CampaignService) ListCampaigns(ctx context.Context, in *campaignv1.ListCampaignsRequest) (*campaignv1.ListCampaignsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list campaigns request is required")
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListCampaignsPageSize,
		Max:     maxListCampaignsPageSize,
	})

	// TODO: Apply access policy/intent gates for campaign listing.
	// Without this, user-scoped filtering should not be interpreted as a full
	// authorization decision for all clients and request contexts.
	// This is intentionally a membership-first filter until role/intent policy
	// evaluation is completed at a common boundary.
	participantID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
	userID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))

	campaignRecords := make([]storage.CampaignRecord, 0, pageSize)
	nextPageToken := ""
	var err error
	if participantID != "" && s.stores.Participant == nil {
		return nil, status.Error(codes.Internal, "participant store is not configured")
	}
	if participantID == "" && userID != "" && s.stores.Participant == nil {
		return nil, status.Error(codes.Internal, "participant store is not configured")
	}

	if participantID == "" && userID == "" {
		page, err := s.stores.Campaign.List(ctx, pageSize, in.GetPageToken())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "list campaigns: %v", err)
		}
		campaignRecords = page.Campaigns
		nextPageToken = page.NextPageToken
	} else if participantID != "" {
		campaignRecords, nextPageToken, err = s.listCampaignsForParticipant(ctx, participantID, pageSize, in.GetPageToken())
	} else {
		campaignRecords, nextPageToken, err = s.listCampaignsForUser(ctx, userID, pageSize, in.GetPageToken())
	}
	if err != nil {
		return nil, err
	}

	response := &campaignv1.ListCampaignsResponse{
		NextPageToken: nextPageToken,
	}
	if len(campaignRecords) == 0 {
		return response, nil
	}

	response.Campaigns = make([]*campaignv1.Campaign, 0, len(campaignRecords))
	for _, c := range campaignRecords {
		response.Campaigns = append(response.Campaigns, campaignToProto(c))
	}

	return response, nil
}

func (s *CampaignService) listCampaignsForUser(ctx context.Context, userID string, pageSize int, pageToken string) ([]storage.CampaignRecord, string, error) {
	userID = strings.TrimSpace(userID)
	campaignIDs, err := s.stores.Participant.ListCampaignIDsByUser(ctx, userID)
	if err != nil {
		return nil, "", status.Errorf(codes.Internal, "list campaign IDs by user: %v", err)
	}
	if len(campaignIDs) == 0 {
		return nil, "", nil
	}

	start := 0
	if pageToken != "" {
		for idx, campaignID := range campaignIDs {
			if strings.TrimSpace(campaignID) == pageToken {
				start = idx + 1
				break
			}
		}
	}
	if start < 0 || start >= len(campaignIDs) {
		start = 0
	}

	end := start + pageSize
	if end > len(campaignIDs) {
		end = len(campaignIDs)
	}

	campaignRecords := make([]storage.CampaignRecord, 0, end-start)
	for _, campaignID := range campaignIDs[start:end] {
		campaignID = strings.TrimSpace(campaignID)
		if campaignID == "" {
			continue
		}
		record, err := s.stores.Campaign.Get(ctx, campaignID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				continue
			}
			return nil, "", status.Errorf(codes.Internal, "get campaign: %v", err)
		}
		campaignRecords = append(campaignRecords, record)
	}

	nextPageToken := ""
	if end < len(campaignIDs) && end > 0 {
		nextPageToken = campaignIDs[end-1]
	}

	return campaignRecords, nextPageToken, nil
}

func (s *CampaignService) listCampaignsForParticipant(ctx context.Context, participantID string, pageSize int, pageToken string) ([]storage.CampaignRecord, string, error) {
	participantID = strings.TrimSpace(participantID)
	campaignIDs, err := s.stores.Participant.ListCampaignIDsByParticipant(ctx, participantID)
	if err != nil {
		return nil, "", status.Errorf(codes.Internal, "list campaign IDs by participant: %v", err)
	}
	if len(campaignIDs) == 0 {
		return nil, "", nil
	}

	start := 0
	if pageToken != "" {
		for idx, campaignID := range campaignIDs {
			if strings.TrimSpace(campaignID) == pageToken {
				start = idx + 1
				break
			}
		}
	}
	if start < 0 || start >= len(campaignIDs) {
		start = 0
	}

	end := start + pageSize
	if end > len(campaignIDs) {
		end = len(campaignIDs)
	}

	campaignRecords := make([]storage.CampaignRecord, 0, end-start)
	for _, campaignID := range campaignIDs[start:end] {
		campaignID = strings.TrimSpace(campaignID)
		if campaignID == "" {
			continue
		}
		record, err := s.stores.Campaign.Get(ctx, campaignID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				continue
			}
			return nil, "", status.Errorf(codes.Internal, "get campaign: %v", err)
		}
		campaignRecords = append(campaignRecords, record)
	}

	nextPageToken := ""
	if end < len(campaignIDs) && end > 0 {
		nextPageToken = campaignIDs[end-1]
	}

	return campaignRecords, nextPageToken, nil
}

// GetCampaign returns a campaign metadata record by ID.
// The domain layer enforces lifecycle validity, while broader policy/intent checks
// are deferred to dedicated access gates so one read model can serve all transport
// surfaces (gRPC, MCP, and web).
// Lifecycle validation is enforced at domain level now; broader access-policy checks
// (gating by intent/role) are intentionally still pending.
func (s *CampaignService) GetCampaign(ctx context.Context, in *campaignv1.GetCampaignRequest) (*campaignv1.GetCampaignResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get campaign request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	// TODO: Apply access policy/intent gates for campaign read.
	// Until policy checks are wired in, clients can retrieve campaign metadata
	// after record fetch and domain-state validation only.
	// Treat this endpoint as a domain-integrity boundary, not a full authorization
	// boundary, until policy checks are centralized.
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpRead); err != nil {
		return nil, handleDomainError(err)
	}

	return &campaignv1.GetCampaignResponse{
		Campaign: campaignToProto(c),
	}, nil
}

// EndCampaign marks a campaign as completed.
func (s *CampaignService) EndCampaign(ctx context.Context, in *campaignv1.EndCampaignRequest) (*campaignv1.EndCampaignResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "end campaign request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	updated, err := newCampaignApplication(s).EndCampaign(ctx, campaignID)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.EndCampaignResponse{Campaign: campaignToProto(updated)}, nil
}

// ArchiveCampaign archives a campaign.
func (s *CampaignService) ArchiveCampaign(ctx context.Context, in *campaignv1.ArchiveCampaignRequest) (*campaignv1.ArchiveCampaignResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "archive campaign request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	updated, err := newCampaignApplication(s).ArchiveCampaign(ctx, campaignID)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.ArchiveCampaignResponse{Campaign: campaignToProto(updated)}, nil
}

// RestoreCampaign restores an archived campaign to draft state.
func (s *CampaignService) RestoreCampaign(ctx context.Context, in *campaignv1.RestoreCampaignRequest) (*campaignv1.RestoreCampaignResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "restore campaign request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	updated, err := newCampaignApplication(s).RestoreCampaign(ctx, campaignID)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.RestoreCampaignResponse{Campaign: campaignToProto(updated)}, nil
}

func ensureNoActiveSession(ctx context.Context, store storage.SessionStore, campaignID string) error {
	if store == nil {
		return status.Error(codes.Internal, "session store is not configured")
	}
	_, err := store.GetActiveSession(ctx, campaignID)
	if err == nil {
		return apperrors.HandleError(storage.ErrActiveSessionExists, apperrors.DefaultLocale)
	}
	if errors.Is(err, storage.ErrNotFound) {
		return nil
	}
	return status.Errorf(codes.Internal, "check active session: %v", err)
}

// handleDomainError converts domain errors to gRPC status using the structured error system.
// For domain errors (*apperrors.Error), it returns a properly formatted gRPC status with
// error details including ErrorInfo and LocalizedMessage.
// For non-domain errors, it falls back to an internal error.
//
// TODO: Extract locale from gRPC metadata (e.g., "accept-language" header) to enable
// proper i18n support. Currently hardcoded to DefaultLocale.
//
// This keeps API responses deterministic today while leaving room for locale-aware
// user experience in follow-up work.
//
// The default locale is intentional so behavior is stable while auth/web and
// gRPC metadata propagation is still being aligned for user-facing localization.
func handleDomainError(err error) error {
	return apperrors.HandleError(err, apperrors.DefaultLocale)
}
