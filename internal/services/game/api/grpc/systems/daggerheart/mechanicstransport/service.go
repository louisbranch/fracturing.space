package mechanicstransport

// Handler owns Daggerheart deterministic mechanics and read-only roll helpers.
//
// The root Daggerheart gRPC package keeps the public constructor and service
// registration surface stable, while this package owns the mechanics-specific
// transport implementation.
type Handler struct {
	seedFunc func() (int64, error)
}

// NewHandler binds deterministic mechanics transport to one seed source.
func NewHandler(seedFunc func() (int64, error)) *Handler {
	return &Handler{seedFunc: seedFunc}
}
