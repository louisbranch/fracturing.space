package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamegrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/system"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type fakeSystemModule struct {
	id      string
	version string
}

func (f fakeSystemModule) ID() string {
	return f.id
}

func (f fakeSystemModule) Version() string {
	return f.version
}

func (f fakeSystemModule) RegisterCommands(*command.Registry) error {
	return nil
}

func (f fakeSystemModule) RegisterEvents(*event.Registry) error {
	return nil
}

func (f fakeSystemModule) EmittableEventTypes() []event.Type {
	return nil
}

func (f fakeSystemModule) Decider() system.Decider {
	return nil
}

func (f fakeSystemModule) Projector() system.Projector {
	return nil
}

func (f fakeSystemModule) StateFactory() system.StateFactory {
	return nil
}

type fakeGameSystem struct {
	id      commonv1.GameSystem
	version string
}

func (f fakeGameSystem) ID() commonv1.GameSystem {
	return f.id
}

func (f fakeGameSystem) Version() string {
	return f.version
}

func (f fakeGameSystem) Name() string {
	return "fake-system"
}

func (f fakeGameSystem) RegistryMetadata() systems.RegistryMetadata {
	return systems.RegistryMetadata{}
}

func (f fakeGameSystem) StateFactory() systems.StateFactory {
	return nil
}

func (f fakeGameSystem) OutcomeApplier() systems.OutcomeApplier {
	return nil
}

type fakeSystemAdapter struct {
	id      commonv1.GameSystem
	version string
}

func (f fakeSystemAdapter) ID() commonv1.GameSystem {
	return f.id
}

func (f fakeSystemAdapter) Version() string {
	return f.version
}

func (f fakeSystemAdapter) Apply(context.Context, event.Event) error {
	return nil
}

func (f fakeSystemAdapter) Snapshot(context.Context, string) (any, error) {
	return nil, nil
}

type fakeProjectionOutboxShadowProcessor struct {
	processed int
	err       error
	calls     atomic.Int32
}

func (f *fakeProjectionOutboxShadowProcessor) ProcessProjectionApplyOutboxShadow(context.Context, time.Time, int) (int, error) {
	f.calls.Add(1)
	return f.processed, f.err
}

type scriptedProjectionOutboxShadowProcessor struct {
	processedByCall []int
	calls           atomic.Int32
}

func (f *scriptedProjectionOutboxShadowProcessor) ProcessProjectionApplyOutboxShadow(context.Context, time.Time, int) (int, error) {
	call := int(f.calls.Add(1))
	if call <= 0 || call > len(f.processedByCall) {
		return 0, nil
	}
	return f.processedByCall[call-1], nil
}

type fakeProjectionOutboxProcessor struct {
	processed int
	err       error
	calls     atomic.Int32
}

func (f *fakeProjectionOutboxProcessor) ProcessProjectionApplyOutbox(ctx context.Context, now time.Time, limit int, apply func(context.Context, event.Event) error) (int, error) {
	f.calls.Add(1)
	if apply != nil {
		_ = apply(ctx, event.Event{
			CampaignID: "camp-apply-worker",
			Seq:        1,
			Timestamp:  now,
			Type:       event.Type("campaign.created"),
			ActorType:  event.ActorTypeSystem,
			EntityType: "campaign",
			EntityID:   "camp-apply-worker",
		})
	}
	return f.processed, f.err
}

type scriptedProjectionOutboxProcessor struct {
	processedByCall []int
	calls           atomic.Int32
}

func (f *scriptedProjectionOutboxProcessor) ProcessProjectionApplyOutbox(ctx context.Context, now time.Time, limit int, apply func(context.Context, event.Event) error) (int, error) {
	call := int(f.calls.Add(1))
	if call <= 0 || call > len(f.processedByCall) {
		return 0, nil
	}
	processed := f.processedByCall[call-1]
	if processed > 0 && apply != nil {
		_ = apply(ctx, event.Event{
			CampaignID: "camp-apply-worker-scripted",
			Seq:        uint64(call),
			Timestamp:  now,
			Type:       event.Type("campaign.created"),
			ActorType:  event.ActorTypeSystem,
			EntityType: "campaign",
			EntityID:   "camp-apply-worker-scripted",
		})
	}
	return processed, nil
}

func TestEnsureDirCreatesParent(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "nested", "store.db")

	if err := ensureDir(path); err != nil {
		t.Fatalf("ensure dir: %v", err)
	}

	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Fatalf("expected dir to exist: %v", err)
	}
}

