package app

// newService keeps a combined-gateway fixture seam available only to package
// tests. Production settings app wiring should stay explicit by owned surface.
type testGateway interface {
	AccountGateway
	AIGateway
}

func newService(gateway testGateway) testServiceBundle {
	return testServiceBundle{
		AccountService: NewAccountService(AccountServiceConfig{
			ProfileGateway:  gateway,
			LocaleGateway:   gateway,
			SecurityGateway: gateway,
		}),
		AIService: NewAIService(AIServiceConfig{
			AIKeyGateway:   gateway,
			AIAgentGateway: gateway,
		}),
	}
}

type testServiceBundle struct {
	AccountService
	AIService
}
