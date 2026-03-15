package scene

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func TestAllActingParticipantsYieldedAfter(t *testing.T) {
	tests := []struct {
		name   string
		state  State
		next   ids.ParticipantID
		expect bool
	}{
		{
			name:   "empty participants returns false",
			state:  State{},
			next:   "p1",
			expect: false,
		},
		{
			name: "single participant yielding returns true",
			state: State{
				PlayerPhaseActingParticipants: map[ids.ParticipantID]bool{"p1": true},
				PlayerPhaseSlots: map[ids.ParticipantID]PlayerPhaseSlot{
					"p1": {ParticipantID: "p1", Yielded: false},
				},
			},
			next:   "p1",
			expect: true,
		},
		{
			name: "two participants both yielded after next",
			state: State{
				PlayerPhaseActingParticipants: map[ids.ParticipantID]bool{
					"p1": true,
					"p2": true,
				},
				PlayerPhaseSlots: map[ids.ParticipantID]PlayerPhaseSlot{
					"p1": {ParticipantID: "p1", Yielded: false},
					"p2": {ParticipantID: "p2", Yielded: true},
				},
			},
			next:   "p1",
			expect: true,
		},
		{
			name: "two participants other not yielded",
			state: State{
				PlayerPhaseActingParticipants: map[ids.ParticipantID]bool{
					"p1": true,
					"p2": true,
				},
				PlayerPhaseSlots: map[ids.ParticipantID]PlayerPhaseSlot{
					"p1": {ParticipantID: "p1", Yielded: false},
					"p2": {ParticipantID: "p2", Yielded: false},
				},
			},
			next:   "p1",
			expect: false,
		},
		{
			name: "participant without slot blocks transition",
			state: State{
				PlayerPhaseActingParticipants: map[ids.ParticipantID]bool{
					"p1": true,
					"p2": true,
				},
				PlayerPhaseSlots: map[ids.ParticipantID]PlayerPhaseSlot{
					"p1": {ParticipantID: "p1", Yielded: false},
					// p2 has no slot — should block, not silently pass.
				},
			},
			next:   "p1",
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := allActingParticipantsYieldedAfter(tt.state, tt.next)
			if got != tt.expect {
				t.Fatalf("allActingParticipantsYieldedAfter() = %v, want %v", got, tt.expect)
			}
		})
	}
}
