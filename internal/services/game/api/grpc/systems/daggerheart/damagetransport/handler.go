package damagetransport

// Handler owns the Daggerheart damage-application transport endpoints.
type Handler struct {
	deps Dependencies
}

// NewHandler builds a Daggerheart damage-application transport handler from
// explicit read-store and write-callback dependencies.
func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}
