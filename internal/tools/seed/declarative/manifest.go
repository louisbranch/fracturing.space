package declarative

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// LoadManifest reads and validates a manifest from disk.
func LoadManifest(path string) (Manifest, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return Manifest{}, fmt.Errorf("manifest path is required")
	}

	f, err := os.Open(trimmed)
	if err != nil {
		return Manifest{}, fmt.Errorf("open manifest: %w", err)
	}
	defer func() { _ = f.Close() }()

	decoder := json.NewDecoder(f)
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode manifest: %w", err)
	}

	if err := ValidateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

// ValidateManifest ensures keys and references are coherent.
func ValidateManifest(manifest Manifest) error {
	if strings.TrimSpace(manifest.Name) == "" {
		return fmt.Errorf("manifest name is required")
	}

	userKeys := make(map[string]struct{}, len(manifest.Users))
	for i, user := range manifest.Users {
		key := strings.TrimSpace(user.Key)
		if key == "" {
			return fmt.Errorf("users[%d].key is required", i)
		}
		if _, exists := userKeys[key]; exists {
			return fmt.Errorf("duplicate user key %q", key)
		}
		userKeys[key] = struct{}{}
		if strings.TrimSpace(user.Email) == "" {
			return fmt.Errorf("users[%d].email is required", i)
		}
		for j, contactKey := range user.Contacts {
			contactKey = strings.TrimSpace(contactKey)
			if contactKey == "" {
				return fmt.Errorf("users[%d].contacts[%d] is required", i, j)
			}
		}
	}

	campaignKeys := make(map[string]struct{}, len(manifest.Campaigns))
	for i, campaign := range manifest.Campaigns {
		key := strings.TrimSpace(campaign.Key)
		if key == "" {
			return fmt.Errorf("campaigns[%d].key is required", i)
		}
		if _, exists := campaignKeys[key]; exists {
			return fmt.Errorf("duplicate campaign key %q", key)
		}
		campaignKeys[key] = struct{}{}

		if strings.TrimSpace(campaign.Name) == "" {
			return fmt.Errorf("campaigns[%d].name is required", i)
		}
		owner := strings.TrimSpace(campaign.OwnerUserKey)
		if owner == "" {
			return fmt.Errorf("campaigns[%d].owner_user_key is required", i)
		}
		if _, exists := userKeys[owner]; !exists {
			return fmt.Errorf("campaigns[%d].owner_user_key %q is not declared in users", i, owner)
		}

		participantKeys := make(map[string]struct{}, len(campaign.Participants))
		for j, participant := range campaign.Participants {
			participantKey := strings.TrimSpace(participant.Key)
			if participantKey == "" {
				return fmt.Errorf("campaigns[%d].participants[%d].key is required", i, j)
			}
			if _, exists := participantKeys[participantKey]; exists {
				return fmt.Errorf("campaigns[%d] has duplicate participant key %q", i, participantKey)
			}
			participantKeys[participantKey] = struct{}{}
			if strings.TrimSpace(participant.Name) == "" {
				return fmt.Errorf("campaigns[%d].participants[%d].name is required", i, j)
			}
			userKey := strings.TrimSpace(participant.UserKey)
			if userKey != "" {
				if _, exists := userKeys[userKey]; !exists {
					return fmt.Errorf("campaigns[%d].participants[%d].user_key %q is not declared in users", i, j, userKey)
				}
			}
		}

		characterKeys := make(map[string]struct{}, len(campaign.Characters))
		for j, character := range campaign.Characters {
			characterKey := strings.TrimSpace(character.Key)
			if characterKey == "" {
				return fmt.Errorf("campaigns[%d].characters[%d].key is required", i, j)
			}
			if _, exists := characterKeys[characterKey]; exists {
				return fmt.Errorf("campaigns[%d] has duplicate character key %q", i, characterKey)
			}
			characterKeys[characterKey] = struct{}{}
			if strings.TrimSpace(character.Name) == "" {
				return fmt.Errorf("campaigns[%d].characters[%d].name is required", i, j)
			}
			controllerKey := strings.TrimSpace(character.ControllerParticipantKey)
			if controllerKey != "" {
				if _, exists := participantKeys[controllerKey]; !exists {
					return fmt.Errorf("campaigns[%d].characters[%d].controller_participant_key %q is not declared in campaign participants", i, j, controllerKey)
				}
			}
		}

		sessionKeys := make(map[string]struct{}, len(campaign.Sessions))
		for j, session := range campaign.Sessions {
			sessionKey := strings.TrimSpace(session.Key)
			if sessionKey == "" {
				return fmt.Errorf("campaigns[%d].sessions[%d].key is required", i, j)
			}
			if _, exists := sessionKeys[sessionKey]; exists {
				return fmt.Errorf("campaigns[%d] has duplicate session key %q", i, sessionKey)
			}
			sessionKeys[sessionKey] = struct{}{}
			if strings.TrimSpace(session.Name) == "" {
				return fmt.Errorf("campaigns[%d].sessions[%d].name is required", i, j)
			}
		}
	}

	for i, user := range manifest.Users {
		for j, contactKey := range user.Contacts {
			contactKey = strings.TrimSpace(contactKey)
			if _, exists := userKeys[contactKey]; !exists {
				return fmt.Errorf("users[%d].contacts[%d] references missing user key %q", i, j, contactKey)
			}
		}
	}

	forkKeys := make(map[string]struct{}, len(manifest.Forks))
	for i, fork := range manifest.Forks {
		key := strings.TrimSpace(fork.Key)
		if key == "" {
			return fmt.Errorf("forks[%d].key is required", i)
		}
		if _, exists := forkKeys[key]; exists {
			return fmt.Errorf("duplicate fork key %q", key)
		}
		forkKeys[key] = struct{}{}

		source := strings.TrimSpace(fork.SourceCampaignKey)
		if source == "" {
			return fmt.Errorf("forks[%d].source_campaign_key is required", i)
		}
		if _, exists := campaignKeys[source]; !exists {
			return fmt.Errorf("forks[%d].source_campaign_key %q is not declared in campaigns", i, source)
		}

		owner := strings.TrimSpace(fork.OwnerUserKey)
		if owner == "" {
			return fmt.Errorf("forks[%d].owner_user_key is required", i)
		}
		if _, exists := userKeys[owner]; !exists {
			return fmt.Errorf("forks[%d].owner_user_key %q is not declared in users", i, owner)
		}
	}

	for i, listing := range manifest.Listings {
		campaignKey := strings.TrimSpace(listing.CampaignKey)
		if campaignKey == "" {
			return fmt.Errorf("listings[%d].campaign_key is required", i)
		}
		if _, exists := campaignKeys[campaignKey]; !exists {
			return fmt.Errorf("listings[%d].campaign_key %q is not declared in campaigns", i, campaignKey)
		}
		if strings.TrimSpace(listing.Title) == "" {
			return fmt.Errorf("listings[%d].title is required", i)
		}
		if strings.TrimSpace(listing.Description) == "" {
			return fmt.Errorf("listings[%d].description is required", i)
		}
		if strings.TrimSpace(listing.ExpectedDurationLabel) == "" {
			return fmt.Errorf("listings[%d].expected_duration_label is required", i)
		}
		if listing.RecommendedParticipantsMin <= 0 {
			return fmt.Errorf("listings[%d].recommended_participants_min must be > 0", i)
		}
		if listing.RecommendedParticipantsMax < listing.RecommendedParticipantsMin {
			return fmt.Errorf("listings[%d].recommended_participants_max must be >= min", i)
		}
	}

	return nil
}