func TestEnsureDirRejectsFileParent(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("data"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	path := filepath.Join(file, "store.db")
	if err := ensureDir(path); err == nil {
		t.Fatal("expected error when parent is a file")
	}
}

func TestOpenProjectionStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projections.db")

	store, err := openProjectionStore(path)
	if err != nil {
		t.Fatalf("open projection store: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close projection store: %v", err)
	}
}

func TestBuildProjectionApplyOutboxApplySkipsDuplicateSeq(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projections.db")
	store, err := openProjectionStore(path)
	if err != nil {
		t.Fatalf("open projection store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("close projection store: %v", closeErr)
		}
	})

	now := time.Date(2026, 2, 18, 20, 0, 0, 0, time.UTC)
	if err := store.Put(context.Background(), storage.CampaignRecord{
		ID:               "camp-outbox-exactly-once",
		Name:             "Exactly Once",
		Locale:           commonv1.Locale_LOCALE_EN_US,
		System:           commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Status:           campaign.StatusDraft,
		GmMode:           campaign.GmModeHuman,
		Intent:           campaign.IntentStandard,
		AccessPolicy:     campaign.AccessPolicyPrivate,
		ParticipantCount: 0,
		CharacterCount:   0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		t.Fatalf("seed campaign: %v", err)
	}

	apply := buildProjectionApplyOutboxApply(store, nil)
	if apply == nil {
		t.Fatal("expected projection apply callback")
	}

	payload, err := json.Marshal(participant.JoinPayload{
		ParticipantID:  "part-apply-once",
		Name:           "Rook",
		Role:           "player",
		Controller:     "human",
		CampaignAccess: "member",
	})
	if err != nil {
		t.Fatalf("marshal participant payload: %v", err)
	}

	evt := event.Event{
		CampaignID:  "camp-outbox-exactly-once",
		Seq:         501,
		Type:        event.Type("participant.joined"),
		Timestamp:   now.Add(time.Second),
		EntityType:  "participant",
		EntityID:    "part-apply-once",
		PayloadJSON: payload,
	}

	if err := apply(context.Background(), evt); err != nil {
		t.Fatalf("first projection apply: %v", err)
	}
	if err := apply(context.Background(), evt); err != nil {
		t.Fatalf("duplicate projection apply: %v", err)
	}

	campaignRecord, err := store.Get(context.Background(), evt.CampaignID)
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if campaignRecord.ParticipantCount != 1 {
		t.Fatalf("expected participant count 1 after duplicate apply, got %d", campaignRecord.ParticipantCount)
	}
}

func TestOpenEventStoreRequiresKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.db")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "")

	if _, err := openEventStore(path, false); err == nil {
		t.Fatal("expected error when HMAC key is missing")
	}
}

func TestOpenEventStoreSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.db")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "test-key")

	store, err := openEventStore(path, false)
	if err != nil {
		t.Fatalf("open event store: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close event store: %v", err)
	}
}

func TestOpenEventStoreProjectionOutboxEnabledEnqueuesOnAppend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.db")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "test-key")

	store, err := openEventStore(path, true)
	if err != nil {
		t.Fatalf("open event store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("close event store: %v", closeErr)
		}
	})

	stored, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-enabled",
		Timestamp:   time.Date(2026, 2, 16, 1, 0, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-enabled",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	dbConn, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	defer dbConn.Close()

	var count int
	if err := dbConn.QueryRowContext(
		context.Background(),
		"SELECT COUNT(*) FROM projection_apply_outbox WHERE campaign_id = ? AND seq = ?",
		stored.CampaignID,
		stored.Seq,
	).Scan(&count); err != nil {
		t.Fatalf("query projection outbox: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one projection outbox row, got %d", count)
	}
}

func TestOpenEventStoreProjectionOutboxDisabledSkipsAppendEnqueue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.db")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "test-key")

	store, err := openEventStore(path, false)
	if err != nil {
		t.Fatalf("open event store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("close event store: %v", closeErr)
		}
	})

	stored, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-outbox-disabled",
		Timestamp:   time.Date(2026, 2, 16, 1, 0, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-outbox-disabled",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	dbConn, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	defer dbConn.Close()

	var count int
	if err := dbConn.QueryRowContext(
		context.Background(),
		"SELECT COUNT(*) FROM projection_apply_outbox WHERE campaign_id = ? AND seq = ?",
		stored.CampaignID,
		stored.Seq,
	).Scan(&count); err != nil {
		t.Fatalf("query projection outbox: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no projection outbox row, got %d", count)
	}
}

