package web

import "errors"

var (
	errRecipientUsernameRequired = errors.New("recipient username is required")
	errConnectionsUnavailable    = errors.New("connections service is not configured")
	errRecipientUsernameFormat   = errors.New("recipient username must start with @")
)
