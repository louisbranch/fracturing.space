package generator

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/tools/seed/worldbuilder"
	"google.golang.org/grpc"
)

// sequenceSource returns a deterministic series of values to force duplicate names.
type sequenceSource struct {
	values []int64
	index  int
}

func (s *sequenceSource) Int63() int64 {
	if len(s.values) == 0 {
		return 0
	}
	if s.index >= len(s.values) {
		return 0
	}
	value := s.values[s.index]
	s.index++
	return value
}

func (s *sequenceSource) Seed(seed int64) {
	s.index = 0
}

func TestGenerator_UniqueUserDisplayNames(t *testing.T) {
	var names []string
	camp := &fakeCampaignCreator{
		createCampaign: func(_ context.Context, _ *statev1.CreateCampaignRequest, _ ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
			return &statev1.CreateCampaignResponse{
				Campaign:         &statev1.Campaign{Id: "camp-1", Name: "camp"},
				OwnerParticipant: &statev1.Participant{Id: "owner-1"},
			}, nil
		},
	}
	partSeq := 0
	part := &fakeParticipantCreator{
		create: func(_ context.Context, in *statev1.CreateParticipantRequest, _ ...grpc.CallOption) (*statev1.CreateParticipantResponse, error) {
			partSeq++
			names = append(names, in.Name)
			return &statev1.CreateParticipantResponse{
				Participant: &statev1.Participant{
					Id:         fmt.Sprintf("p-%d", partSeq),
					Name:       in.Name,
					Controller: in.Controller,
				},
			}, nil
		},
	}

	mid := int64(1 << 62)
	rng := rand.New(&sequenceSource{values: []int64{0, 0, 0, 0, mid, 0, 0, mid, 0, 0}})
	g := newGenerator(Config{}, rng, worldbuilder.New(rng), testDeps(camp, part, nil, nil, nil, nil))

	if _, _, err := g.createCampaign(context.Background(), statev1.GmMode_HUMAN); err != nil {
		t.Fatalf("unexpected campaign error: %v", err)
	}
	if _, err := g.createParticipants(context.Background(), "camp-1", "", 2); err != nil {
		t.Fatalf("unexpected participant error: %v", err)
	}

	if len(names) != 2 {
		t.Fatalf("expected 2 participants created, got %d", len(names))
	}
	seen := make(map[string]struct{})
	for _, name := range names {
		if _, ok := seen[name]; ok {
			t.Fatalf("expected unique participant names, got duplicate %q", name)
		}
		seen[name] = struct{}{}
	}
}
