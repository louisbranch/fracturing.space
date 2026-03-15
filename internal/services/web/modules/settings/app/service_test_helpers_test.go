package app

// newService keeps a combined-gateway fixture seam available only to package
// tests. Production settings app wiring should stay explicit by owned surface.
type testGateway interface {
	AccountGateway
	AIGateway
}

func newService(gateway testGateway) service {
	if gateway == nil {
		return newServiceFromConfig(serviceConfig{})
	}
	return newServiceFromConfig(serviceConfig{
		ProfileGateway:  gateway,
		LocaleGateway:   gateway,
		SecurityGateway: gateway,
		AIKeyGateway:    gateway,
		AIAgentGateway:  gateway,
	})
}
