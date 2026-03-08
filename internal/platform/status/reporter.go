// Package status provides a platform library for pushing capability health
// to the status service. The Reporter is nil-client safe: if the status
// service is unreachable, state accumulates locally and pushes when available.
package status

import (
	"context"
	"log"
	"sync"
	"time"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DefaultPushInterval is the regular push cadence when no status changes occur.
const DefaultPushInterval = 10 * time.Second

// CapabilityStatus mirrors the proto enum for ergonomic use without proto imports.
type CapabilityStatus = statusv1.CapabilityStatus

// Status constants for convenience.
const (
	Operational = statusv1.CapabilityStatus_CAPABILITY_STATUS_OPERATIONAL
	Degraded    = statusv1.CapabilityStatus_CAPABILITY_STATUS_DEGRADED
	Unavailable = statusv1.CapabilityStatus_CAPABILITY_STATUS_UNAVAILABLE
	Maintenance = statusv1.CapabilityStatus_CAPABILITY_STATUS_MAINTENANCE
)

// Capability holds the current state of a single capability.
type Capability struct {
	Name       string
	Status     CapabilityStatus
	Detail     string
	ObservedAt time.Time
}

// Option configures reporter behavior.
type Option func(*Reporter)

// WithPushInterval sets the regular push cadence.
func WithPushInterval(d time.Duration) Option {
	return func(r *Reporter) {
		if d > 0 {
			r.pushInterval = d
		}
	}
}

// WithLogFunc sets the logging function for the reporter.
func WithLogFunc(logf func(string, ...any)) Option {
	return func(r *Reporter) {
		if logf != nil {
			r.logf = logf
		}
	}
}

// Reporter pushes capability health to the status service.
// It is safe for concurrent use.
type Reporter struct {
	service      string
	client       statusv1.StatusServiceClient
	pushInterval time.Duration
	logf         func(string, ...any)
	now          func() time.Time

	mu           sync.RWMutex
	capabilities map[string]*Capability
	notify       chan struct{} // buffered(1), signals immediate push
	loggedNoConn bool          // prevents log spam when client is nil
}

// NewReporter creates a reporter for the named service.
// client may be nil — the reporter accumulates state locally.
func NewReporter(service string, client statusv1.StatusServiceClient, opts ...Option) *Reporter {
	r := &Reporter{
		service:      service,
		client:       client,
		pushInterval: DefaultPushInterval,
		logf:         log.Printf,
		now:          time.Now,
		capabilities: make(map[string]*Capability),
		notify:       make(chan struct{}, 1),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Register adds a capability with an initial status.
func (r *Reporter) Register(name string, initial CapabilityStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.capabilities[name] = &Capability{
		Name:       name,
		Status:     initial,
		ObservedAt: r.now(),
	}
}

// Set updates a capability's status and detail, triggering an immediate push.
func (r *Reporter) Set(name string, status CapabilityStatus, detail string) {
	r.mu.Lock()
	cap, ok := r.capabilities[name]
	if !ok {
		cap = &Capability{Name: name}
		r.capabilities[name] = cap
	}
	changed := cap.Status != status || cap.Detail != detail
	cap.Status = status
	cap.Detail = detail
	cap.ObservedAt = r.now()
	r.mu.Unlock()

	if changed {
		r.triggerPush()
	}
}

// SetOperational is a convenience for Set(name, Operational, "").
func (r *Reporter) SetOperational(name string) {
	r.Set(name, Operational, "")
}

// SetDegraded is a convenience for Set(name, Degraded, detail).
func (r *Reporter) SetDegraded(name, detail string) {
	r.Set(name, Degraded, detail)
}

// SetUnavailable is a convenience for Set(name, Unavailable, detail).
func (r *Reporter) SetUnavailable(name, detail string) {
	r.Set(name, Unavailable, detail)
}

// Snapshot returns the current local capability state.
func (r *Reporter) Snapshot() []Capability {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Capability, 0, len(r.capabilities))
	for _, cap := range r.capabilities {
		result = append(result, *cap)
	}
	return result
}

// Start begins the background push loop. It returns a stop function that
// cancels the loop and waits for it to finish.
func (r *Reporter) Start(ctx context.Context) func() {
	loopCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		r.pushLoop(loopCtx)
	}()
	return func() {
		cancel()
		<-done
	}
}

func (r *Reporter) pushLoop(ctx context.Context) {
	ticker := time.NewTicker(r.pushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Final push attempt on shutdown.
			r.push(context.Background())
			return
		case <-ticker.C:
			r.push(ctx)
		case <-r.notify:
			r.push(ctx)
			// Drain any queued notifications to debounce.
			select {
			case <-r.notify:
			default:
			}
		}
	}
}

// SetClient replaces the status-service client at runtime. This allows
// late-binding when the status service connection becomes available after
// boot (e.g., via ManagedConn). A nil client disables pushing until the
// next SetClient call. If a non-nil client is set, an immediate push is
// triggered to flush any accumulated state.
func (r *Reporter) SetClient(client statusv1.StatusServiceClient) {
	r.mu.Lock()
	r.client = client
	r.loggedNoConn = false
	r.mu.Unlock()
	if client != nil {
		r.triggerPush()
	}
}

func (r *Reporter) push(ctx context.Context) {
	r.mu.RLock()
	client := r.client
	loggedNoConn := r.loggedNoConn
	caps := make([]*statusv1.CapabilityReport, 0, len(r.capabilities))
	for _, cap := range r.capabilities {
		caps = append(caps, &statusv1.CapabilityReport{
			Name:       cap.Name,
			Status:     cap.Status,
			Detail:     cap.Detail,
			ObservedAt: timestamppb.New(cap.ObservedAt),
		})
	}
	r.mu.RUnlock()

	if client == nil {
		if !loggedNoConn {
			r.mu.Lock()
			r.loggedNoConn = true
			r.mu.Unlock()
			r.logf("status reporter: no client connection, accumulating locally")
		}
		return
	}

	now := r.now()
	_, err := client.ReportStatus(ctx, &statusv1.ReportStatusRequest{
		Report: &statusv1.ServiceStatusReport{
			Service:      r.service,
			Capabilities: caps,
			ReportedAt:   timestamppb.New(now),
		},
	})
	if err != nil {
		r.logf("status reporter: push failed: %v", err)
	}
}

func (r *Reporter) triggerPush() {
	select {
	case r.notify <- struct{}{}:
	default:
		// Already queued.
	}
}
