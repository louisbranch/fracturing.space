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
// Admin override requests are allowed to enumerate campaigns without participant scope.
// Non-admin calls remain participant/user scoped and only return member campaigns.
func (s *CampaignService) ListCampaigns(ctx context.Context, in *campaignv1.ListCampaignsRequest) (*campaignv1.ListCampaignsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list campaigns request is required")
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListCampaignsPageSize,
		Max:     maxListCampaignsPageSize,
	})

	participantID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
	userID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	overrideReason, overrideRequested := adminOverrideFromContext(ctx)

	if overrideRequested {
		if overrideReason == "" {
			err := status.Error(codes.PermissionDenied, "admin override reason is required")
			emitAuthzDecisionTelemetry(ctx, s.stores.Telemetry, "", policyActionReadCampaign, authzDecisionDeny, authzReasonDenyOverrideReasonRequired, storage.ParticipantRecord{}, err, nil)
			return nil, err
		}
		emitAuthzDecisionTelemetry(ctx, s.stores.Telemetry, "", policyActionReadCampaign, authzDecisionAllow, authzReasonAllowAdminOverride, storage.ParticipantRecord{}, nil, nil)
		page, err := s.stores.Campaign.List(ctx, pageSize, in.GetPageToken())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "list campaigns: %v", err)
		}
		response := &campaignv1.ListCampaignsResponse{NextPageToken: page.NextPageToken}
		if len(page.Campaigns) == 0 {
			return response, nil
		}
		response.Campaigns = make([]*campaignv1.Campaign, 0, len(page.Campaigns))
		for _, c := range page.Campaigns {
			response.Campaigns = append(response.Campaigns, campaignToProto(c))
		}
		return response, nil
	}

	if participantID == "" && userID == "" {
		err := status.Error(codes.PermissionDenied, "missing participant identity")
		emitAuthzDecisionTelemetry(ctx, s.stores.Telemetry, "", policyActionReadCampaign, authzDecisionDeny, authzReasonDenyMissingIdentity, storage.ParticipantRecord{}, err, nil)
		return nil, err
	}
	if s.stores.Participant == nil {
		return nil, status.Error(codes.Internal, "participant store is not configured")
	}

	campaignRecords := make([]storage.CampaignRecord, 0, pageSize)
	nextPageToken := ""
	var err error
	if participantID != "" {
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
	return s.listCampaignsByIDs(ctx, campaignIDs, pageSize, pageToken)
}

func (s *CampaignService) listCampaignsForParticipant(ctx context.Context, participantID string, pageSize int, pageToken string) ([]storage.CampaignRecord, string, error) {
	participantID = strings.TrimSpace(participantID)
	campaignIDs, err := s.stores.Participant.ListCampaignIDsByParticipant(ctx, participantID)
	if err != nil {
		return nil, "", status.Errorf(codes.Internal, "list campaign IDs by participant: %v", err)
	}
	return s.listCampaignsByIDs(ctx, campaignIDs, pageSize, pageToken)
}

// listCampaignsByIDs paginates a pre-resolved list of campaign IDs and fetches
// the corresponding records, skipping any that have been deleted.
func (s *CampaignService) listCampaignsByIDs(ctx context.Context, campaignIDs []string, pageSize int, pageToken string) ([]storage.CampaignRecord, string, error) {
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
// Lifecycle validation and read-policy checks are enforced so one read model
// can serve all transport surfaces (gRPC, MCP, and web).
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
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpRead); err != nil {
		return nil, handleDomainError(err)
	}
	if err := requireReadPolicy(ctx, s.stores, c); err != nil {
		return nil, err
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

// SetCampaignCover updates the selected built-in campaign cover.
func (s *CampaignService) SetCampaignCover(ctx context.Context, in *campaignv1.SetCampaignCoverRequest) (*campaignv1.SetCampaignCoverResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "set campaign cover request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	coverAssetID := strings.TrimSpace(in.GetCoverAssetId())
	if coverAssetID == "" {
		return nil, status.Error(codes.InvalidArgument, "cover asset id is required")
	}
	coverSetID := strings.TrimSpace(in.GetCoverSetId())

	updated, err := newCampaignApplication(s).SetCampaignCover(ctx, campaignID, coverAssetID, coverSetID)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.SetCampaignCoverResponse{Campaign: campaignToProto(updated)}, nil
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
