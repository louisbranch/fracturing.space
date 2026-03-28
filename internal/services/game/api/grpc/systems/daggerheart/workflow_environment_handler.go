package daggerheart

import "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/environmenttransport"

func (s *DaggerheartService) environmentHandler() *environmenttransport.Handler {
	return environmenttransport.NewHandler(environmenttransport.Dependencies{
		Campaign:             s.stores.Campaign,
		Session:              s.stores.Session,
		Gate:                 s.stores.SessionGate,
		Daggerheart:          s.stores.Daggerheart,
		Content:              s.stores.Content,
		ExecuteDomainCommand: s.executeWorkflowDomainCommand,
	})
}
