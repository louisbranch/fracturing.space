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

// accountService defines the account-owned concrete service used by transport.
type accountService struct {
	profileGateway  ProfileGateway
	localeGateway   LocaleGateway
	securityGateway SecurityGateway
}

// aiService defines the AI-owned concrete service used by transport.
type aiService struct {
	aiKeyGateway   AIKeyGateway
	aiAgentGateway AIAgentGateway
}

// NewAccountService constructs an account-surface service with fail-closed defaults.
func NewAccountService(config AccountServiceConfig) AccountService {
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
	return accountService{
		profileGateway:  profileGateway,
		localeGateway:   localeGateway,
		securityGateway: securityGateway,
	}
}

// NewAIService constructs an AI-surface service with fail-closed defaults.
func NewAIService(config AIServiceConfig) AIService {
	aiKeyGateway := config.AIKeyGateway
	if aiKeyGateway == nil {
		aiKeyGateway = unavailableGateway{}
	}
	aiAgentGateway := config.AIAgentGateway
	if aiAgentGateway == nil {
		aiAgentGateway = unavailableGateway{}
	}
	return aiService{
		aiKeyGateway:   aiKeyGateway,
		aiAgentGateway: aiAgentGateway,
	}
}
