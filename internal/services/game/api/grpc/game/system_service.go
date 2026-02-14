package game

import (
	"context"
	"sort"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SystemService implements the game.v1.SystemService gRPC API.
type SystemService struct {
	gamev1.UnimplementedSystemServiceServer
	registry *systems.Registry
}

// NewSystemService creates a SystemService backed by the registry.
func NewSystemService(registry *systems.Registry) *SystemService {
	if registry == nil {
		registry = systems.DefaultRegistry
	}
	return &SystemService{registry: registry}
}

// ListGameSystems returns all registered game systems.
func (s *SystemService) ListGameSystems(ctx context.Context, in *gamev1.ListGameSystemsRequest) (*gamev1.ListGameSystemsResponse, error) {
	_ = ctx
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list game systems request is required")
	}
	if s.registry == nil {
		return nil, status.Error(codes.Internal, "game system registry is not configured")
	}

	systemsList := s.registry.List()
	defaultVersions := make(map[commonv1.GameSystem]string, len(systemsList))
	for _, system := range systemsList {
		if _, ok := defaultVersions[system.ID()]; !ok {
			defaultVersions[system.ID()] = s.registry.DefaultVersion(system.ID())
		}
	}

	sort.Slice(systemsList, func(i, j int) bool {
		left := systemsList[i]
		right := systemsList[j]
		if left.ID() == right.ID() {
			return left.Version() < right.Version()
		}
		return left.ID() < right.ID()
	})

	response := &gamev1.ListGameSystemsResponse{
		Systems: make([]*gamev1.GameSystemInfo, 0, len(systemsList)),
	}
	for _, system := range systemsList {
		info := systemToProto(system, defaultVersions[system.ID()])
		response.Systems = append(response.Systems, info)
	}

	return response, nil
}

// GetGameSystem returns a registered game system by ID and optional version.
func (s *SystemService) GetGameSystem(ctx context.Context, in *gamev1.GetGameSystemRequest) (*gamev1.GetGameSystemResponse, error) {
	_ = ctx
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get game system request is required")
	}
	if in.GetId() == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "game system id is required")
	}
	if s.registry == nil {
		return nil, status.Error(codes.Internal, "game system registry is not configured")
	}

	system := s.registry.GetVersion(in.GetId(), in.GetVersion())
	if system == nil {
		return nil, status.Error(codes.NotFound, "game system not registered")
	}
	defaultVersion := s.registry.DefaultVersion(in.GetId())

	return &gamev1.GetGameSystemResponse{
		System: systemToProto(system, defaultVersion),
	}, nil
}

func systemToProto(system systems.GameSystem, defaultVersion string) *gamev1.GameSystemInfo {
	metadata := system.RegistryMetadata()
	version := strings.TrimSpace(system.Version())
	return &gamev1.GameSystemInfo{
		Id:                  system.ID(),
		Name:                system.Name(),
		Version:             version,
		ImplementationStage: metadata.ImplementationStage,
		OperationalStatus:   metadata.OperationalStatus,
		AccessLevel:         metadata.AccessLevel,
		IsDefault:           version != "" && version == strings.TrimSpace(defaultVersion),
		Notes:               metadata.Notes,
	}
}
