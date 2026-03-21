package app

// startupRollback tracks startup-time resources that should be closed when
// bootstrap fails. Closers run in reverse registration order.
type startupRollback struct {
	closers []func()
}

func (r *startupRollback) add(closeFn func()) {
	if closeFn == nil {
		return
	}
	r.closers = append(r.closers, closeFn)
}

func (r *startupRollback) cleanup() {
	for idx := len(r.closers) - 1; idx >= 0; idx-- {
		r.closers[idx]()
	}
	r.closers = nil
}

func (r *startupRollback) release() {
	r.closers = nil
}
