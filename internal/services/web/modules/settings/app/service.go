package app

// service defines an internal contract used at this web package boundary.
type service struct {
	gateway Gateway
}

// NewService constructs a settings service with fail-closed gateway defaults.
func NewService(gateway Gateway) Service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return service{gateway: gateway}
}
