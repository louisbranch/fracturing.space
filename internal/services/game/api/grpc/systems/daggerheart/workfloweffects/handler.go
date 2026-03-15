package workfloweffects

type Handler struct {
	deps Dependencies
}

func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}
