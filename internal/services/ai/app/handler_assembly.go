package server

import (
	"context"
	"fmt"
	"log/slog"

	aiservice "github.com/louisbranch/fracturing.space/internal/services/ai/api/grpc/ai"
	"github.com/louisbranch/fracturing.space/internal/services/ai/openviking"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	svcpkg "github.com/louisbranch/fracturing.space/internal/services/ai/service"
)

// serviceHandlers collects all constructed gRPC service implementations.
type serviceHandlers struct {
	credentials           *aiservice.CredentialHandlers
	agents                *aiservice.AgentHandlers
	invocations           *aiservice.InvocationHandlers
	campaignOrchestration *aiservice.CampaignOrchestrationHandlers
	campaignDebug         *aiservice.CampaignDebugHandlers
	campaignArtifacts     *aiservice.CampaignArtifactHandlers
	systemReferences      *aiservice.SystemReferenceHandlers
	providerGrants        *aiservice.ProviderGrantHandlers
	accessRequests        *aiservice.AccessRequestHandlers
}

type workflowDeps struct {
	runtime                 runtimeDeps
	usagePolicy             *svcpkg.UsagePolicy
	agentBindingUsageReader *svcpkg.AgentBindingUsageReader
	authMaterialResolver    *svcpkg.AuthMaterialResolver
	accessibleAgentResolver *svcpkg.AccessibleAgentResolver
	debugUpdateBroker       *svcpkg.CampaignDebugUpdateBroker
	campaignTurnRunner      orchestration.CampaignTurnRunner
	campaignLogger          *slog.Logger
}

type authLifecycleModule struct {
	credentials    *aiservice.CredentialHandlers
	providerGrants *aiservice.ProviderGrantHandlers
	accessRequests *aiservice.AccessRequestHandlers
}

type agentRuntimeModule struct {
	agents      *aiservice.AgentHandlers
	invocations *aiservice.InvocationHandlers
}

type campaignRuntimeModule struct {
	campaignOrchestration *aiservice.CampaignOrchestrationHandlers
	campaignDebug         *aiservice.CampaignDebugHandlers
	campaignArtifacts     *aiservice.CampaignArtifactHandlers
	systemReferences      *aiservice.SystemReferenceHandlers
}

func buildHandlers(d runtimeDeps) (serviceHandlers, error) {
	w := buildWorkflowDeps(d)

	authModule, err := buildAuthLifecycleModule(w)
	if err != nil {
		return serviceHandlers{}, err
	}
	agentModule, err := buildAgentRuntimeModule(w)
	if err != nil {
		return serviceHandlers{}, err
	}
	campaignModule, err := buildCampaignRuntimeModule(w)
	if err != nil {
		return serviceHandlers{}, err
	}

	return serviceHandlers{
		credentials:           authModule.credentials,
		agents:                agentModule.agents,
		invocations:           agentModule.invocations,
		campaignOrchestration: campaignModule.campaignOrchestration,
		campaignDebug:         campaignModule.campaignDebug,
		campaignArtifacts:     campaignModule.campaignArtifacts,
		systemReferences:      campaignModule.systemReferences,
		providerGrants:        authModule.providerGrants,
		accessRequests:        authModule.accessRequests,
	}, nil
}

func buildWorkflowDeps(d runtimeDeps) workflowDeps {
	agentBindingUsageReader := svcpkg.NewAgentBindingUsageReader(d.gameBridge)
	authReferenceUsageReader := svcpkg.NewAuthReferenceUsageReader(d.store, agentBindingUsageReader)
	providerGrantRuntime := svcpkg.NewProviderGrantRuntime(svcpkg.ProviderGrantRuntimeConfig{
		ProviderGrantStore: d.store,
		ProviderRegistry:   d.providerRegistry,
		Sealer:             d.sealer,
	})
	return workflowDeps{
		runtime:                 d,
		usagePolicy:             svcpkg.NewUsagePolicy(svcpkg.UsagePolicyConfig{AgentBindingUsageReader: agentBindingUsageReader, AuthReferenceUsageReader: authReferenceUsageReader}),
		agentBindingUsageReader: agentBindingUsageReader,
		authMaterialResolver:    svcpkg.NewAuthMaterialResolver(svcpkg.AuthMaterialResolverConfig{CredentialStore: d.store, Sealer: d.sealer, ProviderGrantRuntime: providerGrantRuntime}),
		accessibleAgentResolver: svcpkg.NewAccessibleAgentResolver(d.store, d.store),
		debugUpdateBroker:       svcpkg.NewCampaignDebugUpdateBroker(),
		campaignTurnRunner:      buildCampaignTurnRunner(d),
		campaignLogger:          slog.Default().With("service", "ai", "component", "campaign_debug"),
	}
}

