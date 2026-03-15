package app

// newService keeps a combined publicauth fixture seam available only to
// package tests. Production wiring should stay on the owned page/session/
// passkey/recovery constructors.
type testGateway interface {
	SessionGateway
	PasskeyGateway
	RecoveryGateway
}

func newService(gateway testGateway, authBaseURL string) service {
	return newServiceState(serviceConfig{
		SessionGateway:  gateway,
		PasskeyGateway:  gateway,
		RecoveryGateway: gateway,
		AuthBaseURL:     authBaseURL,
	})
}
