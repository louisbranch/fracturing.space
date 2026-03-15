package conditiontransport

// Handler owns the Daggerheart condition and life-state mutation transport
// endpoints.
type Handler struct {
	deps Dependencies
}

// NewHandler builds a Daggerheart condition transport handler from explicit
// read-store and write-callback dependencies.
func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}
