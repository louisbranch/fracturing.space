package sessionrolltransport

// Handler owns the low-level Daggerheart session roll endpoints.
type Handler struct {
	deps Dependencies
}

// NewHandler builds a low-level Daggerheart session roll handler from explicit
// read-store and write-callback dependencies.
func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}
