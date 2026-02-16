package scenario

import (
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

type scenarioEnv struct {
	campaignClient    gamev1.CampaignServiceClient
	sessionClient     gamev1.SessionServiceClient
	characterClient   gamev1.CharacterServiceClient
	participantClient gamev1.ParticipantServiceClient
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
	ownerParticipantID   string
	sessionID            string
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
