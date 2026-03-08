package templates

import "net/http"

const (
	appErrorPageTitleNotFoundKey   = "web.error.page_title_not_found"
	appErrorPageTitleClientErrKey  = "web.error.page_title_client_error"
	appErrorPageTitleServerErrKey  = "web.error.page_title_server_error"
	appErrorHeadingNotFoundKey     = "web.error.title_not_found"
	appErrorHeadingClientErrKey    = "web.error.title_client_error"
	appErrorHeadingServerErrKey    = "web.error.title_server_error"
	appErrorMessageNotFoundKey     = "web.error.message_not_found"
	appErrorMessageClientErrKey    = "web.error.message_client_error"
	appErrorMessageServerErrKey    = "web.error.message_server_error"
	appErrorBackToDashboardTextKey = "web.error.action_back_to_dashboard"
)

// AppErrorPageTitle returns the browser page title for app error pages.
func AppErrorPageTitle(statusCode int, loc Localizer) string {
	switch normalizeAppErrorStatus(statusCode) {
	case http.StatusNotFound:
		return T(loc, appErrorPageTitleNotFoundKey)
	case http.StatusBadRequest:
		return T(loc, appErrorPageTitleClientErrKey)
	default:
		return T(loc, appErrorPageTitleServerErrKey)
	}
}

// appErrorHeading centralizes this web behavior in one helper seam.
func appErrorHeading(statusCode int, loc Localizer) string {
	switch normalizeAppErrorStatus(statusCode) {
	case http.StatusNotFound:
		return T(loc, appErrorHeadingNotFoundKey)
	case http.StatusBadRequest:
		return T(loc, appErrorHeadingClientErrKey)
	default:
		return T(loc, appErrorHeadingServerErrKey)
	}
}

// appErrorMessage centralizes this web behavior in one helper seam.
func appErrorMessage(statusCode int, loc Localizer) string {
	switch normalizeAppErrorStatus(statusCode) {
	case http.StatusNotFound:
		return T(loc, appErrorMessageNotFoundKey)
	case http.StatusBadRequest:
		return T(loc, appErrorMessageClientErrKey)
	default:
		return T(loc, appErrorMessageServerErrKey)
	}
}

// normalizeAppErrorStatus buckets HTTP status codes into display categories:
// 404 (not found), 400 (client errors), and 500 (server errors).
func normalizeAppErrorStatus(statusCode int) int {
	switch {
	case statusCode == http.StatusNotFound:
		return http.StatusNotFound
	case statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
