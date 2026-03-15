package discovery

import (
	"context"
	"errors"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/discovery/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultListDiscoveryEntriesPageSize = 10
	maxListDiscoveryEntriesPageSize     = 50
)

// Service exposes discovery.v1 gRPC operations.
type Service struct {
	discoveryv1.UnimplementedDiscoveryServiceServer
	store storage.DiscoveryEntryStore
	clock func() time.Time
}

// NewService creates a discovery service backed by discovery entry storage.
func NewService(store storage.DiscoveryEntryStore) *Service {
	return &Service{
		store: store,
		clock: time.Now,
	}
}

// CreateDiscoveryEntry creates one discovery entry record.
func (s *Service) CreateDiscoveryEntry(ctx context.Context, in *discoveryv1.CreateDiscoveryEntryRequest) (*discoveryv1.CreateDiscoveryEntryResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create discovery entry request is required")
	}
	if s == nil || s.store == nil {
		return nil, status.Error(codes.Internal, "discovery entry store is not configured")
	}
	if in.GetEntry() == nil {
		return nil, status.Error(codes.InvalidArgument, "entry is required")
	}

	entry := in.GetEntry()
	entryID := strings.TrimSpace(entry.GetEntryId())
	title := strings.TrimSpace(entry.GetTitle())
	description := strings.TrimSpace(entry.GetDescription())
	expectedDurationLabel := strings.TrimSpace(entry.GetExpectedDurationLabel())
	if entryID == "" {
		return nil, status.Error(codes.InvalidArgument, "entry id is required")
	}
	if entry.GetKind() == discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "entry kind is required")
	}
	if title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	if description == "" {
		return nil, status.Error(codes.InvalidArgument, "description is required")
	}
	if expectedDurationLabel == "" {
		return nil, status.Error(codes.InvalidArgument, "expected duration label is required")
	}
	if entry.GetDifficultyTier() == discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "difficulty tier is required")
	}
	if entry.GetSystem() == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "game system is required")
	}
	if entry.GetRecommendedParticipantsMin() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "recommended participants min must be greater than zero")
	}
	if entry.GetRecommendedParticipantsMax() < entry.GetRecommendedParticipantsMin() {
		return nil, status.Error(codes.InvalidArgument, "recommended participants max must be greater than or equal to min")
	}

	now := time.Now().UTC()
	if s.clock != nil {
		now = s.clock().UTC()
	}
	record := storage.DiscoveryEntry{
		EntryID:                    entryID,
		Kind:                       entry.GetKind(),
		SourceID:                   strings.TrimSpace(entry.GetSourceId()),
		Title:                      title,
		Description:                description,
		CampaignTheme:              strings.TrimSpace(entry.GetCampaignTheme()),
		RecommendedParticipantsMin: int(entry.GetRecommendedParticipantsMin()),
		RecommendedParticipantsMax: int(entry.GetRecommendedParticipantsMax()),
		DifficultyTier:             entry.GetDifficultyTier(),
		ExpectedDurationLabel:      expectedDurationLabel,
		System:                     entry.GetSystem(),
		GmMode:                     entry.GetGmMode(),
		Intent:                     entry.GetIntent(),
		Level:                      int(entry.GetLevel()),
		CharacterCount:             int(entry.GetCharacterCount()),
		Storyline:                  strings.TrimSpace(entry.GetStoryline()),
		Tags:                       entry.GetTags(),
		PreviewHook:                strings.TrimSpace(entry.GetPreviewHook()),
		PreviewPlaystyleLabel:      strings.TrimSpace(entry.GetPreviewPlaystyleLabel()),
		PreviewCharacterName:       strings.TrimSpace(entry.GetPreviewCharacterName()),
		PreviewCharacterSummary:    strings.TrimSpace(entry.GetPreviewCharacterSummary()),
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}
	if err := s.store.CreateDiscoveryEntry(ctx, record); err != nil {
		if errors.Is(err, storage.ErrAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "discovery entry already exists")
		}
		return nil, status.Errorf(codes.Internal, "create discovery entry: %v", err)
	}
	return &discoveryv1.CreateDiscoveryEntryResponse{Entry: discoveryEntryToProto(record)}, nil
}

// GetDiscoveryEntry returns one discovery entry record by entry ID.
func (s *Service) GetDiscoveryEntry(ctx context.Context, in *discoveryv1.GetDiscoveryEntryRequest) (*discoveryv1.GetDiscoveryEntryResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get discovery entry request is required")
	}
	if s == nil || s.store == nil {
		return nil, status.Error(codes.Internal, "discovery entry store is not configured")
	}
	entryID := strings.TrimSpace(in.GetEntryId())
	if entryID == "" {
		return nil, status.Error(codes.InvalidArgument, "entry id is required")
	}

	record, err := s.store.GetDiscoveryEntry(ctx, entryID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "discovery entry not found")
		}
		return nil, status.Errorf(codes.Internal, "get discovery entry: %v", err)
	}
	return &discoveryv1.GetDiscoveryEntryResponse{Entry: discoveryEntryToProto(record)}, nil
}

// ListDiscoveryEntries returns a page of discovery entry records.
func (s *Service) ListDiscoveryEntries(ctx context.Context, in *discoveryv1.ListDiscoveryEntriesRequest) (*discoveryv1.ListDiscoveryEntriesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list discovery entries request is required")
	}
	if s == nil || s.store == nil {
		return nil, status.Error(codes.Internal, "discovery entry store is not configured")
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListDiscoveryEntriesPageSize,
		Max:     maxListDiscoveryEntriesPageSize,
	})
	page, err := s.store.ListDiscoveryEntries(ctx, pageSize, in.GetPageToken(), in.GetKind())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list discovery entries: %v", err)
	}

	resp := &discoveryv1.ListDiscoveryEntriesResponse{
		Entries:       make([]*discoveryv1.DiscoveryEntry, 0, len(page.Entries)),
		NextPageToken: page.NextPageToken,
	}
	for _, entry := range page.Entries {
		resp.Entries = append(resp.Entries, discoveryEntryToProto(entry))
	}
	return resp, nil
}

func discoveryEntryToProto(entry storage.DiscoveryEntry) *discoveryv1.DiscoveryEntry {
	return &discoveryv1.DiscoveryEntry{
		EntryId:                    entry.EntryID,
		Kind:                       entry.Kind,
		SourceId:                   entry.SourceID,
		Title:                      entry.Title,
		Description:                entry.Description,
		CampaignTheme:              entry.CampaignTheme,
		RecommendedParticipantsMin: int32(entry.RecommendedParticipantsMin),
		RecommendedParticipantsMax: int32(entry.RecommendedParticipantsMax),
		DifficultyTier:             entry.DifficultyTier,
		ExpectedDurationLabel:      entry.ExpectedDurationLabel,
		System:                     entry.System,
		GmMode:                     entry.GmMode,
		Intent:                     entry.Intent,
		Level:                      int32(entry.Level),
		CharacterCount:             int32(entry.CharacterCount),
		Storyline:                  entry.Storyline,
		Tags:                       entry.Tags,
		PreviewHook:                entry.PreviewHook,
		PreviewPlaystyleLabel:      entry.PreviewPlaystyleLabel,
		PreviewCharacterName:       entry.PreviewCharacterName,
		PreviewCharacterSummary:    entry.PreviewCharacterSummary,
		CreatedAt:                  timestamppb.New(entry.CreatedAt),
		UpdatedAt:                  timestamppb.New(entry.UpdatedAt),
	}
}
