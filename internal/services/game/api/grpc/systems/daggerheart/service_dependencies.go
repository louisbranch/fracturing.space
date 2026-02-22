package daggerheart

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type serviceDependency int

const (
	dependencyCampaignStore serviceDependency = iota
	dependencyCharacterStore
	dependencySessionStore
	dependencySessionGateStore
	dependencySessionSpotlightStore
	dependencyDaggerheartStore
	dependencyEventStore
	dependencyDomainEngine
	dependencySeedGenerator
)

func (s *DaggerheartService) requireDependencies(dependencies ...serviceDependency) error {
	for _, dependency := range dependencies {
		if s.hasDependency(dependency) {
			continue
		}
		return status.Error(codes.Internal, missingDependencyMessage(dependency))
	}
	return nil
}

func (s *DaggerheartService) hasDependency(dependency serviceDependency) bool {
	switch dependency {
	case dependencyCampaignStore:
		return s.stores.Campaign != nil
	case dependencyCharacterStore:
		return s.stores.Character != nil
	case dependencySessionStore:
		return s.stores.Session != nil
	case dependencySessionGateStore:
		return s.stores.SessionGate != nil
	case dependencySessionSpotlightStore:
		return s.stores.SessionSpotlight != nil
	case dependencyDaggerheartStore:
		return s.stores.Daggerheart != nil
	case dependencyEventStore:
		return s.stores.Event != nil
	case dependencyDomainEngine:
		return s.stores.Domain != nil
	case dependencySeedGenerator:
		return s.seedFunc != nil
	default:
		return false
	}
}

func missingDependencyMessage(dependency serviceDependency) string {
	switch dependency {
	case dependencyCampaignStore:
		return "campaign store is not configured"
	case dependencyCharacterStore:
		return "character store is not configured"
	case dependencySessionStore:
		return "session store is not configured"
	case dependencySessionGateStore:
		return "session gate store is not configured"
	case dependencySessionSpotlightStore:
		return "session spotlight store is not configured"
	case dependencyDaggerheartStore:
		return "daggerheart store is not configured"
	case dependencyEventStore:
		return "event store is not configured"
	case dependencyDomainEngine:
		return "domain engine is not configured"
	case dependencySeedGenerator:
		return "seed generator is not configured"
	default:
		return "service dependency is not configured"
	}
}
