package app

import (
	"net/http"

	gogrpccodes "google.golang.org/grpc/codes"
	gogrpcstatus "google.golang.org/grpc/status"
)

// rpcErrorMessages maps gRPC status codes to safe, user-facing error messages
// so upstream implementation details (field names, proto paths) are never
// leaked to the browser.
var rpcErrorMessages = map[gogrpccodes.Code]struct {
	httpStatus int
	message    string
}{
	gogrpccodes.InvalidArgument:    {http.StatusBadRequest, "invalid request"},
	gogrpccodes.PermissionDenied:   {http.StatusForbidden, "permission denied"},
	gogrpccodes.NotFound:           {http.StatusNotFound, "resource not found"},
	gogrpccodes.FailedPrecondition: {http.StatusConflict, "action not allowed in current state"},
	gogrpccodes.Aborted:            {http.StatusConflict, "action not allowed in current state"},
	gogrpccodes.Unauthenticated:    {http.StatusUnauthorized, "authentication required"},
	gogrpccodes.ResourceExhausted:  {http.StatusTooManyRequests, "too many requests"},
}

func writeRPCError(w http.ResponseWriter, err error) {
	if w == nil {
		return
	}
	if err == nil {
		writeJSONError(w, http.StatusInternalServerError, "unknown error")
		return
	}
	code := gogrpcstatus.Code(err)
	if mapped, ok := rpcErrorMessages[code]; ok {
		writeJSONError(w, mapped.httpStatus, mapped.message)
		return
	}
	writeJSONError(w, http.StatusBadGateway, "upstream request failed")
}
