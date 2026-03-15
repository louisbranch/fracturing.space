package app

import (
	"fmt"
	"log/slog"
	"net/http"
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

func writeRPCError(w http.ResponseWriter, err error) {
	if w == nil {
		return
	}
	if err == nil {
		writeJSONError(w, http.StatusInternalServerError, "unknown error")
		return
	}
	code := gogrpcstatus.Code(err)
	switch code {
	case gogrpccodes.InvalidArgument:
		writeJSONError(w, http.StatusBadRequest, gogrpcstatus.Convert(err).Message())
	case gogrpccodes.PermissionDenied:
		writeJSONError(w, http.StatusForbidden, gogrpcstatus.Convert(err).Message())
	case gogrpccodes.NotFound:
		writeJSONError(w, http.StatusNotFound, gogrpcstatus.Convert(err).Message())
	case gogrpccodes.FailedPrecondition, gogrpccodes.Aborted:
		writeJSONError(w, http.StatusConflict, gogrpcstatus.Convert(err).Message())
	case gogrpccodes.Unauthenticated:
		writeJSONError(w, http.StatusUnauthorized, gogrpcstatus.Convert(err).Message())
	default:
		writeJSONError(w, http.StatusBadGateway, "upstream request failed")
	}
}

func parseInt64(value string) (int64, error) {
	var parsed int64
	_, err := fmt.Sscan(strings.TrimSpace(value), &parsed)
	return parsed, err
}

func parseInt(value string) (int, error) {
	var parsed int
	_, err := fmt.Sscan(strings.TrimSpace(value), &parsed)
	return parsed, err
}

func pathForCampaignAPI(campaignID string, suffix string) string {
	campaignID = strings.TrimSpace(campaignID)
	suffix = strings.Trim(strings.TrimSpace(suffix), "/")
	if suffix == "" {
		return "/api/campaigns/" + campaignID
	}
	return "/api/campaigns/" + campaignID + "/" + suffix
}
