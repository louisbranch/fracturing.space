package app

// profileGatewayHealthReporter lets adapters report profile-surface readiness.
type profileGatewayHealthReporter interface {
	ProfileGatewayHealthy() bool
}

// localeGatewayHealthReporter lets adapters report locale-surface readiness.
type localeGatewayHealthReporter interface {
	LocaleGatewayHealthy() bool
}

// securityGatewayHealthReporter lets adapters report security-surface readiness.
type securityGatewayHealthReporter interface {
	SecurityGatewayHealthy() bool
}

// aiKeyGatewayHealthReporter lets adapters report AI-key surface readiness.
type aiKeyGatewayHealthReporter interface {
	AIKeyGatewayHealthy() bool
}

// aiAgentGatewayHealthReporter lets adapters report AI-agent surface readiness.
type aiAgentGatewayHealthReporter interface {
	AIAgentGatewayHealthy() bool
}

// unavailableGateway defines an internal contract used at this web package boundary.
type unavailableGateway struct{}

// NewUnavailableGateway returns a fail-closed settings gateway.
func NewUnavailableGateway() unavailableGateway {
	return unavailableGateway{}
}

// IsProfileGatewayHealthy reports whether profile settings can serve requests.
func IsProfileGatewayHealthy(gateway ProfileGateway) bool {
	if gateway == nil {
		return false
	}
	if reporter, ok := gateway.(profileGatewayHealthReporter); ok {
		return reporter.ProfileGatewayHealthy()
	}
	_, unavailable := gateway.(unavailableGateway)
	return !unavailable
}

// IsLocaleGatewayHealthy reports whether locale settings can serve requests.
func IsLocaleGatewayHealthy(gateway LocaleGateway) bool {
	if gateway == nil {
		return false
	}
	if reporter, ok := gateway.(localeGatewayHealthReporter); ok {
		return reporter.LocaleGatewayHealthy()
	}
	_, unavailable := gateway.(unavailableGateway)
	return !unavailable
}

// IsSecurityGatewayHealthy reports whether security settings can serve requests.
func IsSecurityGatewayHealthy(gateway SecurityGateway) bool {
	if gateway == nil {
		return false
	}
	if reporter, ok := gateway.(securityGatewayHealthReporter); ok {
		return reporter.SecurityGatewayHealthy()
	}
	_, unavailable := gateway.(unavailableGateway)
	return !unavailable
}

// IsAIKeyGatewayHealthy reports whether AI credential settings can serve requests.
func IsAIKeyGatewayHealthy(gateway AIKeyGateway) bool {
	if gateway == nil {
		return false
	}
	if reporter, ok := gateway.(aiKeyGatewayHealthReporter); ok {
		return reporter.AIKeyGatewayHealthy()
	}
	_, unavailable := gateway.(unavailableGateway)
	return !unavailable
}

// IsAIAgentGatewayHealthy reports whether AI agent settings can serve requests.
func IsAIAgentGatewayHealthy(gateway AIAgentGateway) bool {
	if gateway == nil {
		return false
	}
	if reporter, ok := gateway.(aiAgentGatewayHealthReporter); ok {
		return reporter.AIAgentGatewayHealthy()
	}
	_, unavailable := gateway.(unavailableGateway)
	return !unavailable
}

// IsAccountGatewayHealthy reports whether any account-owned settings surface is available.
func IsAccountGatewayHealthy(gateway AccountGateway) bool {
	return IsProfileGatewayHealthy(gateway) || IsLocaleGatewayHealthy(gateway) || IsSecurityGatewayHealthy(gateway)
}

// IsAIGatewayHealthy reports whether any AI-owned settings surface is available.
func IsAIGatewayHealthy(gateway AIGateway) bool {
	return IsAIKeyGatewayHealthy(gateway) || IsAIAgentGatewayHealthy(gateway)
}