func TestOpenStorageBundleSuccess(t *testing.T) {
	base := t.TempDir()
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "test-key")

	srvEnv := serverEnv{
		EventsDBPath:      filepath.Join(base, "events.db"),
		ProjectionsDBPath: filepath.Join(base, "projections.db"),
		ContentDBPath:     filepath.Join(base, "content.db"),
	}
	bundle, err := openStorageBundle(srvEnv)
	if err != nil {
		t.Fatalf("open storage bundle: %v", err)
	}
	bundle.Close()
}

func TestOpenStorageBundleProjectionFailure(t *testing.T) {
	base := t.TempDir()
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "test-key")

	// Point projections at a file (not a directory) to force failure.
	blocker := filepath.Join(base, "blocker")
	if err := os.WriteFile(blocker, []byte("data"), 0o600); err != nil {
		t.Fatalf("write blocker: %v", err)
	}

	srvEnv := serverEnv{
		EventsDBPath:      filepath.Join(base, "events.db"),
		ProjectionsDBPath: filepath.Join(blocker, "projections.db"),
		ContentDBPath:     filepath.Join(base, "content.db"),
	}
	if _, err := openStorageBundle(srvEnv); err == nil {
		t.Fatal("expected error when projection store fails to open")
	}
}

func TestDialAuthGRPCTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	if _, err := dialAuthGRPC(ctx, "127.0.0.1:1"); err == nil {
		t.Fatal("expected dial auth error")
	}
}

func TestLoadServerEnvDomainEnabledDefaults(t *testing.T) {
	key := "FRACTURING_SPACE_GAME_DOMAIN_ENABLED"
	if val, ok := os.LookupEnv(key); ok {
		t.Cleanup(func() { _ = os.Setenv(key, val) })
	} else {
		t.Cleanup(func() { _ = os.Unsetenv(key) })
	}
	_ = os.Unsetenv(key)

	cfg := loadServerEnv()
	if !cfg.DomainEnabled {
		t.Fatal("expected domain to be enabled by default")
	}
}

func TestLoadServerEnvDomainEnabled(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_DOMAIN_ENABLED", "true")

	cfg := loadServerEnv()
	if !cfg.DomainEnabled {
		t.Fatal("expected domain to be enabled")
	}
}

func TestLoadServerEnvCompatibilityAppendDefaults(t *testing.T) {
	key := "FRACTURING_SPACE_GAME_COMPATIBILITY_APPEND_ENABLED"
	if val, ok := os.LookupEnv(key); ok {
		t.Cleanup(func() { _ = os.Setenv(key, val) })
	} else {
		t.Cleanup(func() { _ = os.Unsetenv(key) })
	}
	_ = os.Unsetenv(key)

	cfg := loadServerEnv()
	if cfg.CompatibilityAppendEnabled {
		t.Fatal("expected compatibility append to be disabled by default")
	}
}

func TestLoadServerEnvCompatibilityAppendEnabled(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_COMPATIBILITY_APPEND_ENABLED", "true")

	cfg := loadServerEnv()
	if !cfg.CompatibilityAppendEnabled {
		t.Fatal("expected compatibility append to be enabled")
	}
}

func TestLoadServerEnvProjectionApplyOutboxDefaults(t *testing.T) {
	key := "FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_ENABLED"
	if val, ok := os.LookupEnv(key); ok {
		t.Cleanup(func() { _ = os.Setenv(key, val) })
	} else {
		t.Cleanup(func() { _ = os.Unsetenv(key) })
	}
	_ = os.Unsetenv(key)

	cfg := loadServerEnv()
	if cfg.ProjectionApplyOutboxEnabled {
		t.Fatal("expected projection apply outbox to be disabled by default")
	}
}

func TestLoadServerEnvProjectionApplyOutboxEnabled(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_ENABLED", "true")

	cfg := loadServerEnv()
	if !cfg.ProjectionApplyOutboxEnabled {
		t.Fatal("expected projection apply outbox to be enabled")
	}
}

func TestLoadServerEnvProjectionApplyOutboxShadowWorkerDefaults(t *testing.T) {
	key := "FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_SHADOW_WORKER_ENABLED"
	if val, ok := os.LookupEnv(key); ok {
		t.Cleanup(func() { _ = os.Setenv(key, val) })
	} else {
		t.Cleanup(func() { _ = os.Unsetenv(key) })
	}
	_ = os.Unsetenv(key)

	cfg := loadServerEnv()
	if cfg.ProjectionApplyOutboxShadowWorkerEnabled {
		t.Fatal("expected projection apply outbox shadow worker to be disabled by default")
	}
}

