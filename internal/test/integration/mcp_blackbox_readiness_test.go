//go:build integration

package integration

import (
	"fmt"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/tools/seed"
)

type blackboxInvoker func(request map[string]any) (map[string]any, []byte, error)

func maybeEnsureSessionStartReadinessForBlackbox(
	t *testing.T,
	stepName string,
	request map[string]any,
	captures map[string]string,
	invoke blackboxInvoker,
) {
	t.Helper()

	if !isSessionStartToolCall(request) {
		return
	}
	campaignID := strings.TrimSpace(captures["campaign_id"])
	ownerID := strings.TrimSpace(captures["owner_id"])
	if campaignID == "" || ownerID == "" {
		return
	}

	readReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      fmt.Sprintf("%s-readiness-characters", stepName),
		"method":  "resources/read",
		"params": map[string]any{
			"uri": "campaign://" + campaignID + "/characters",
		},
	}
	readResp, readRaw, err := invoke(readReq)
	if err != nil {
		t.Fatalf("%s readiness character read: %v", stepName, err)
	}
	if toolErr := seed.FormatJSONRPCError(readResp); toolErr != "" {
		t.Fatalf("%s readiness character read error: %s (response=%s)", stepName, toolErr, string(readRaw))
	}

	playerParticipantIDs := readPlayerParticipantIDsForBlackbox(t, stepName, campaignID, invoke)
	characters, err := readCharactersFromResourceResponse(readResp)
	if err != nil {
		t.Fatalf("%s readiness parse characters: %v (response=%s)", stepName, err, string(readRaw))
	}

	for idx, character := range characters {
		characterID := strings.TrimSpace(character.id)
		if characterID == "" || strings.TrimSpace(character.participantID) != "" {
			continue
		}
		setReq := map[string]any{
			"jsonrpc": "2.0",
			"id":      fmt.Sprintf("%s-readiness-set-%d", stepName, idx),
			"method":  "tools/call",
			"params": map[string]any{
				"name": "character_control_set",
				"arguments": map[string]any{
					"campaign_id":    campaignID,
					"character_id":   characterID,
					"participant_id": ownerID,
				},
			},
		}
		setResp, setRaw, callErr := invoke(setReq)
		if callErr != nil {
			t.Fatalf("%s readiness set control for %s: %v", stepName, characterID, callErr)
		}
		if toolErr := seed.FormatJSONRPCError(setResp); toolErr != "" {
			t.Fatalf("%s readiness set control for %s error: %s (response=%s)", stepName, characterID, toolErr, string(setRaw))
		}
		isError, _ := seed.LookupJSONPath(setResp, "result.isError")
		if isError == true {
			t.Fatalf("%s readiness set control for %s returned tool error (response=%s)", stepName, characterID, string(setRaw))
		}
	}

	readResp, readRaw, err = invoke(readReq)
	if err != nil {
		t.Fatalf("%s readiness character re-read: %v", stepName, err)
	}
	if toolErr := seed.FormatJSONRPCError(readResp); toolErr != "" {
		t.Fatalf("%s readiness character re-read error: %s (response=%s)", stepName, toolErr, string(readRaw))
	}
	characters, err = readCharactersFromResourceResponse(readResp)
	if err != nil {
		t.Fatalf("%s readiness parse characters after assignments: %v (response=%s)", stepName, err, string(readRaw))
	}

	playerCharacterCount := make(map[string]int, len(playerParticipantIDs))
	for _, playerID := range playerParticipantIDs {
		playerCharacterCount[playerID] = 0
	}
	ownerControlledCharacters := make([]string, 0, len(characters))
	for _, character := range characters {
		if _, ok := playerCharacterCount[character.participantID]; ok {
			playerCharacterCount[character.participantID]++
		}
		if strings.TrimSpace(character.participantID) == ownerID {
			ownerControlledCharacters = append(ownerControlledCharacters, character.id)
		}
	}

	createdCounter := 0
	for _, playerID := range playerParticipantIDs {
		if playerCharacterCount[playerID] > 0 {
			continue
		}
		characterID := ""
		if len(ownerControlledCharacters) > 0 {
			characterID = ownerControlledCharacters[0]
			ownerControlledCharacters = ownerControlledCharacters[1:]
		} else {
			createdCounter++
			createReq := map[string]any{
				"jsonrpc": "2.0",
				"id":      fmt.Sprintf("%s-readiness-create-%d", stepName, createdCounter),
				"method":  "tools/call",
				"params": map[string]any{
					"name": "character_create",
					"arguments": map[string]any{
						"campaign_id": campaignID,
						"name":        fmt.Sprintf("Readiness Character %d", createdCounter),
						"kind":        "PC",
					},
				},
			}
			createResp, createRaw, createErr := invoke(createReq)
			if createErr != nil {
				t.Fatalf("%s readiness create character for player %s: %v", stepName, playerID, createErr)
			}
			if toolErr := seed.FormatJSONRPCError(createResp); toolErr != "" {
				t.Fatalf("%s readiness create character for player %s error: %s (response=%s)", stepName, playerID, toolErr, string(createRaw))
			}
			isError, _ := seed.LookupJSONPath(createResp, "result.isError")
			if isError == true {
				t.Fatalf("%s readiness create character for player %s returned tool error (response=%s)", stepName, playerID, string(createRaw))
			}
			capturedID, captureErr := seed.CaptureFromPaths(createResp, seed.CaptureDefaults["character"])
			if captureErr != nil {
				t.Fatalf("%s readiness capture created character id for player %s: %v (response=%s)", stepName, playerID, captureErr, string(createRaw))
			}
			characterID = capturedID
		}

		setReq := map[string]any{
			"jsonrpc": "2.0",
			"id":      fmt.Sprintf("%s-readiness-set-player-%d", stepName, createdCounter),
			"method":  "tools/call",
			"params": map[string]any{
				"name": "character_control_set",
				"arguments": map[string]any{
					"campaign_id":    campaignID,
					"character_id":   characterID,
					"participant_id": playerID,
				},
			},
		}
		setResp, setRaw, callErr := invoke(setReq)
		if callErr != nil {
			t.Fatalf("%s readiness set control for player %s character %s: %v", stepName, playerID, characterID, callErr)
		}
		if toolErr := seed.FormatJSONRPCError(setResp); toolErr != "" {
			t.Fatalf("%s readiness set control for player %s character %s error: %s (response=%s)", stepName, playerID, characterID, toolErr, string(setRaw))
		}
		isError, _ := seed.LookupJSONPath(setResp, "result.isError")
		if isError == true {
			t.Fatalf("%s readiness set control for player %s character %s returned tool error (response=%s)", stepName, playerID, characterID, string(setRaw))
		}
	}
}

