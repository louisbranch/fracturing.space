package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func ValidateLoadoutSwapPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.LoadoutSwapPayload) error {
		if err := RequireCharacterID(p.CharacterID); err != nil {
			return err
		}
		return RequireTrimmedValue(p.CardID, "card_id")
	})
}

func ValidateLoadoutSwappedPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.LoadoutSwappedPayload) error {
		if err := RequireCharacterID(p.CharacterID); err != nil {
			return err
		}
		return RequireTrimmedValue(p.CardID, "card_id")
	})
}

func ValidateRestTakePayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.RestTakePayload) error {
		if err := RequireTrimmedValue(p.RestType, "rest_type"); err != nil {
			return err
		}
		if len(p.Participants) == 0 {
			return errors.New("participants are required")
		}
		for _, participantID := range p.Participants {
			if err := RequireTrimmedValue(participantID.String(), "participants.character_id"); err != nil {
				return err
			}
		}
		for _, advance := range p.CampaignCountdownAdvances {
			if err := ValidateRestCampaignCountdownPayload(advance); err != nil {
				return err
			}
		}
		for _, move := range p.DowntimeMoves {
			if err := ValidateDowntimeMoveAppliedPayloadFields(move); err != nil {
				return err
			}
		}
		if !HasRestTakeMutation(p) {
			return errors.New("rest.take must record at least one durable outcome")
		}
		return nil
	})
}

func ValidateRestTakenPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.RestTakenPayload) error {
		if err := RequireTrimmedValue(p.RestType, "rest_type"); err != nil {
			return err
		}
		if p.GMFear < daggerheartstate.GMFearMin || p.GMFear > daggerheartstate.GMFearMax {
			return fmt.Errorf("gm_fear_after must be in range %d..%d", daggerheartstate.GMFearMin, daggerheartstate.GMFearMax)
		}
		if len(p.Participants) == 0 {
			return errors.New("participants are required")
		}
		return nil
	})
}

func ValidateRestCampaignCountdownPayload(p payload.CampaignCountdownAdvancePayload) error {
	if strings.TrimSpace(p.CountdownID.String()) == "" {
		return errors.New("campaign_countdown.countdown_id is required")
	}
	if p.AdvancedBy <= 0 {
		return errors.New("campaign_countdown advance must be positive")
	}
	return nil
}

func ValidateRestLongTermCountdownPayload(p payload.CampaignCountdownAdvancePayload) error {
	return ValidateRestCampaignCountdownPayload(p)
}

func HasRestTakeMutation(p payload.RestTakePayload) bool {
	if p.GMFearBefore != p.GMFearAfter ||
		p.ShortRestsBefore != p.ShortRestsAfter ||
		p.RefreshRest ||
		p.RefreshLongRest ||
		p.Interrupted ||
		len(p.CampaignCountdownAdvances) > 0 ||
		len(p.CountdownAdvances) > 0 ||
		len(p.DowntimeMoves) > 0 {
		return true
	}
	return len(p.Participants) > 0
}
