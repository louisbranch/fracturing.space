package daggerheart

import "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/adversarytransport"

func (s *DaggerheartService) adversaryHandler() *adversarytransport.Handler {
	return adversarytransport.NewHandler(adversarytransport.Dependencies{
		Campaign:             s.stores.Campaign,
		Session:              s.stores.Session,
		Gate:                 s.stores.SessionGate,
		Daggerheart:          s.stores.Daggerheart,
		Content:              s.stores.Content,
		ExecuteDomainCommand: s.executeWorkflowDomainCommand,
	})
}
