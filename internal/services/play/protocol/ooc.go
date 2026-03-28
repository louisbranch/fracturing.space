package protocol

import (
	"strings"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

// OOCState represents the out-of-character session pause state.
type OOCState struct {
	Open                        bool      `json:"open"`
	Posts                       []OOCPost `json:"posts"`
	ReadyToResumeParticipantIDs []string  `json:"ready_to_resume_participant_ids"`
}

// OOCPost is a single out-of-character message.
type OOCPost struct {
	PostID        string `json:"post_id"`
	ParticipantID string `json:"participant_id"`
	Body          string `json:"body"`
	CreatedAt     string `json:"created_at,omitempty"`
}

// OOCFromGameOOC maps a proto OOCState to protocol.
func OOCFromGameOOC(ooc *gamev1.OOCState) *OOCState {
	if ooc == nil {
		return nil
	}
	posts := make([]OOCPost, 0, len(ooc.GetPosts()))
	for _, p := range ooc.GetPosts() {
		posts = append(posts, OOCPost{
			PostID:        strings.TrimSpace(p.GetPostId()),
			ParticipantID: strings.TrimSpace(p.GetParticipantId()),
			Body:          strings.TrimSpace(p.GetBody()),
			CreatedAt:     FormatTimestamp(p.GetCreatedAt()),
		})
	}
	return &OOCState{
		Open:                        ooc.GetOpen(),
		Posts:                       posts,
		ReadyToResumeParticipantIDs: TrimStringSlice(ooc.GetReadyToResumeParticipantIds()),
	}
}
