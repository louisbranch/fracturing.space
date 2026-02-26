package templates

import "net/http"

const (
	appErrorPageTitleNotFoundKey   = "web.error.page_title_not_found"
	appErrorPageTitleServerErrKey  = "web.error.page_title_server_error"
	appErrorHeadingNotFoundKey     = "web.error.title_not_found"
	appErrorHeadingServerErrKey    = "web.error.title_server_error"
	appErrorMessageNotFoundKey     = "web.error.message_not_found"
	appErrorMessageServerErrKey    = "web.error.message_server_error"
	appErrorBackToDashboardTextKey = "web.error.action_back_to_dashboard"
)

// AppErrorPageTitle returns the browser page title for app error pages.
func AppErrorPageTitle(statusCode int, loc Localizer) string {
	if normalizeAppErrorStatus(statusCode) == http.StatusNotFound {
		return T(loc, appErrorPageTitleNotFoundKey)
	}
	return T(loc, appErrorPageTitleServerErrKey)
}

func appErrorHeading(statusCode int, loc Localizer) string {
	if normalizeAppErrorStatus(statusCode) == http.StatusNotFound {
		return T(loc, appErrorHeadingNotFoundKey)
	}
	return T(loc, appErrorHeadingServerErrKey)
}

func appErrorMessage(statusCode int, loc Localizer) string {
	if normalizeAppErrorStatus(statusCode) == http.StatusNotFound {
		return T(loc, appErrorMessageNotFoundKey)
	}
	return T(loc, appErrorMessageServerErrKey)
}

func normalizeAppErrorStatus(statusCode int) int {
	if statusCode == http.StatusNotFound {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}
