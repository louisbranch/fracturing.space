package scenario

import (
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

type scenarioEnv struct {
	campaignClient    gamev1.CampaignServiceClient
	sessionClient     gamev1.SessionServiceClient
	sceneClient       gamev1.SceneServiceClient
	characterClient   gamev1.CharacterServiceClient
	participantClient gamev1.ParticipantServiceClient
	interactionClient gamev1.InteractionServiceClient
	snapshotClient    gamev1.SnapshotServiceClient
	eventClient       gamev1.EventServiceClient
	daggerheartClient daggerheartv1.DaggerheartServiceClient
	userID            string
}

type actionRollResult struct {
	rollSeq    uint64
	hopeDie    int
	fearDie    int
	total      int
	difficulty int
	success    bool
	crit       bool
}

type scenarioState struct {
	campaignID           string
	campaignSystem       commonv1.GameSystem
	ownerParticipantID   string
	sessionID            string
	activeSceneID        string
	scenes               map[string]string
	actors               map[string]string
	participants         map[string]string
	adversaries          map[string]string
	countdowns           map[string]string
	gmFear               int
	userID               string
	lastRollSeq          uint64
	lastDamageRollSeq    uint64
	lastAdversaryRollSeq uint64
	rollOutcomes         map[uint64]actionRollResult
}
