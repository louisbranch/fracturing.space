package app

// newService keeps a combined publicauth fixture seam available only to
// package tests. Production wiring should stay on the owned page/session/
// passkey/recovery constructors.
type testGateway interface {
	SessionGateway
	PasskeyGateway
	RecoveryGateway
}

func newService(gateway testGateway, authBaseURL string) testServiceBundle {
	return testServiceBundle{
		PageService:     NewPageService(authBaseURL),
		SessionService:  NewSessionService(gateway, authBaseURL),
		PasskeyService:  NewPasskeyService(gateway, authBaseURL),
		RecoveryService: NewRecoveryService(gateway, authBaseURL),
	}
}

type testServiceBundle struct {
	PageService
	SessionService
	PasskeyService
	RecoveryService
}
