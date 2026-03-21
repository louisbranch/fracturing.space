package app

import (
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	gogrpccodes "google.golang.org/grpc/codes"
	gogrpcstatus "google.golang.org/grpc/status"
)

func loggerOrDefault(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		return slog.Default()
	}
	return logger
}

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

func parseInt64(value string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(value), 10, 64)
}

func parseInt(value string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(value))
}

func pathForCampaignAPI(campaignID string, suffix string) string {
	campaignID = url.PathEscape(strings.TrimSpace(campaignID))
	suffix = strings.Trim(strings.TrimSpace(suffix), "/")
	if suffix == "" {
		return "/api/campaigns/" + campaignID
	}
	return "/api/campaigns/" + campaignID + "/" + suffix
}
