package gmmovetransport

// Handler owns the Daggerheart GM-move transport endpoint.
type Handler struct {
	deps Dependencies
}

// NewHandler builds a Daggerheart GM-move transport handler from explicit
// read-store and write-callback dependencies.
func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}
