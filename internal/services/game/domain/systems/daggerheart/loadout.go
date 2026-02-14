package daggerheart

import "errors"

const (
	LoadoutMaxCards = 5
)

var (
	ErrLoadoutFull   = errors.New("loadout is full")
	ErrCardNotFound  = errors.New("card not found")
	ErrDuplicateCard = errors.New("card appears in both loadout and vault")
)

// Loadout tracks active and vaulted domain cards.
type Loadout struct {
	Active []string
	Vault  []string
}

// DomainCard captures recall cost metadata for loadout swaps.
type DomainCard struct {
	ID         string
	RecallCost int
}

// NewLoadout validates and returns a loadout.
func NewLoadout(active, vault []string) (Loadout, error) {
	if len(active) > LoadoutMaxCards {
		return Loadout{}, ErrLoadoutFull
	}
	seen := make(map[string]struct{}, len(active)+len(vault))
	for _, id := range active {
		if _, ok := seen[id]; ok {
			return Loadout{}, ErrDuplicateCard
		}
		seen[id] = struct{}{}
	}
	for _, id := range vault {
		if _, ok := seen[id]; ok {
			return Loadout{}, ErrDuplicateCard
		}
		seen[id] = struct{}{}
	}
	return Loadout{Active: active, Vault: vault}, nil
}

// MoveToActive moves a card from vault to active.
func (l Loadout) MoveToActive(cardID string) (Loadout, error) {
	index := indexOf(l.Vault, cardID)
	if index == -1 {
		return Loadout{}, ErrCardNotFound
	}
	if len(l.Active) >= LoadoutMaxCards {
		return Loadout{}, ErrLoadoutFull
	}
	active := append([]string{}, l.Active...)
	vault := append([]string{}, l.Vault...)
	active = append(active, cardID)
	vault = append(vault[:index], vault[index+1:]...)
	return Loadout{Active: active, Vault: vault}, nil
}

// MoveToVault moves a card from active to vault.
func (l Loadout) MoveToVault(cardID string) (Loadout, error) {
	index := indexOf(l.Active, cardID)
	if index == -1 {
		return Loadout{}, ErrCardNotFound
	}
	active := append([]string{}, l.Active...)
	vault := append([]string{}, l.Vault...)
	active = append(active[:index], active[index+1:]...)
	vault = append(vault, cardID)
	return Loadout{Active: active, Vault: vault}, nil
}

// MoveToActiveWithRecall moves a card from vault to active, applying recall cost when not at rest.
func (l Loadout) MoveToActiveWithRecall(card DomainCard, state *CharacterState, inRest bool) (Loadout, error) {
	if !inRest && card.RecallCost > 0 {
		if _, _, err := state.SpendResource(ResourceStress, card.RecallCost); err != nil {
			return Loadout{}, err
		}
	}
	return l.MoveToActive(card.ID)
}

func indexOf(values []string, target string) int {
	for i, value := range values {
		if value == target {
			return i
		}
	}
	return -1
}
