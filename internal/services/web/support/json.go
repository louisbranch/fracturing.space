package support

import (
	"encoding/json"
	"net/http"
)

// WriteJSON writes a JSON body with normalized headers and status.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
