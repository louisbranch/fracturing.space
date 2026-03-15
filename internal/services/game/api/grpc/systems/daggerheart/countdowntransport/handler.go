package countdowntransport

// Handler owns the Daggerheart countdown mutation transport endpoints.
type Handler struct {
	deps Dependencies
}

// NewHandler builds a countdown mutation transport handler from explicit read
// stores, ID generation, and write-callback dependencies.
func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}