func readPlayerParticipantIDsForBlackbox(
	t *testing.T,
	stepName string,
	campaignID string,
	invoke blackboxInvoker,
) []string {
	t.Helper()

	readReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      fmt.Sprintf("%s-readiness-participants", stepName),
		"method":  "resources/read",
		"params": map[string]any{
			"uri": "campaign://" + campaignID + "/participants",
		},
	}
	resp, raw, err := invoke(readReq)
	if err != nil {
		t.Fatalf("%s readiness participant read: %v", stepName, err)
	}
	if toolErr := seed.FormatJSONRPCError(resp); toolErr != "" {
		t.Fatalf("%s readiness participant read error: %s (response=%s)", stepName, toolErr, string(raw))
	}

	textAny, err := seed.LookupJSONPath(resp, "result.contents[0].text")
	if err != nil {
		t.Fatalf("%s readiness participant read parse: %v (response=%s)", stepName, err, string(raw))
	}
	text, ok := textAny.(string)
	if !ok {
		t.Fatalf("%s readiness participant read parse: text payload is not a string", stepName)
	}
	decoded, err := seed.DecodeJSONValue([]byte(text))
	if err != nil {
		t.Fatalf("%s readiness participant read decode: %v", stepName, err)
	}
	obj, ok := decoded.(map[string]any)
	if !ok {
		t.Fatalf("%s readiness participant payload is not an object", stepName)
	}
	participantsAny, ok := obj["participants"]
	if !ok {
		return nil
	}
	participants, ok := participantsAny.([]any)
	if !ok {
		t.Fatalf("%s readiness participant payload participants is not an array", stepName)
	}
	out := make([]string, 0, len(participants))
	for _, item := range participants {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		role := strings.TrimSpace(firstNonEmptyStringField(entry, "role", "participant_role", "participantRole"))
		if role != "PLAYER" {
			continue
		}
		id := strings.TrimSpace(stringField(entry, "id"))
		if id == "" {
			continue
		}
		out = append(out, id)
	}
	return out
}

type resourceCharacter struct {
	id            string
	participantID string
}

func readCharactersFromResourceResponse(response map[string]any) ([]resourceCharacter, error) {
	textAny, err := seed.LookupJSONPath(response, "result.contents[0].text")
	if err != nil {
		return nil, err
	}
	text, ok := textAny.(string)
	if !ok {
		return nil, fmt.Errorf("resource text is not a string")
	}
	decoded, err := seed.DecodeJSONValue([]byte(text))
	if err != nil {
		return nil, err
	}
	obj, ok := decoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("resource payload is not an object")
	}
	arrayAny, ok := obj["characters"]
	if !ok {
		return nil, nil
	}
	array, ok := arrayAny.([]any)
	if !ok {
		return nil, fmt.Errorf("characters is not an array")
	}
	out := make([]resourceCharacter, 0, len(array))
	for _, item := range array {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, resourceCharacter{
			id:            stringField(entry, "id"),
			participantID: firstNonEmptyStringField(entry, "participant_id", "participantId"),
		})
	}
	return out, nil
}

func isSessionStartToolCall(request map[string]any) bool {
	method, _ := request["method"].(string)
	if method != "tools/call" {
		return false
	}
	params, _ := request["params"].(map[string]any)
	name, _ := params["name"].(string)
	return name == "session_start"
}

func stringField(obj map[string]any, key string) string {
	v, _ := obj[key]
	s, _ := v.(string)
	return s
}

func firstNonEmptyStringField(obj map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(stringField(obj, key)); value != "" {
			return value
		}
	}
	return ""
}
