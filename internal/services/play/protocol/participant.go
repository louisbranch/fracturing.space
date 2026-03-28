package protocol

import (
	"strings"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
)

// PlayAvatarDeliveryWidthPX is the CDN delivery width used for avatar images
// in the play UI.
const PlayAvatarDeliveryWidthPX = 384

// Participant represents an enriched campaign participant for the browser.
type Participant struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Role         string   `json:"role,omitempty"`
	AvatarURL    string   `json:"avatar_url,omitempty"`
	CharacterIDs []string `json:"character_ids,omitempty"`
}

// ParticipantFromGameParticipant maps a proto Participant to protocol.
func ParticipantFromGameParticipant(assetBaseURL string, p *gamev1.Participant) Participant {
	if p == nil {
		return Participant{}
	}
	avatarEntityID := strings.TrimSpace(p.GetId())
	if avatarEntityID == "" {
		avatarEntityID = strings.TrimSpace(p.GetUserId())
	}
	if avatarEntityID == "" {
		avatarEntityID = strings.TrimSpace(p.GetCampaignId())
	}
	return Participant{
		ID:   strings.TrimSpace(p.GetId()),
		Name: strings.TrimSpace(p.GetName()),
		Role: interactionRoleString(p.GetRole()),
		AvatarURL: websupport.AvatarImageURL(
			assetBaseURL,
			catalog.AvatarRoleParticipant,
			avatarEntityID,
			strings.TrimSpace(p.GetAvatarSetId()),
			strings.TrimSpace(p.GetAvatarAssetId()),
			PlayAvatarDeliveryWidthPX,
		),
	}
}

// CharacterCardPortrait holds avatar rendering data for a character card.
type CharacterCardPortrait struct {
	Alt string `json:"alt"`
	Src string `json:"src,omitempty"`
}

// CharacterCardIdentity holds character kind and pronoun metadata.
type CharacterCardIdentity struct {
	Kind       string   `json:"kind,omitempty"`
	Controller string   `json:"controller,omitempty"`
	Pronouns   string   `json:"pronouns,omitempty"`
	Aliases    []string `json:"aliases,omitempty"`
}

// CharacterInspection holds both card and sheet data for a character.
// Card and Sheet are typed as any because the concrete shape depends on the
// game system. For Daggerheart, Card is DaggerheartCharacterCardData and Sheet
// is DaggerheartCharacterSheetData. New game systems should follow the same
// pattern and document their concrete types here.
type CharacterInspection struct {
	System string `json:"system"`
	Card   any    `json:"card"`
	Sheet  any    `json:"sheet"`
}
