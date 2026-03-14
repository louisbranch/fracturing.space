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

// serviceConfig keeps all settings gateway dependencies explicit inside the
// app package while transport callers stay on the owned account-vs-AI seams.
type serviceConfig struct {
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

// NewAccountService constructs an account-surface service with fail-closed defaults.
func NewAccountService(config AccountServiceConfig) AccountService {
	return newServiceFromConfig(serviceConfig{
		ProfileGateway:  config.ProfileGateway,
		LocaleGateway:   config.LocaleGateway,
		SecurityGateway: config.SecurityGateway,
	})
}

// NewAIService constructs an AI-surface service with fail-closed defaults.
func NewAIService(config AIServiceConfig) AIService {
	return newServiceFromConfig(serviceConfig{
		AIKeyGateway:   config.AIKeyGateway,
		AIAgentGateway: config.AIAgentGateway,
	})
}

// newServiceFromConfig builds package wiring with fail-closed defaults per surface.
func newServiceFromConfig(config serviceConfig) service {
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
