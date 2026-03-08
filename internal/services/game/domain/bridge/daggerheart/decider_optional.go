package daggerheart

import (
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

// ── Gold/Currency ───────────────────────────────────────────────────────

const (
	goldHandfulsMax = 9 // 10 handfuls = 1 bag
	goldBagsMax     = 9 // 10 bags = 1 chest
	goldChestsMax   = 1

	rejectionCodeGoldInvalid = "GOLD_INVALID"
)

func decideGoldUpdate(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		EventTypeGoldUpdated, "character",
		func(p *GoldUpdatePayload) string { return strings.TrimSpace(p.CharacterID.String()) },
		func(_ SnapshotState, _ bool, p *GoldUpdatePayload, _ func() time.Time) *command.Rejection {
			p.CharacterID = ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
			p.Reason = strings.TrimSpace(p.Reason)

			if p.HandfulsAfter < 0 || p.HandfulsAfter > goldHandfulsMax {
				return &command.Rejection{Code: rejectionCodeGoldInvalid, Message: "handfuls must be in range 0..9"}
			}
			if p.BagsAfter < 0 || p.BagsAfter > goldBagsMax {
				return &command.Rejection{Code: rejectionCodeGoldInvalid, Message: "bags must be in range 0..9"}
			}
			if p.ChestsAfter < 0 || p.ChestsAfter > goldChestsMax {
				return &command.Rejection{Code: rejectionCodeGoldInvalid, Message: "chests must be in range 0..1"}
			}
			if p.HandfulsBefore == p.HandfulsAfter && p.BagsBefore == p.BagsAfter && p.ChestsBefore == p.ChestsAfter {
				return &command.Rejection{Code: rejectionCodeGoldInvalid, Message: "gold update must change at least one denomination"}
			}
			return nil
		},
		func(_ SnapshotState, _ bool, p GoldUpdatePayload) GoldUpdatedPayload {
			return GoldUpdatedPayload{
				CharacterID: p.CharacterID,
				Handfuls:    p.HandfulsAfter,
				Bags:        p.BagsAfter,
				Chests:      p.ChestsAfter,
				Reason:      p.Reason,
			}
		},
		now)
}

// ── Domain Card Vault ───────────────────────────────────────────────────

const (
	rejectionCodeDomainCardAcquireInvalid = "DOMAIN_CARD_ACQUIRE_INVALID"
)

func decideDomainCardAcquire(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, EventTypeDomainCardAcquired, "character",
		func(p *DomainCardAcquirePayload) string { return strings.TrimSpace(p.CharacterID.String()) },
		func(p *DomainCardAcquirePayload, _ func() time.Time) *command.Rejection {
			p.CharacterID = ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
			p.CardID = strings.TrimSpace(p.CardID)
			p.Destination = strings.TrimSpace(p.Destination)
			if p.Destination != "vault" && p.Destination != "loadout" {
				return &command.Rejection{Code: rejectionCodeDomainCardAcquireInvalid, Message: "destination must be vault or loadout"}
			}
			return nil
		}, now)
}

// ── Equipment ───────────────────────────────────────────────────────────

const (
	rejectionCodeEquipmentSwapInvalid = "EQUIPMENT_SWAP_INVALID"
)

func decideEquipmentSwap(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, EventTypeEquipmentSwapped, "character",
		func(p *EquipmentSwapPayload) string { return strings.TrimSpace(p.CharacterID.String()) },
		func(p *EquipmentSwapPayload, _ func() time.Time) *command.Rejection {
			p.CharacterID = ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
			p.ItemID = strings.TrimSpace(p.ItemID)
			p.ItemType = strings.TrimSpace(p.ItemType)
			p.From = strings.TrimSpace(p.From)
			p.To = strings.TrimSpace(p.To)

			if p.ItemType != "weapon" && p.ItemType != "armor" {
				return &command.Rejection{Code: rejectionCodeEquipmentSwapInvalid, Message: "item_type must be weapon or armor"}
			}
			validSlot := func(s string) bool {
				return s == "active" || s == "inventory" || s == "none"
			}
			if !validSlot(p.From) || !validSlot(p.To) {
				return &command.Rejection{Code: rejectionCodeEquipmentSwapInvalid, Message: "from and to must be active, inventory, or none"}
			}
			if p.From == p.To {
				return &command.Rejection{Code: rejectionCodeEquipmentSwapInvalid, Message: "from and to must differ"}
			}
			return nil
		}, now)
}

// ── Consumables ─────────────────────────────────────────────────────────

const (
	consumableStackMax             = 5
	rejectionCodeConsumableInvalid = "CONSUMABLE_INVALID"
)

func decideConsumableUse(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		EventTypeConsumableUsed, "character",
		func(p *ConsumableUsePayload) string { return strings.TrimSpace(p.CharacterID.String()) },
		func(_ SnapshotState, _ bool, p *ConsumableUsePayload, _ func() time.Time) *command.Rejection {
			p.CharacterID = ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
			p.ConsumableID = strings.TrimSpace(p.ConsumableID)
			if p.QuantityBefore <= 0 {
				return &command.Rejection{Code: rejectionCodeConsumableInvalid, Message: "quantity_before must be positive"}
			}
			if p.QuantityAfter != p.QuantityBefore-1 {
				return &command.Rejection{Code: rejectionCodeConsumableInvalid, Message: "quantity_after must be quantity_before - 1"}
			}
			return nil
		},
		func(_ SnapshotState, _ bool, p ConsumableUsePayload) ConsumableUsedPayload {
			return ConsumableUsedPayload{
				CharacterID:  p.CharacterID,
				ConsumableID: p.ConsumableID,
				Quantity:     p.QuantityAfter,
			}
		},
		now)
}

func decideConsumableAcquire(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		EventTypeConsumableAcquired, "character",
		func(p *ConsumableAcquirePayload) string { return strings.TrimSpace(p.CharacterID.String()) },
		func(_ SnapshotState, _ bool, p *ConsumableAcquirePayload, _ func() time.Time) *command.Rejection {
			p.CharacterID = ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
			p.ConsumableID = strings.TrimSpace(p.ConsumableID)
			if p.QuantityAfter < 1 || p.QuantityAfter > consumableStackMax {
				return &command.Rejection{Code: rejectionCodeConsumableInvalid, Message: "quantity_after must be in range 1..5"}
			}
			if p.QuantityAfter != p.QuantityBefore+1 {
				return &command.Rejection{Code: rejectionCodeConsumableInvalid, Message: "quantity_after must be quantity_before + 1"}
			}
			return nil
		},
		func(_ SnapshotState, _ bool, p ConsumableAcquirePayload) ConsumableAcquiredPayload {
			return ConsumableAcquiredPayload{
				CharacterID:  p.CharacterID,
				ConsumableID: p.ConsumableID,
				Quantity:     p.QuantityAfter,
			}
		},
		now)
}
