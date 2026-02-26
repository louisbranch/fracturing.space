package discovery

import "net/http"

type handlers struct {
	service service
}

func newHandlers(s service) handlers {
	return handlers{service: s}
}

func (h handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	// TODO(web-parity): replace raw text scaffold response with shared pagerender/weberror app shell flow.
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(h.service.body()))
}