func TestLoadServerEnvProjectionApplyOutboxShadowWorkerEnabled(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_SHADOW_WORKER_ENABLED", "true")

	cfg := loadServerEnv()
	if !cfg.ProjectionApplyOutboxShadowWorkerEnabled {
		t.Fatal("expected projection apply outbox shadow worker to be enabled")
	}
}

func TestLoadServerEnvProjectionApplyOutboxWorkerDefaults(t *testing.T) {
	key := "FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_WORKER_ENABLED"
	if val, ok := os.LookupEnv(key); ok {
		t.Cleanup(func() { _ = os.Setenv(key, val) })
	} else {
		t.Cleanup(func() { _ = os.Unsetenv(key) })
	}
	_ = os.Unsetenv(key)

	cfg := loadServerEnv()
	if cfg.ProjectionApplyOutboxWorkerEnabled {
		t.Fatal("expected projection apply outbox worker to be disabled by default")
	}
}

func TestLoadServerEnvProjectionApplyOutboxWorkerEnabled(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_WORKER_ENABLED", "true")

	cfg := loadServerEnv()
	if !cfg.ProjectionApplyOutboxWorkerEnabled {
		t.Fatal("expected projection apply outbox worker to be enabled")
	}
}

func TestResolveProjectionApplyOutboxModes_DefaultInlineMode(t *testing.T) {
	applyWorker, shadowWorker, mode, err := resolveProjectionApplyOutboxModes(serverEnv{
		ProjectionApplyOutboxEnabled:             false,
		ProjectionApplyOutboxWorkerEnabled:       false,
		ProjectionApplyOutboxShadowWorkerEnabled: false,
	})
	if err != nil {
		t.Fatalf("resolve projection outbox modes: %v", err)
	}
	if applyWorker {
		t.Fatal("expected apply worker to be disabled")
	}
	if shadowWorker {
		t.Fatal("expected shadow worker to be disabled")
	}
	if mode != projectionApplyModeInlineApplyOnly {
		t.Fatalf("mode = %q, want %q", mode, projectionApplyModeInlineApplyOnly)
	}
}

func TestResolveProjectionApplyOutboxModes_OutboxApplyOnly(t *testing.T) {
	applyWorker, shadowWorker, mode, err := resolveProjectionApplyOutboxModes(serverEnv{
		ProjectionApplyOutboxEnabled:             true,
		ProjectionApplyOutboxWorkerEnabled:       true,
		ProjectionApplyOutboxShadowWorkerEnabled: false,
	})
	if err != nil {
		t.Fatalf("resolve projection outbox modes: %v", err)
	}
	if !applyWorker {
		t.Fatal("expected apply worker to be enabled")
	}
	if shadowWorker {
		t.Fatal("expected shadow worker to be disabled")
	}
	if mode != projectionApplyModeOutboxApplyOnly {
		t.Fatalf("mode = %q, want %q", mode, projectionApplyModeOutboxApplyOnly)
	}
}

func TestResolveProjectionApplyOutboxModes_ShadowOnly(t *testing.T) {
	applyWorker, shadowWorker, mode, err := resolveProjectionApplyOutboxModes(serverEnv{
		ProjectionApplyOutboxEnabled:             true,
		ProjectionApplyOutboxWorkerEnabled:       false,
		ProjectionApplyOutboxShadowWorkerEnabled: true,
	})
	if err != nil {
		t.Fatalf("resolve projection outbox modes: %v", err)
	}
	if applyWorker {
		t.Fatal("expected apply worker to be disabled")
	}
	if !shadowWorker {
		t.Fatal("expected shadow worker to be enabled")
	}
	if mode != projectionApplyModeShadowOnly {
		t.Fatalf("mode = %q, want %q", mode, projectionApplyModeShadowOnly)
	}
}

func TestResolveProjectionApplyOutboxModes_InvalidWhenOutboxDisabledWithWorker(t *testing.T) {
	_, _, _, err := resolveProjectionApplyOutboxModes(serverEnv{
		ProjectionApplyOutboxEnabled:             false,
		ProjectionApplyOutboxWorkerEnabled:       true,
		ProjectionApplyOutboxShadowWorkerEnabled: false,
	})
	if err == nil {
		t.Fatal("expected worker without outbox to fail")
	}
}

func TestResolveProjectionApplyOutboxModes_InvalidWhenOutboxDisabledWithShadowWorker(t *testing.T) {
	_, _, _, err := resolveProjectionApplyOutboxModes(serverEnv{
		ProjectionApplyOutboxEnabled:             false,
		ProjectionApplyOutboxWorkerEnabled:       false,
		ProjectionApplyOutboxShadowWorkerEnabled: true,
	})
	if err == nil {
		t.Fatal("expected shadow worker without outbox to fail")
	}
}

