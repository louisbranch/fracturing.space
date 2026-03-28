package aieval

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext/instructionset"
)

const (
	instructionsSourceEmbedded = "embedded_defaults"
	instructionsSourceFS       = "filesystem_override"
)

// BuildPromptContext resolves a compact prompt-context summary for one eval run.
func BuildPromptContext(profile string, instructionsRoot string) PromptContext {
	profile = strings.TrimSpace(profile)
	root := strings.TrimSpace(instructionsRoot)
	source := instructionsSourceEmbedded
	if root != "" {
		source = instructionsSourceFS
		if absRoot, err := filepath.Abs(root); err == nil {
			root = absRoot
		}
	}

	ctx := PromptContext{
		Profile:            profile,
		InstructionsRoot:   root,
		InstructionsSource: source,
		Summary:            promptSummary(profile, source),
	}
	ctx.InstructionsDigest = promptContextDigest(root)
	return ctx
}

func promptSummary(profile string, source string) string {
	switch strings.TrimSpace(profile) {
	case string(PromptProfileMechanicsHardened):
		if source == instructionsSourceFS {
			return "Mechanics-hardened instruction override loaded from a filesystem root."
		}
		return "Mechanics-hardened instruction profile for the live GM harness."
	default:
		if source == instructionsSourceFS {
			return "Baseline GM instruction profile loaded from a filesystem root."
		}
		return "Baseline GM instruction profile using the repo default instruction bundle."
	}
}

func promptContextDigest(root string) string {
	loader := instructionset.New(root)
	skills, skillsErr := loader.LoadSkills(campaigncontext.DaggerheartSystem)
	interaction, interactionErr := loader.LoadCoreInteraction()
	if skillsErr != nil || interactionErr != nil {
		return ""
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(skills) + "\n---\n" + strings.TrimSpace(interaction)))
	return hex.EncodeToString(sum[:8])
}
