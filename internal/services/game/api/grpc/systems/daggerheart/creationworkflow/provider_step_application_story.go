package creationworkflow

import (
	"context"
	"strings"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// applyBackgroundInput records the selected background text once it passes the
// shared required-field validation rules.
func applyBackgroundInput(profile *projectionstore.DaggerheartCharacterProfile, input *daggerheartv1.DaggerheartCreationStepBackgroundInput) error {
	background, err := validate.RequiredID(input.GetBackground(), "background")
	if err != nil {
		return err
	}
	profile.Background = background
	return nil
}

// applyExperiencesInput records the two starting experiences in the storage
// shape used by the creation profile.
func applyExperiencesInput(profile *projectionstore.DaggerheartCharacterProfile, input *daggerheartv1.DaggerheartCreationStepExperiencesInput) error {
	items := input.GetExperiences()
	if len(items) != 2 {
		return status.Error(codes.InvalidArgument, "exactly two experiences are required")
	}
	experiences := make([]projectionstore.DaggerheartExperience, 0, len(items))
	for _, item := range items {
		name, err := validate.RequiredID(item.GetName(), "experience name")
		if err != nil {
			return err
		}
		experiences = append(experiences, projectionstore.DaggerheartExperience{
			Name:     name,
			Modifier: 2,
		})
	}
	profile.Experiences = experiences
	return nil
}

// applyDomainCardsInput validates the starter domain cards against the
// selected class domains before persisting them to the profile.
func applyDomainCardsInput(ctx context.Context, content contentstore.DaggerheartContentReadStore, profile *projectionstore.DaggerheartCharacterProfile, input *daggerheartv1.DaggerheartCreationStepDomainCardsInput) error {
	if strings.TrimSpace(profile.ClassID) == "" {
		return status.Error(codes.FailedPrecondition, "class must be selected before domain cards")
	}
	class, err := content.GetDaggerheartClass(ctx, profile.ClassID)
	if err != nil {
		return invalidContentLookup(ctx, err, "get class", "class_id %q is not found", profile.ClassID)
	}
	allowedDomains := make(map[string]struct{}, len(class.DomainIDs))
	for _, domainID := range class.DomainIDs {
		trimmedDomainID := strings.TrimSpace(domainID)
		if trimmedDomainID == "" {
			continue
		}
		allowedDomains[trimmedDomainID] = struct{}{}
	}
	if len(allowedDomains) == 0 {
		return status.Errorf(codes.InvalidArgument, "class_id %q has no configured domains", profile.ClassID)
	}

	domainCardIDs := input.GetDomainCardIds()
	if len(domainCardIDs) != 2 {
		return status.Error(codes.InvalidArgument, "exactly two domain cards are required")
	}
	normalizedIDs := make([]string, 0, len(domainCardIDs))
	for _, domainCardID := range domainCardIDs {
		trimmed := strings.TrimSpace(domainCardID)
		if trimmed == "" {
			return status.Error(codes.InvalidArgument, "domain_card_ids must not contain empty values")
		}
		card, err := content.GetDaggerheartDomainCard(ctx, trimmed)
		if err != nil {
			return invalidContentLookup(ctx, err, "get domain card", "domain_card_id %q is not found", trimmed)
		}
		if card.Level != 1 {
			return status.Errorf(codes.InvalidArgument, "domain_card_id %q is level %d, only level 1 cards are allowed at creation", trimmed, card.Level)
		}
		if _, ok := allowedDomains[strings.TrimSpace(card.DomainID)]; !ok {
			return status.Errorf(codes.InvalidArgument, "domain_card_id %q is not in class domains", trimmed)
		}
		normalizedIDs = append(normalizedIDs, trimmed)
	}
	profile.DomainCardIDs = normalizedIDs
	return nil
}

// applyConnectionsInput records the finalized connection text once it passes
// the shared required-field validation rules.
func applyConnectionsInput(profile *projectionstore.DaggerheartCharacterProfile, input *daggerheartv1.DaggerheartCreationStepConnectionsInput) error {
	connections, err := validate.RequiredID(input.GetConnections(), "connections")
	if err != nil {
		return err
	}
	profile.Connections = connections
	return nil
}