func TestResolveProjectionApplyOutboxModes_InvalidWhenBothWorkersEnabled(t *testing.T) {
	_, _, _, err := resolveProjectionApplyOutboxModes(serverEnv{
		ProjectionApplyOutboxEnabled:             true,
		ProjectionApplyOutboxWorkerEnabled:       true,
		ProjectionApplyOutboxShadowWorkerEnabled: true,
	})
	if err == nil {
		t.Fatal("expected both workers to fail")
	}
}

func TestRunProjectionApplyOutboxWorkerRunsImmediateAndPeriodic(t *testing.T) {
	processor := &fakeProjectionOutboxProcessor{processed: 1}
	var applyCalls atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		runProjectionApplyOutboxWorker(
			ctx,
			processor,
			func(context.Context, event.Event) error {
				applyCalls.Add(1)
				return nil
			},
			5*time.Millisecond,
			16,
			func() time.Time { return time.Date(2026, 2, 16, 4, 30, 0, 0, time.UTC) },
			func(string, ...any) {},
		)
		close(done)
	}()

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if processor.calls.Load() >= 2 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if processor.calls.Load() < 2 {
		t.Fatalf("expected at least 2 worker calls, got %d", processor.calls.Load())
	}
	if applyCalls.Load() < 2 {
		t.Fatalf("expected at least 2 apply calls, got %d", applyCalls.Load())
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected worker loop to stop on context cancellation")
	}
}

func TestRunProjectionApplyOutboxWorkerDrainsBacklogBeforeTicker(t *testing.T) {
	processor := &scriptedProjectionOutboxProcessor{
		processedByCall: []int{16, 16, 0},
	}
	var applyCalls atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		runProjectionApplyOutboxWorker(
			ctx,
			processor,
			func(context.Context, event.Event) error {
				applyCalls.Add(1)
				return nil
			},
			time.Hour,
			16,
			func() time.Time { return time.Date(2026, 2, 16, 4, 45, 0, 0, time.UTC) },
			func(string, ...any) {},
		)
		close(done)
	}()

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if processor.calls.Load() >= 3 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if processor.calls.Load() < 3 {
		t.Fatalf("expected backlog to drain with immediate re-pass, got %d calls", processor.calls.Load())
	}
	if applyCalls.Load() < 2 {
		t.Fatalf("expected apply callback for each processed backlog pass, got %d calls", applyCalls.Load())
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected worker loop to stop on context cancellation")
	}
}

func TestRunProjectionApplyOutboxWorkerNoopWithoutProcessor(t *testing.T) {
	done := make(chan struct{})
	go func() {
		runProjectionApplyOutboxWorker(
			context.Background(),
			nil,
			func(context.Context, event.Event) error { return nil },
			5*time.Millisecond,
			8,
			time.Now,
			func(string, ...any) {},
		)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected worker to return immediately when processor is nil")
	}
}

func TestRunProjectionApplyOutboxWorkerNoopWithoutApply(t *testing.T) {
	done := make(chan struct{})
	go func() {
		runProjectionApplyOutboxWorker(
			context.Background(),
			&fakeProjectionOutboxProcessor{},
			nil,
			5*time.Millisecond,
			8,
			time.Now,
			func(string, ...any) {},
		)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected worker to return immediately when apply callback is nil")
	}
}

func TestRunProjectionApplyOutboxShadowWorkerRunsImmediateAndPeriodic(t *testing.T) {
	processor := &fakeProjectionOutboxShadowProcessor{processed: 1}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		runProjectionApplyOutboxShadowWorker(
			ctx,
			processor,
			5*time.Millisecond,
			16,
			func() time.Time { return time.Date(2026, 2, 16, 4, 0, 0, 0, time.UTC) },
			func(string, ...any) {},
		)
		close(done)
	}()

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if processor.calls.Load() >= 2 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if processor.calls.Load() < 2 {
		t.Fatalf("expected at least 2 worker calls, got %d", processor.calls.Load())
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected worker loop to stop on context cancellation")
	}
}

