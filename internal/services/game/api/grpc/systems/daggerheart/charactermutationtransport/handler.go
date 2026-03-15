package charactermutationtransport

// Handler owns the Daggerheart character progression and inventory transport
// endpoints.
type Handler struct {
	deps Dependencies
}

// NewHandler builds a character mutation transport handler from explicit
// campaign/profile reads and a character-command callback.
func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}
