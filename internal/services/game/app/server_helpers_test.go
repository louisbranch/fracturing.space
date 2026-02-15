package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	gamegrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

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

func TestOpenEventStoreRequiresKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.db")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "")

	if _, err := openEventStore(path); err == nil {
		t.Fatal("expected error when HMAC key is missing")
	}
}

func TestOpenEventStoreSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.db")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "test-key")

	store, err := openEventStore(path)
	if err != nil {
		t.Fatalf("open event store: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close event store: %v", err)
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

	if _, _, err := dialAuthGRPC(ctx, "127.0.0.1:1"); err == nil {
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
		Type:          command.Type("action.gm_fear.set"),
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
	if store.events["c1"][0].Type != event.Type("action.gm_fear_changed") {
		t.Fatalf("event type = %s, want %s", store.events["c1"][0].Type, event.Type("action.gm_fear_changed"))
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
	events  map[string][]event.Event
	nextSeq map[string]uint64
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