func TestRunProjectionApplyOutboxShadowWorkerDrainsBacklogBeforeTicker(t *testing.T) {
	processor := &scriptedProjectionOutboxShadowProcessor{
		processedByCall: []int{16, 16, 0},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		runProjectionApplyOutboxShadowWorker(
			ctx,
			processor,
			time.Hour,
			16,
			func() time.Time { return time.Date(2026, 2, 16, 4, 10, 0, 0, time.UTC) },
			func(string, ...any) {},
		)
		close(done)
	}()

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if processor.calls.Load() >= 3 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if processor.calls.Load() < 3 {
		t.Fatalf("expected backlog to drain with immediate re-pass, got %d calls", processor.calls.Load())
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected worker loop to stop on context cancellation")
	}
}

func TestRunProjectionApplyOutboxShadowWorkerNoopWithoutProcessor(t *testing.T) {
	done := make(chan struct{})
	go func() {
		runProjectionApplyOutboxShadowWorker(
			context.Background(),
			nil,
			5*time.Millisecond,
			8,
			time.Now,
			func(string, ...any) {},
		)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected worker to return immediately when processor is nil")
	}
}

func TestStartProjectionApplyOutboxWorkerNoopWhenDisabled(t *testing.T) {
	srv := &Server{}
	stop := srv.startProjectionApplyOutboxWorker(context.Background())
	stop()
}

func TestStartProjectionApplyOutboxWorkerProcessesRowsWhenEnabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.db")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "test-key")

	store, err := openEventStore(path, true)
	if err != nil {
		t.Fatalf("open event store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("close event store: %v", closeErr)
		}
	})

	stored, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-apply-worker-enabled",
		Timestamp:   time.Date(2026, 2, 16, 5, 30, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-apply-worker-enabled",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	var applyCalls atomic.Int32
	srv := &Server{
		stores: &storageBundle{
			events: store,
		},
		projectionApplyOutboxWorkerEnabled: true,
		projectionApplyOutboxApply: func(context.Context, event.Event) error {
			applyCalls.Add(1)
			return nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	stop := srv.startProjectionApplyOutboxWorker(ctx)
	t.Cleanup(func() {
		cancel()
		stop()
	})

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		rows, err := store.ListProjectionApplyOutboxRows(context.Background(), "", 10)
		if err == nil {
			remaining := false
			for _, row := range rows {
				if row.CampaignID == stored.CampaignID && row.Seq == stored.Seq {
					remaining = true
					break
				}
			}
			if !remaining && applyCalls.Load() > 0 {
				return
			}
		}
		time.Sleep(15 * time.Millisecond)
	}

	t.Fatal("expected apply worker to process and remove outbox row")
}

func TestStartProjectionApplyOutboxShadowWorkerNoopWhenDisabled(t *testing.T) {
	srv := &Server{}
	stop := srv.startProjectionApplyOutboxShadowWorker(context.Background())
	stop()
}

func TestStartProjectionApplyOutboxShadowWorkerProcessesRowsWhenEnabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.db")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "test-key")

	store, err := openEventStore(path, true)
	if err != nil {
		t.Fatalf("open event store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("close event store: %v", closeErr)
		}
	})

	stored, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-shadow-worker-enabled",
		Timestamp:   time.Date(2026, 2, 16, 5, 0, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    "camp-shadow-worker-enabled",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	srv := &Server{
		stores: &storageBundle{
			events: store,
		},
		projectionApplyOutboxShadowWorkerEnabled: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	stop := srv.startProjectionApplyOutboxShadowWorker(ctx)
	t.Cleanup(func() {
		cancel()
		stop()
	})

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		rows, err := store.ListProjectionApplyOutboxRows(context.Background(), "failed", 10)
		if err == nil {
			for _, row := range rows {
				if row.CampaignID == stored.CampaignID && row.Seq == stored.Seq && row.AttemptCount == 1 {
					return
				}
			}
		}
		time.Sleep(15 * time.Millisecond)
	}

	t.Fatal("expected shadow worker to observe and requeue outbox row")
}

