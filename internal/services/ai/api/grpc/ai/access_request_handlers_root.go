package ai

import (
	"fmt"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
)

// AccessRequestHandlers serves access-request and audit-event RPCs as thin
// transport wrappers over the access-request service.
type AccessRequestHandlers struct {
	aiv1.UnimplementedAccessRequestServiceServer
	svc *service.AccessRequestService
}

// AccessRequestHandlersConfig declares the dependencies for access-request RPCs.
type AccessRequestHandlersConfig struct {
	AccessRequestService *service.AccessRequestService
}

// NewAccessRequestHandlers builds an access-request RPC server from a service.
func NewAccessRequestHandlers(cfg AccessRequestHandlersConfig) (*AccessRequestHandlers, error) {
	if cfg.AccessRequestService == nil {
		return nil, fmt.Errorf("ai: NewAccessRequestHandlers: access request service is required")
	}
	return &AccessRequestHandlers{svc: cfg.AccessRequestService}, nil
}
