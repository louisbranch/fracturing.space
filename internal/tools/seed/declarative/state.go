package declarative

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const stateVersion = 1

type seedState struct {
	Version   int               `json:"version"`
	UpdatedAt string            `json:"updated_at"`
	Entries   map[string]string `json:"entries"`
}

func defaultState() seedState {
	return seedState{
		Version: stateVersion,
		Entries: map[string]string{},
	}
}

func loadState(path string) (seedState, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return defaultState(), nil
	}
	data, err := os.ReadFile(trimmed)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultState(), nil
		}
		return seedState{}, fmt.Errorf("read state file: %w", err)
	}

	var state seedState
	if err := json.Unmarshal(data, &state); err != nil {
		return seedState{}, fmt.Errorf("decode state file: %w", err)
	}
	if state.Version == 0 {
		state.Version = stateVersion
	}
	if state.Entries == nil {
		state.Entries = map[string]string{}
	}
	return state, nil
}

func saveState(path string, state seedState) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil
	}
	if state.Entries == nil {
		state.Entries = map[string]string{}
	}
	state.Version = stateVersion
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state file: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(trimmed), 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	if err := os.WriteFile(trimmed, data, 0o600); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}
	return nil
}

func stateKeyUser(userKey string) string {
	return "user:" + strings.TrimSpace(userKey)
}

func stateKeyCampaign(campaignKey string) string {
	return "campaign:" + strings.TrimSpace(campaignKey)
}

func stateKeyParticipant(campaignKey, participantKey string) string {
	return "participant:" + strings.TrimSpace(campaignKey) + ":" + strings.TrimSpace(participantKey)
}

func stateKeyCharacter(campaignKey, characterKey string) string {
	return "character:" + strings.TrimSpace(campaignKey) + ":" + strings.TrimSpace(characterKey)
}

func stateKeySession(campaignKey, sessionKey string) string {
	return "session:" + strings.TrimSpace(campaignKey) + ":" + strings.TrimSpace(sessionKey)
}

func stateKeyFork(forkKey string) string {
	return "fork:" + strings.TrimSpace(forkKey)
}
