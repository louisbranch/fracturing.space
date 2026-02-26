package routing

import "net/http"

// StringParamHandler is a handler callback that receives one parsed path value.
type StringParamHandler func(http.ResponseWriter, *http.Request, string)

// CallOrNotFound invokes a handler callback when present or responds with 404.
func CallOrNotFound(w http.ResponseWriter, r *http.Request, fn http.HandlerFunc) {
	if fn == nil {
		http.NotFound(w, r)
		return
	}
	fn(w, r)
}

// CallStringOrNotFound invokes a path-parameter handler when present or responds with 404.
func CallStringOrNotFound(w http.ResponseWriter, r *http.Request, fn StringParamHandler, value string) {
	if fn == nil {
		http.NotFound(w, r)
		return
	}
	fn(w, r, value)
}