func buildAuthLifecycleModule(w workflowDeps) (authLifecycleModule, error) {
	credentialService, err := svcpkg.NewCredentialService(svcpkg.CredentialServiceConfig{
		CredentialStore:  w.runtime.store,
		ProviderRegistry: w.runtime.providerRegistry,
		Sealer:           w.runtime.sealer,
		UsagePolicy:      w.usagePolicy,
	})
	if err != nil {
		return authLifecycleModule{}, fmt.Errorf("credential service: %w", err)
	}
	credentialHandlers, err := aiservice.NewCredentialHandlers(aiservice.CredentialHandlersConfig{
		CredentialService: credentialService,
	})
	if err != nil {
		return authLifecycleModule{}, fmt.Errorf("credential handlers: %w", err)
	}

	providerGrantService, err := svcpkg.NewProviderGrantService(svcpkg.ProviderGrantServiceConfig{
		ProviderGrantStore:  w.runtime.store,
		ConnectSessionStore: w.runtime.store,
		ConnectFinisher:     w.runtime.store,
		Sealer:              w.runtime.sealer,
		ProviderRegistry:    w.runtime.providerRegistry,
		UsagePolicy:         w.usagePolicy,
	})
	if err != nil {
		return authLifecycleModule{}, fmt.Errorf("provider grant service: %w", err)
	}
	providerGrantHandlers, err := aiservice.NewProviderGrantHandlers(aiservice.ProviderGrantHandlersConfig{
		ProviderGrantService: providerGrantService,
	})
	if err != nil {
		return authLifecycleModule{}, fmt.Errorf("provider grant handlers: %w", err)
	}

	accessRequestService, err := svcpkg.NewAccessRequestService(svcpkg.AccessRequestServiceConfig{
		AgentStore:         w.runtime.store,
		AccessRequestStore: w.runtime.store,
		AuditEventStore:    w.runtime.store,
	})
	if err != nil {
		return authLifecycleModule{}, fmt.Errorf("access request service: %w", err)
	}
	accessRequestHandlers, err := aiservice.NewAccessRequestHandlers(aiservice.AccessRequestHandlersConfig{
		AccessRequestService: accessRequestService,
	})
	if err != nil {
		return authLifecycleModule{}, fmt.Errorf("access request handlers: %w", err)
	}

	return authLifecycleModule{
		credentials:    credentialHandlers,
		providerGrants: providerGrantHandlers,
		accessRequests: accessRequestHandlers,
	}, nil
}

func buildAgentRuntimeModule(w workflowDeps) (agentRuntimeModule, error) {
	authReferencePolicy, err := svcpkg.NewAuthReferencePolicy(svcpkg.AuthReferencePolicyConfig{
		CredentialStore:      w.runtime.store,
		ProviderGrantStore:   w.runtime.store,
		ProviderRegistry:     w.runtime.providerRegistry,
		AuthMaterialResolver: w.authMaterialResolver,
	})
	if err != nil {
		return agentRuntimeModule{}, fmt.Errorf("auth reference policy: %w", err)
	}

	agentService, err := svcpkg.NewAgentService(svcpkg.AgentServiceConfig{
		AgentStore:              w.runtime.store,
		AuthReferencePolicy:     authReferencePolicy,
		AccessibleAgentResolver: w.accessibleAgentResolver,
		UsagePolicy:             w.usagePolicy,
		AgentBindingUsageReader: w.agentBindingUsageReader,
	})
	if err != nil {
		return agentRuntimeModule{}, fmt.Errorf("agent service: %w", err)
	}
	agentHandlers, err := aiservice.NewAgentHandlers(aiservice.AgentHandlersConfig{
		AgentService: agentService,
	})
	if err != nil {
		return agentRuntimeModule{}, fmt.Errorf("agent handlers: %w", err)
	}

	invocationService, err := svcpkg.NewInvocationService(svcpkg.InvocationServiceConfig{
		AgentStore:              w.runtime.store,
		AuditEventStore:         w.runtime.store,
		AccessibleAgentResolver: w.accessibleAgentResolver,
		AuthMaterialResolver:    w.authMaterialResolver,
		ProviderRegistry:        w.runtime.providerRegistry,
	})
	if err != nil {
		return agentRuntimeModule{}, fmt.Errorf("invocation service: %w", err)
	}
	invocationHandlers, err := aiservice.NewInvocationHandlers(aiservice.InvocationHandlersConfig{
		InvocationService: invocationService,
	})
	if err != nil {
		return agentRuntimeModule{}, fmt.Errorf("invocation handlers: %w", err)
	}

	return agentRuntimeModule{
		agents:      agentHandlers,
		invocations: invocationHandlers,
	}, nil
}

