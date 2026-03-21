package app

// unavailableGateway is the fail-closed gateway used when composition omits one
// or more settings dependencies.
type unavailableGateway struct{}

// NewUnavailableGateway returns a fail-closed settings gateway.
func NewUnavailableGateway() unavailableGateway {
	return unavailableGateway{}
}
