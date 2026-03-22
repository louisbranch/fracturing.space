package countdowntransport

// Handler owns the Daggerheart countdown read and mutation transport endpoints.
type Handler struct {
	deps Dependencies
}

// NewHandler builds a countdown transport handler from explicit read stores,
// ID generation, and write-callback dependencies.
func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}
