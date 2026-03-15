package sessionflowtransport

// Handler owns the Daggerheart session gameplay flow transport surface behind a
// narrow callback-based seam.
type Handler struct {
	deps Dependencies
}

// NewHandler builds a session flow transport handler.
func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}
