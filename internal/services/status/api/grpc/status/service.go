// Package status implements the StatusService gRPC handler.
package status

import (
	"context"
	"sort"
	"strings"
	"time"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	"github.com/louisbranch/fracturing.space/internal/services/status/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// OverrideStore persists operator overrides across restarts.
type OverrideStore interface {
	PutOverride(ctx context.Context, ov domain.Override) error
	DeleteOverride(ctx context.Context, service, capability string) error
}

// Service implements statusv1.StatusServiceServer.
type Service struct {
	statusv1.UnimplementedStatusServiceServer
	aggregator *domain.Aggregator
	overrides  OverrideStore
	now        func() time.Time
}

// NewService creates a status service backed by the given aggregator and override store.
func NewService(aggregator *domain.Aggregator, overrides OverrideStore, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{
		aggregator: aggregator,
		overrides:  overrides,
		now:        now,
	}
}

// ReportStatus accepts a capability health push from a service.
func (s *Service) ReportStatus(_ context.Context, req *statusv1.ReportStatusRequest) (*statusv1.ReportStatusResponse, error) {
	if req == nil || req.Report == nil {
		return nil, status.Error(codes.InvalidArgument, "report is required")
	}
	report := req.Report
	service := strings.TrimSpace(report.Service)
	if service == "" {
		return nil, status.Error(codes.InvalidArgument, "service name is required")
	}

	caps := make([]domain.CapabilityReport, 0, len(report.Capabilities))
	for _, c := range report.Capabilities {
		if c == nil {
			continue
		}
		cr := domain.CapabilityReport{
			Name:   c.Name,
			Status: domain.CapabilityStatus(c.Status),
			Detail: c.Detail,
		}
		if c.ObservedAt != nil {
			cr.ObservedAt = c.ObservedAt.AsTime()
		}
		caps = append(caps, cr)
	}

	reportedAt := s.now()
	if report.ReportedAt != nil {
		reportedAt = report.ReportedAt.AsTime()
	}

	s.aggregator.ApplyReport(service, caps, reportedAt)
	return &statusv1.ReportStatusResponse{}, nil
}

// GetSystemStatus returns the resolved health view of all known services.
func (s *Service) GetSystemStatus(_ context.Context, _ *statusv1.GetSystemStatusRequest) (*statusv1.GetSystemStatusResponse, error) {
	snapshots := s.aggregator.Snapshot()
	services := make([]*statusv1.ServiceStatus, 0, len(snapshots))
	for _, ss := range snapshots {
		services = append(services, serviceSnapshotToProto(ss))
	}
	sort.Slice(services, func(i, j int) bool {
		return services[i].GetService() < services[j].GetService()
	})
	return &statusv1.GetSystemStatusResponse{Services: services}, nil
}

// SetOverride applies an operator override to a specific capability.
func (s *Service) SetOverride(ctx context.Context, req *statusv1.SetOverrideRequest) (*statusv1.SetOverrideResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "set override request is required")
	}
	service := strings.TrimSpace(req.GetService())
	capability := strings.TrimSpace(req.GetCapability())
	if service == "" {
		return nil, status.Error(codes.InvalidArgument, "service is required")
	}
	if capability == "" {
		return nil, status.Error(codes.InvalidArgument, "capability is required")
	}

	ov := domain.Override{
		Service:    service,
		Capability: capability,
		Status:     domain.CapabilityStatus(req.GetStatus()),
		Reason:     domain.OverrideReason(req.GetReason()),
		Detail:     req.GetDetail(),
		SetAt:      s.now(),
	}

	s.aggregator.SetOverride(ov)

	if s.overrides != nil {
		if err := s.overrides.PutOverride(ctx, ov); err != nil {
			return nil, status.Errorf(codes.Internal, "persist override: %v", err)
		}
	}
	return &statusv1.SetOverrideResponse{}, nil
}

// ClearOverride removes an operator override from a capability.
func (s *Service) ClearOverride(ctx context.Context, req *statusv1.ClearOverrideRequest) (*statusv1.ClearOverrideResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "clear override request is required")
	}
	service := strings.TrimSpace(req.GetService())
	capability := strings.TrimSpace(req.GetCapability())
	if service == "" {
		return nil, status.Error(codes.InvalidArgument, "service is required")
	}
	if capability == "" {
		return nil, status.Error(codes.InvalidArgument, "capability is required")
	}

	s.aggregator.ClearOverride(service, capability)

	if s.overrides != nil {
		if err := s.overrides.DeleteOverride(ctx, service, capability); err != nil {
			return nil, status.Errorf(codes.Internal, "delete override: %v", err)
		}
	}
	return &statusv1.ClearOverrideResponse{}, nil
}

func serviceSnapshotToProto(ss domain.ServiceSnapshot) *statusv1.ServiceStatus {
	caps := make([]*statusv1.CapabilitySnapshot, 0, len(ss.Capabilities))
	for _, c := range ss.Capabilities {
		caps = append(caps, capabilitySnapshotToProto(c))
	}
	var lastReport *timestamppb.Timestamp
	if !ss.LastReportAt.IsZero() {
		lastReport = timestamppb.New(ss.LastReportAt)
	}
	return &statusv1.ServiceStatus{
		Service:         ss.Service,
		AggregateStatus: statusv1.CapabilityStatus(ss.AggregateStatus),
		Capabilities:    caps,
		LastReportAt:    lastReport,
		HasOverrides:    ss.HasOverrides,
	}
}

func capabilitySnapshotToProto(c domain.CapabilitySnapshot) *statusv1.CapabilitySnapshot {
	snap := &statusv1.CapabilitySnapshot{
		Name:            c.Name,
		ReportedStatus:  statusv1.CapabilityStatus(c.ReportedStatus),
		ReportedDetail:  c.ReportedDetail,
		EffectiveStatus: statusv1.CapabilityStatus(c.EffectiveStatus),
		HasOverride:     c.HasOverride,
	}
	if !c.ObservedAt.IsZero() {
		snap.ObservedAt = timestamppb.New(c.ObservedAt)
	}
	if c.Override != nil {
		snap.Override = &statusv1.CapabilityOverride{
			Capability: c.Override.Capability,
			Status:     statusv1.CapabilityStatus(c.Override.Status),
			Reason:     statusv1.OverrideReason(c.Override.Reason),
			Detail:     c.Override.Detail,
		}
		if !c.Override.SetAt.IsZero() {
			snap.Override.SetAt = timestamppb.New(c.Override.SetAt)
		}
	}
	return snap
}