func TestBuildDomainEngine_SpotlightSet(t *testing.T) {
	store := newFakeDomainEventStore()
	engine, err := buildDomainEngine(store)
	if err != nil {
		t.Fatalf("build domain engine: %v", err)
	}

	cmd := command.Command{
		CampaignID:  "c1",
		Type:        command.Type("session.spotlight_set"),
		SessionID:   "s1",
		EntityType:  "session",
		EntityID:    "s1",
		PayloadJSON: []byte(`{"spotlight_type":"character","character_id":"char-1"}`),
	}

	result, err := engine.Execute(context.Background(), cmd)
	if err != nil {
		t.Fatalf("execute command: %v", err)
	}
	if len(result.Decision.Rejections) > 0 {
		t.Fatalf("expected no rejections, got %d", len(result.Decision.Rejections))
	}
	if got := len(store.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if store.events["c1"][0].Type != event.Type("session.spotlight_set") {
		t.Fatalf("event type = %s, want %s", store.events["c1"][0].Type, event.Type("session.spotlight_set"))
	}
}

func TestBuildDomainEngine_CampaignCreate(t *testing.T) {
	store := newFakeDomainEventStore()
	engine, err := buildDomainEngine(store)
	if err != nil {
		t.Fatalf("build domain engine: %v", err)
	}

	cmd := command.Command{
		CampaignID:  "c1",
		Type:        command.Type("campaign.create"),
		EntityType:  "campaign",
		EntityID:    "c1",
		PayloadJSON: []byte(`{"name":"Test Campaign","game_system":"daggerheart","gm_mode":"human"}`),
	}

	result, err := engine.Execute(context.Background(), cmd)
	if err != nil {
		t.Fatalf("execute command: %v", err)
	}
	if len(result.Decision.Rejections) > 0 {
		t.Fatalf("expected no rejections, got %d", len(result.Decision.Rejections))
	}
	if got := len(store.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if store.events["c1"][0].Type != event.Type("campaign.created") {
		t.Fatalf("event type = %s, want %s", store.events["c1"][0].Type, event.Type("campaign.created"))
	}
}

func TestBuildDomainEngine_SystemCommand(t *testing.T) {
	store := newFakeDomainEventStore()
	engine, err := buildDomainEngine(store)
	if err != nil {
		t.Fatalf("build domain engine: %v", err)
	}

	cmd := command.Command{
		CampaignID:    "c1",
		Type:          command.Type("sys.daggerheart.gm_fear.set"),
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		EntityType:    "campaign",
		EntityID:      "c1",
		PayloadJSON:   []byte(`{"after":2}`),
	}

	result, err := engine.Execute(context.Background(), cmd)
	if err != nil {
		t.Fatalf("execute command: %v", err)
	}
	if len(result.Decision.Rejections) > 0 {
		t.Fatalf("expected no rejections, got %d", len(result.Decision.Rejections))
	}
	if got := len(store.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if store.events["c1"][0].Type != event.Type("sys.daggerheart.gm_fear_changed") {
		t.Fatalf("event type = %s, want %s", store.events["c1"][0].Type, event.Type("sys.daggerheart.gm_fear_changed"))
	}
}

func TestValidateSystemRegistrationParity(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		modules := []system.Module{
			fakeSystemModule{id: "DAGGERHEART", version: "v1"},
		}
		registry := systems.NewRegistry()
		if err := registry.Register(fakeGameSystem{
			id:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			version: "v1",
		}); err != nil {
			t.Fatalf("register metadata system: %v", err)
		}
		adapters := systems.NewAdapterRegistry()
		if err := adapters.Register(fakeSystemAdapter{
			id:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			version: "v1",
		}); err != nil {
			t.Fatalf("register adapter: %v", err)
		}
		if err := validateSystemRegistrationParity(modules, registry, adapters); err != nil {
			t.Fatalf("validate parity: %v", err)
		}
	})

	t.Run("missing adapter", func(t *testing.T) {
		modules := []system.Module{
			fakeSystemModule{id: "DAGGERHEART", version: "v1"},
		}
		registry := systems.NewRegistry()
		if err := registry.Register(fakeGameSystem{
			id:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			version: "v1",
		}); err != nil {
			t.Fatalf("register metadata system: %v", err)
		}
		adapters := systems.NewAdapterRegistry()

		err := validateSystemRegistrationParity(modules, registry, adapters)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "adapter") {
			t.Fatalf("error = %q, want adapter detail", err.Error())
		}
	})

	t.Run("metadata without module", func(t *testing.T) {
		registry := systems.NewRegistry()
		if err := registry.Register(fakeGameSystem{
			id:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			version: "v1",
		}); err != nil {
			t.Fatalf("register metadata system: %v", err)
		}
		adapters := systems.NewAdapterRegistry()
		if err := adapters.Register(fakeSystemAdapter{
			id:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			version: "v1",
		}); err != nil {
			t.Fatalf("register adapter: %v", err)
		}

		err := validateSystemRegistrationParity(nil, registry, adapters)
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, errSystemModuleRegistryMismatch) {
			t.Fatalf("error = %v, want %v", err, errSystemModuleRegistryMismatch)
		}
	})
}

