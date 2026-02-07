package dice

import (
	"math/rand"
	"testing"
)

func TestRollDice_Basic(t *testing.T) {
	tests := []struct {
		name    string
		request Request
		wantErr error
	}{
		{
			name: "single d6",
			request: Request{
				Dice: []Spec{{Sides: 6, Count: 1}},
				Seed: 42,
			},
			wantErr: nil,
		},
		{
			name: "2d6 + 1d8",
			request: Request{
				Dice: []Spec{
					{Sides: 6, Count: 2},
					{Sides: 8, Count: 1},
				},
				Seed: 42,
			},
			wantErr: nil,
		},
		{
			name: "no dice",
			request: Request{
				Dice: []Spec{},
				Seed: 42,
			},
			wantErr: ErrMissingDice,
		},
		{
			name: "invalid sides",
			request: Request{
				Dice: []Spec{{Sides: 0, Count: 1}},
				Seed: 42,
			},
			wantErr: ErrInvalidDiceSpec,
		},
		{
			name: "invalid count",
			request: Request{
				Dice: []Spec{{Sides: 6, Count: 0}},
				Seed: 42,
			},
			wantErr: ErrInvalidDiceSpec,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RollDice(tt.request)
			if err != tt.wantErr {
				t.Errorf("RollDice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr != nil {
				return
			}

			// Verify we got the right number of rolls
			if len(result.Rolls) != len(tt.request.Dice) {
				t.Errorf("RollDice() got %d rolls, want %d", len(result.Rolls), len(tt.request.Dice))
			}

			// Verify each roll has the right number of results
			for i, roll := range result.Rolls {
				if len(roll.Results) != tt.request.Dice[i].Count {
					t.Errorf("Roll[%d] got %d results, want %d", i, len(roll.Results), tt.request.Dice[i].Count)
				}
				if roll.Sides != tt.request.Dice[i].Sides {
					t.Errorf("Roll[%d] sides = %d, want %d", i, roll.Sides, tt.request.Dice[i].Sides)
				}

				// Verify results are within range
				for j, r := range roll.Results {
					if r < 1 || r > roll.Sides {
						t.Errorf("Roll[%d].Results[%d] = %d, out of range [1, %d]", i, j, r, roll.Sides)
					}
				}

				// Verify total
				sum := 0
				for _, r := range roll.Results {
					sum += r
				}
				if roll.Total != sum {
					t.Errorf("Roll[%d].Total = %d, want %d", i, roll.Total, sum)
				}
			}

			// Verify overall total
			total := 0
			for _, roll := range result.Rolls {
				total += roll.Total
			}
			if result.Total != total {
				t.Errorf("Result.Total = %d, want %d", result.Total, total)
			}
		})
	}
}

func TestRollDice_Determinism(t *testing.T) {
	request := Request{
		Dice: []Spec{
			{Sides: 12, Count: 2},
			{Sides: 6, Count: 4},
		},
		Seed: 12345,
	}

	result1, err := RollDice(request)
	if err != nil {
		t.Fatalf("RollDice() error = %v", err)
	}

	result2, err := RollDice(request)
	if err != nil {
		t.Fatalf("RollDice() error = %v", err)
	}

	// Results should be identical
	if result1.Total != result2.Total {
		t.Errorf("Totals differ: %d vs %d", result1.Total, result2.Total)
	}

	for i := range result1.Rolls {
		if result1.Rolls[i].Total != result2.Rolls[i].Total {
			t.Errorf("Roll[%d].Total differs: %d vs %d", i, result1.Rolls[i].Total, result2.Rolls[i].Total)
		}
		for j := range result1.Rolls[i].Results {
			if result1.Rolls[i].Results[j] != result2.Rolls[i].Results[j] {
				t.Errorf("Roll[%d].Results[%d] differs: %d vs %d", i, j, result1.Rolls[i].Results[j], result2.Rolls[i].Results[j])
			}
		}
	}
}

func TestRollWithRng(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	specs := []Spec{
		{Sides: 6, Count: 2},
	}

	result, err := RollWithRng(rng, specs)
	if err != nil {
		t.Fatalf("RollWithRng() error = %v", err)
	}

	if len(result.Rolls) != 1 {
		t.Errorf("RollWithRng() got %d rolls, want 1", len(result.Rolls))
	}

	if len(result.Rolls[0].Results) != 2 {
		t.Errorf("Roll[0] got %d results, want 2", len(result.Rolls[0].Results))
	}
}
