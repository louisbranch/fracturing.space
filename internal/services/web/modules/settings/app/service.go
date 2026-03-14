package app

// AccountServiceConfig keeps account-surface gateway dependencies explicit.
type AccountServiceConfig struct {
	ProfileGateway  ProfileGateway
	LocaleGateway   LocaleGateway
	SecurityGateway SecurityGateway
}

// AIServiceConfig keeps AI-surface gateway dependencies explicit.
type AIServiceConfig struct {
	AIKeyGateway   AIKeyGateway
	AIAgentGateway AIAgentGateway
}

// ServiceConfig keeps all settings gateway dependencies explicit.
type ServiceConfig struct {
	ProfileGateway  ProfileGateway
	LocaleGateway   LocaleGateway
	SecurityGateway SecurityGateway
	AIKeyGateway    AIKeyGateway
	AIAgentGateway  AIAgentGateway
}

// service defines an internal contract used at this web package boundary.
type service struct {
	profileGateway  ProfileGateway
	localeGateway   LocaleGateway
	securityGateway SecurityGateway
	aiKeyGateway    AIKeyGateway
	aiAgentGateway  AIAgentGateway
}

// NewService constructs a settings service with fail-closed gateway defaults.
func NewService(config ServiceConfig) Service {
	return newServiceFromConfig(config)
}

// NewAccountService constructs an account-surface service with fail-closed defaults.
func NewAccountService(config AccountServiceConfig) AccountService {
	return newServiceFromConfig(ServiceConfig{
		ProfileGateway:  config.ProfileGateway,
		LocaleGateway:   config.LocaleGateway,
		SecurityGateway: config.SecurityGateway,
	})
}

// NewAIService constructs an AI-surface service with fail-closed defaults.
func NewAIService(config AIServiceConfig) AIService {
	return newServiceFromConfig(ServiceConfig{
		AIKeyGateway:   config.AIKeyGateway,
		AIAgentGateway: config.AIAgentGateway,
	})
}

// newService keeps package-local tests on a combined-gateway seam while
// production callers stay explicit by surface.
func newService(gateway Gateway) service {
	if gateway == nil {
		return newServiceFromConfig(ServiceConfig{})
	}
	return newServiceFromConfig(ServiceConfig{
		ProfileGateway:  gateway,
		LocaleGateway:   gateway,
		SecurityGateway: gateway,
		AIKeyGateway:    gateway,
		AIAgentGateway:  gateway,
	})
}

// newServiceFromConfig builds package wiring with fail-closed defaults per surface.
func newServiceFromConfig(config ServiceConfig) service {
	profileGateway := config.ProfileGateway
	if profileGateway == nil {
		profileGateway = unavailableGateway{}
	}
	localeGateway := config.LocaleGateway
	if localeGateway == nil {
		localeGateway = unavailableGateway{}
	}
	securityGateway := config.SecurityGateway
	if securityGateway == nil {
		securityGateway = unavailableGateway{}
	}
	aiKeyGateway := config.AIKeyGateway
	if aiKeyGateway == nil {
		aiKeyGateway = unavailableGateway{}
	}
	aiAgentGateway := config.AIAgentGateway
	if aiAgentGateway == nil {
		aiAgentGateway = unavailableGateway{}
	}
	return service{
		profileGateway:  profileGateway,
		localeGateway:   localeGateway,
		securityGateway: securityGateway,
		aiKeyGateway:    aiKeyGateway,
		aiAgentGateway:  aiAgentGateway,
	}
}