func TestBuildDomainEngine_ReusesCheckpointedStateForReplay(t *testing.T) {
	store := newFakeDomainEventStore()
	engine, err := buildDomainEngine(store)
	if err != nil {
		t.Fatalf("build domain engine: %v", err)
	}

	createCmd := command.Command{
		CampaignID:  "c1",
		Type:        command.Type("campaign.create"),
		EntityType:  "campaign",
		EntityID:    "c1",
		PayloadJSON: []byte(`{"name":"Test Campaign","game_system":"daggerheart","gm_mode":"human"}`),
	}
	createResult, err := engine.Execute(context.Background(), createCmd)
	if err != nil {
		t.Fatalf("execute campaign.create: %v", err)
	}
	if len(createResult.Decision.Rejections) > 0 {
		t.Fatalf("campaign.create rejected: %s", createResult.Decision.Rejections[0].Message)
	}

	updateCmd := command.Command{
		CampaignID:  "c1",
		Type:        command.Type("campaign.update"),
		EntityType:  "campaign",
		EntityID:    "c1",
		PayloadJSON: []byte(`{"fields":{"status":"active"}}`),
	}
	updateResult, err := engine.Execute(context.Background(), updateCmd)
	if err != nil {
		t.Fatalf("execute campaign.update: %v", err)
	}
	if len(updateResult.Decision.Rejections) > 0 {
		t.Fatalf("campaign.update rejected: %s", updateResult.Decision.Rejections[0].Message)
	}

	zeroSeqStarts := 0
	for _, afterSeq := range store.listAfterSeq {
		if afterSeq == 0 {
			zeroSeqStarts++
		}
	}
	if zeroSeqStarts != 1 {
		t.Fatalf("expected 1 replay start from seq 0, got %d (calls: %v)", zeroSeqStarts, store.listAfterSeq)
	}
}

func TestConfigureDomainEnabled_SetsDomain(t *testing.T) {
	store := newFakeDomainEventStore()
	stores := gamegrpc.Stores{Event: store}

	if err := configureDomain(serverEnv{DomainEnabled: true}, &stores); err != nil {
		t.Fatalf("configure domain: %v", err)
	}
	if stores.Domain == nil {
		t.Fatal("expected domain to be configured")
	}
}

type fakeDomainEventStore struct {
	events       map[string][]event.Event
	nextSeq      map[string]uint64
	listAfterSeq []uint64
}

func newFakeDomainEventStore() *fakeDomainEventStore {
	return &fakeDomainEventStore{
		events:  make(map[string][]event.Event),
		nextSeq: make(map[string]uint64),
	}
}

func (s *fakeDomainEventStore) AppendEvent(_ context.Context, evt event.Event) (event.Event, error) {
	seq := s.nextSeq[evt.CampaignID]
	if seq == 0 {
		seq = 1
	}
	stored := evt
	stored.Seq = seq
	s.nextSeq[evt.CampaignID] = seq + 1
	s.events[evt.CampaignID] = append(s.events[evt.CampaignID], stored)
	return stored, nil
}

func (s *fakeDomainEventStore) GetEventByHash(_ context.Context, _ string) (event.Event, error) {
	return event.Event{}, storage.ErrNotFound
}

func (s *fakeDomainEventStore) GetEventBySeq(_ context.Context, campaignID string, seq uint64) (event.Event, error) {
	for _, evt := range s.events[campaignID] {
		if evt.Seq == seq {
			return evt, nil
		}
	}
	return event.Event{}, storage.ErrNotFound
}

func (s *fakeDomainEventStore) ListEvents(_ context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error) {
	s.listAfterSeq = append(s.listAfterSeq, afterSeq)
	entries := s.events[campaignID]
	result := make([]event.Event, 0, len(entries))
	for _, evt := range entries {
		if evt.Seq > afterSeq {
			result = append(result, evt)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (s *fakeDomainEventStore) ListEventsBySession(_ context.Context, campaignID, sessionID string, afterSeq uint64, limit int) ([]event.Event, error) {
	entries := s.events[campaignID]
	result := make([]event.Event, 0, len(entries))
	for _, evt := range entries {
		if evt.Seq > afterSeq && evt.SessionID == sessionID {
			result = append(result, evt)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (s *fakeDomainEventStore) GetLatestEventSeq(_ context.Context, campaignID string) (uint64, error) {
	entries := s.events[campaignID]
	if len(entries) == 0 {
		return 0, nil
	}
	return entries[len(entries)-1].Seq, nil
}

func (s *fakeDomainEventStore) ListEventsPage(_ context.Context, _ storage.ListEventsPageRequest) (storage.ListEventsPageResult, error) {
	return storage.ListEventsPageResult{}, nil
}
