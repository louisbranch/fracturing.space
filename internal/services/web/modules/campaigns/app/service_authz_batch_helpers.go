package app

import "strings"

// buildAuthorizationChecksByID builds one check per unique item ID while retaining all index positions.
func buildAuthorizationChecksByID(
	itemCount int,
	itemID func(int) string,
	checkForID func(string, int) AuthorizationCheck,
) ([]AuthorizationCheck, map[string][]int) {
	checks := make([]AuthorizationCheck, 0, itemCount)
	indexesByCheckID := make(map[string][]int, itemCount)
	for idx := 0; idx < itemCount; idx++ {
		checkID := strings.TrimSpace(itemID(idx))
		if checkID == "" {
			continue
		}
		indexesByCheckID[checkID] = append(indexesByCheckID[checkID], idx)
		if len(indexesByCheckID[checkID]) > 1 {
			continue
		}
		checks = append(checks, checkForID(checkID, idx))
	}
	return checks, indexesByCheckID
}

// applyAuthorizationDecisions projects decisions onto item indexes grouped by check ID.
func applyAuthorizationDecisions(
	checks []AuthorizationCheck,
	indexesByCheckID map[string][]int,
	decisions []AuthorizationDecision,
	apply func(int, AuthorizationDecision),
) {
	for idx, decision := range decisions {
		checkID := resolvedDecisionCheckID(decision, idx, checks)
		if checkID == "" {
			continue
		}
		indexes, found := indexesByCheckID[checkID]
		if !found {
			continue
		}
		for _, itemIndex := range indexes {
			apply(itemIndex, decision)
		}
	}
}

// resolvedDecisionCheckID resolves decision check IDs with request-order fallback.
func resolvedDecisionCheckID(decision AuthorizationDecision, idx int, checks []AuthorizationCheck) string {
	checkID := strings.TrimSpace(decision.CheckID)
	if checkID != "" {
		return checkID
	}
	if idx < 0 || idx >= len(checks) {
		return ""
	}
	return strings.TrimSpace(checks[idx].CheckID)
}

// allowedByCheckID returns evaluated+allowed decisions keyed by resolved check ID.
func allowedByCheckID(checks []AuthorizationCheck, decisions []AuthorizationDecision) map[string]bool {
	allowed := make(map[string]bool, len(decisions))
	for idx, decision := range decisions {
		if !decision.Evaluated || !decision.Allowed {
			continue
		}
		checkID := resolvedDecisionCheckID(decision, idx, checks)
		if checkID == "" {
			continue
		}
		allowed[checkID] = true
	}
	return allowed
}
