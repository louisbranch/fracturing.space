package adversarytransport

import (
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"google.golang.org/grpc/codes"
)

func findEntryFeature(entry contentstore.DaggerheartAdversaryEntry, featureID string) (contentstore.DaggerheartAdversaryFeature, bool) {
	for _, feature := range entry.Features {
		if strings.TrimSpace(feature.ID) == strings.TrimSpace(featureID) {
			return feature, true
		}
	}
	return contentstore.DaggerheartAdversaryFeature{}, false
}

func featureApplyStateStatus(rule *rules.AdversaryFeatureRule) string {
	switch rule.Kind {
	case rules.AdversaryFeatureRuleKindRetaliatoryDamageOnCloseHit:
		return "ready"
	default:
		return "active"
	}
}

func upsertAdversaryFeatureState(current []projectionstore.DaggerheartAdversaryFeatureState, next rules.AdversaryFeatureState) []projectionstore.DaggerheartAdversaryFeatureState {
	updated := make([]projectionstore.DaggerheartAdversaryFeatureState, 0, len(current)+1)
	seen := false
	for _, state := range current {
		if strings.TrimSpace(state.FeatureID) == strings.TrimSpace(next.FeatureID) {
			updated = append(updated, projectionstore.DaggerheartAdversaryFeatureState{
				FeatureID:       strings.TrimSpace(next.FeatureID),
				Status:          strings.TrimSpace(next.Status),
				FocusedTargetID: strings.TrimSpace(next.FocusedTargetID),
			})
			seen = true
			continue
		}
		updated = append(updated, state)
	}
	if !seen {
		updated = append(updated, projectionstore.DaggerheartAdversaryFeatureState{
			FeatureID:       strings.TrimSpace(next.FeatureID),
			Status:          strings.TrimSpace(next.Status),
			FocusedTargetID: strings.TrimSpace(next.FocusedTargetID),
		})
	}
	return updated
}

func toBridgeAdversaryFeatureStates(in []projectionstore.DaggerheartAdversaryFeatureState) []rules.AdversaryFeatureState {
	out := make([]rules.AdversaryFeatureState, 0, len(in))
	for _, state := range in {
		out = append(out, rules.AdversaryFeatureState{
			FeatureID:       strings.TrimSpace(state.FeatureID),
			Status:          strings.TrimSpace(state.Status),
			FocusedTargetID: strings.TrimSpace(state.FocusedTargetID),
		})
	}
	return out
}

func toBridgeAdversaryPendingExperience(in *projectionstore.DaggerheartAdversaryPendingExperience) *rules.AdversaryPendingExperience {
	if in == nil {
		return nil
	}
	return &rules.AdversaryPendingExperience{
		Name:     strings.TrimSpace(in.Name),
		Modifier: in.Modifier,
	}
}

func intPtr(value int) *int {
	return &value
}

func invalidArgument(message string) error {
	return statusError(codes.InvalidArgument, message)
}

func internal(message string) error {
	return statusError(codes.Internal, message)
}
