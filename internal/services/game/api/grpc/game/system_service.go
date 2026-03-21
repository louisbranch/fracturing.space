package game

import (
	"context"
	"sort"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SystemService implements the game.v1.SystemService gRPC API.
type SystemService struct {
	gamev1.UnimplementedSystemServiceServer
	registry *bridge.MetadataRegistry
}

// NewSystemService creates a SystemService backed by the registry.
func NewSystemService(registry *bridge.MetadataRegistry) *SystemService {
	return &SystemService{registry: registry}
}

// ListGameSystems returns all registered game bridge.
func (s *SystemService) ListGameSystems(ctx context.Context, in *gamev1.ListGameSystemsRequest) (*gamev1.ListGameSystemsResponse, error) {
	_ = ctx
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list game systems request is required")
	}
	if s.registry == nil {
		return nil, status.Error(codes.Internal, "game system registry is not configured")
	}

	systemsList := s.registry.List()
	defaultVersions := make(map[bridge.SystemID]string, len(systemsList))
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
	systemID, ok := systemIDFromProto(in.GetId())
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "game system id is required")
	}
	if s.registry == nil {
		return nil, status.Error(codes.Internal, "game system registry is not configured")
	}

	system := s.registry.GetVersion(systemID, in.GetVersion())
	if system == nil {
		return nil, status.Error(codes.NotFound, "game system not registered")
	}
	defaultVersion := s.registry.DefaultVersion(systemID)

	return &gamev1.GetGameSystemResponse{
		System: systemToProto(system, defaultVersion),
	}, nil
}

func systemToProto(system bridge.GameSystem, defaultVersion string) *gamev1.GameSystemInfo {
	metadata := system.RegistryMetadata()
	version := strings.TrimSpace(system.Version())
	return &gamev1.GameSystemInfo{
		Id:                  systemIDToProto(system.ID()),
		Name:                system.Name(),
		Version:             version,
		ImplementationStage: implementationStageToProto(metadata.ImplementationStage),
		OperationalStatus:   operationalStatusToProto(metadata.OperationalStatus),
		AccessLevel:         accessLevelToProto(metadata.AccessLevel),
		IsDefault:           version != "" && version == strings.TrimSpace(defaultVersion),
		Notes:               metadata.Notes,
	}
}

func systemIDFromProto(value commonv1.GameSystem) (bridge.SystemID, bool) {
	switch value {
	case commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART:
		return bridge.SystemIDDaggerheart, true
	default:
		return bridge.SystemIDUnspecified, false
	}
}

func systemIDToProto(value bridge.SystemID) commonv1.GameSystem {
	switch value {
	case bridge.SystemIDDaggerheart:
		return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART
	default:
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED
	}
}

func implementationStageToProto(value bridge.ImplementationStage) commonv1.GameSystemImplementationStage {
	switch value {
	case bridge.ImplementationStagePlanned:
		return commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_PLANNED
	case bridge.ImplementationStagePartial:
		return commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_PARTIAL
	case bridge.ImplementationStageComplete:
		return commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_COMPLETE
	case bridge.ImplementationStageDeprecated:
		return commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_DEPRECATED
	default:
		return commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_UNSPECIFIED
	}
}

func operationalStatusToProto(value bridge.OperationalStatus) commonv1.GameSystemOperationalStatus {
	switch value {
	case bridge.OperationalStatusOffline:
		return commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OFFLINE
	case bridge.OperationalStatusDegraded:
		return commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_DEGRADED
	case bridge.OperationalStatusOperational:
		return commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OPERATIONAL
	case bridge.OperationalStatusMaintenance:
		return commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_MAINTENANCE
	default:
		return commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_UNSPECIFIED
	}
}

func accessLevelToProto(value bridge.AccessLevel) commonv1.GameSystemAccessLevel {
	switch value {
	case bridge.AccessLevelInternal:
		return commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_INTERNAL
	case bridge.AccessLevelBeta:
		return commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_BETA
	case bridge.AccessLevelPublic:
		return commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_PUBLIC
	case bridge.AccessLevelRetired:
		return commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_RETIRED
	default:
		return commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_UNSPECIFIED
	}
}
