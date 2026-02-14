package daggerheart

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
)

func TestActionHandlersRejectNilRequests(t *testing.T) {
	ctx := context.Background()
	service := &DaggerheartService{}

	tests := []struct {
		name string
		call func(*DaggerheartService) error
	}{
		{
			name: "ApplyDamage",
			call: func(s *DaggerheartService) error {
				_, err := s.ApplyDamage(ctx, nil)
				return err
			},
		},
		{
			name: "ApplyRest",
			call: func(s *DaggerheartService) error {
				_, err := s.ApplyRest(ctx, nil)
				return err
			},
		},
		{
			name: "ApplyDowntimeMove",
			call: func(s *DaggerheartService) error {
				_, err := s.ApplyDowntimeMove(ctx, nil)
				return err
			},
		},
		{
			name: "SwapLoadout",
			call: func(s *DaggerheartService) error {
				_, err := s.SwapLoadout(ctx, nil)
				return err
			},
		},
		{
			name: "ApplyDeathMove",
			call: func(s *DaggerheartService) error {
				_, err := s.ApplyDeathMove(ctx, nil)
				return err
			},
		},
		{
			name: "ApplyConditions",
			call: func(s *DaggerheartService) error {
				_, err := s.ApplyConditions(ctx, nil)
				return err
			},
		},
		{
			name: "ApplyGmMove",
			call: func(s *DaggerheartService) error {
				_, err := s.ApplyGmMove(ctx, nil)
				return err
			},
		},
		{
			name: "CreateCountdown",
			call: func(s *DaggerheartService) error {
				_, err := s.CreateCountdown(ctx, nil)
				return err
			},
		},
		{
			name: "UpdateCountdown",
			call: func(s *DaggerheartService) error {
				_, err := s.UpdateCountdown(ctx, nil)
				return err
			},
		},
		{
			name: "DeleteCountdown",
			call: func(s *DaggerheartService) error {
				_, err := s.DeleteCountdown(ctx, nil)
				return err
			},
		},
		{
			name: "ResolveBlazeOfGlory",
			call: func(s *DaggerheartService) error {
				_, err := s.ResolveBlazeOfGlory(ctx, nil)
				return err
			},
		},
		{
			name: "SessionActionRoll",
			call: func(s *DaggerheartService) error {
				_, err := s.SessionActionRoll(ctx, nil)
				return err
			},
		},
		{
			name: "SessionDamageRoll",
			call: func(s *DaggerheartService) error {
				_, err := s.SessionDamageRoll(ctx, nil)
				return err
			},
		},
		{
			name: "SessionAttackFlow",
			call: func(s *DaggerheartService) error {
				_, err := s.SessionAttackFlow(ctx, nil)
				return err
			},
		},
		{
			name: "SessionReactionFlow",
			call: func(s *DaggerheartService) error {
				_, err := s.SessionReactionFlow(ctx, nil)
				return err
			},
		},
		{
			name: "SessionAdversaryAttackRoll",
			call: func(s *DaggerheartService) error {
				_, err := s.SessionAdversaryAttackRoll(ctx, nil)
				return err
			},
		},
		{
			name: "SessionAdversaryActionCheck",
			call: func(s *DaggerheartService) error {
				_, err := s.SessionAdversaryActionCheck(ctx, nil)
				return err
			},
		},
		{
			name: "SessionAdversaryAttackFlow",
			call: func(s *DaggerheartService) error {
				_, err := s.SessionAdversaryAttackFlow(ctx, nil)
				return err
			},
		},
		{
			name: "SessionGroupActionFlow",
			call: func(s *DaggerheartService) error {
				_, err := s.SessionGroupActionFlow(ctx, nil)
				return err
			},
		},
		{
			name: "SessionTagTeamFlow",
			call: func(s *DaggerheartService) error {
				_, err := s.SessionTagTeamFlow(ctx, nil)
				return err
			},
		},
		{
			name: "ApplyRollOutcome",
			call: func(s *DaggerheartService) error {
				_, err := s.ApplyRollOutcome(ctx, nil)
				return err
			},
		},
		{
			name: "ApplyAttackOutcome",
			call: func(s *DaggerheartService) error {
				_, err := s.ApplyAttackOutcome(ctx, nil)
				return err
			},
		},
		{
			name: "ApplyAdversaryAttackOutcome",
			call: func(s *DaggerheartService) error {
				_, err := s.ApplyAdversaryAttackOutcome(ctx, nil)
				return err
			},
		},
		{
			name: "ApplyReactionOutcome",
			call: func(s *DaggerheartService) error {
				_, err := s.ApplyReactionOutcome(ctx, nil)
				return err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertStatusCode(t, tc.call(service), codes.InvalidArgument)
		})
	}
}