func buildCampaignRuntimeModule(w workflowDeps) (campaignRuntimeModule, error) {
	campaignArtifactHandlers, err := aiservice.NewCampaignArtifactHandlers(aiservice.CampaignArtifactHandlersConfig{
		Manager:            w.runtime.campaignArtifactManager,
		CampaignAuthorizer: w.runtime.gameBridge,
	})
	if err != nil {
		return campaignRuntimeModule{}, fmt.Errorf("campaign artifact handlers: %w", err)
	}

	campaignOrchestrationService, err := svcpkg.NewCampaignOrchestrationService(svcpkg.CampaignOrchestrationServiceConfig{
		AgentStore:              w.runtime.store,
		CampaignArtifactManager: w.runtime.campaignArtifactManager,
		CampaignAuthStateReader: w.runtime.gameBridge,
		ProviderRegistry:        w.runtime.providerRegistry,
		CampaignTurnRunner:      w.campaignTurnRunner,
		TurnMemorySync: func(ctx context.Context, input svcpkg.TurnMemorySyncInput) error {
			if w.runtime.openVikingSessionSync == nil {
				return nil
			}
			return w.runtime.openVikingSessionSync.SyncTurn(ctx, openviking.TurnSyncInput{
				CampaignID:        input.CampaignID,
				SessionID:         input.SessionID,
				ParticipantID:     input.ParticipantID,
				UserText:          input.UserText,
				AssistantText:     input.AssistantText,
				RetrievedContexts: input.RetrievedContexts,
			})
		},
		DebugTraceStore:      w.runtime.store,
		DebugUpdateBroker:    w.debugUpdateBroker,
		SessionGrantConfig:   w.runtime.cfg.SessionGrantConfig,
		AuthMaterialResolver: w.authMaterialResolver,
		Logger:               w.campaignLogger,
	})
	if err != nil {
		return campaignRuntimeModule{}, fmt.Errorf("campaign orchestration service: %w", err)
	}
	campaignOrchestrationHandlers, err := aiservice.NewCampaignOrchestrationHandlers(aiservice.CampaignOrchestrationHandlersConfig{
		CampaignOrchestrationService: campaignOrchestrationService,
	})
	if err != nil {
		return campaignRuntimeModule{}, fmt.Errorf("campaign orchestration handlers: %w", err)
	}

	campaignDebugService, err := svcpkg.NewCampaignDebugService(svcpkg.CampaignDebugServiceConfig{
		DebugTraceStore: w.runtime.store,
		UpdateBroker:    w.debugUpdateBroker,
	})
	if err != nil {
		return campaignRuntimeModule{}, fmt.Errorf("campaign debug service: %w", err)
	}
	campaignDebugHandlers, err := aiservice.NewCampaignDebugHandlers(aiservice.CampaignDebugHandlersConfig{
		CampaignDebugService: campaignDebugService,
		CampaignAuthorizer:   w.runtime.gameBridge,
	})
	if err != nil {
		return campaignRuntimeModule{}, fmt.Errorf("campaign debug handlers: %w", err)
	}

	return campaignRuntimeModule{
		campaignOrchestration: campaignOrchestrationHandlers,
		campaignDebug:         campaignDebugHandlers,
		campaignArtifacts:     campaignArtifactHandlers,
		systemReferences:      w.runtime.systemReferenceHandlers,
	}, nil
}
