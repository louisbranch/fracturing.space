package sessionrolltransport

import (
	"context"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type actionRollContext struct {
	CampaignID        string
	SessionID         string
	SceneID           string
	CharacterID       string
	Trait             string
	RollKind          pb.RollKind
	Difficulty        int
	Advantage         int
	Disadvantage      int
	ModifierTotal     int
	ModifierMetadata  []workflowtransport.RollModifierMetadata
	Underwater        bool
	HopeSpends        []hopeSpend
	SpendEventCount   int
	SpendTotal        int
	BreathCountdownID string
	CharacterState    projectionstore.DaggerheartCharacterState
}

func (h *Handler) loadSessionActionRollContext(ctx context.Context, in *pb.SessionActionRollRequest) (actionRollContext, error) {
	if err := h.requireActionRollDependencies(); err != nil {
		return actionRollContext{}, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return actionRollContext{}, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return actionRollContext{}, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return actionRollContext{}, err
	}
	trait, err := validate.RequiredID(in.GetTrait(), "trait")
	if err != nil {
		return actionRollContext{}, err
	}

	c, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return actionRollContext{}, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return actionRollContext{}, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(c, "campaign system does not support daggerheart rolls"); err != nil {
		return actionRollContext{}, err
	}

	sess, err := h.deps.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return actionRollContext{}, grpcerror.HandleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return actionRollContext{}, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return actionRollContext{}, err
	}

	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return actionRollContext{}, grpcerror.HandleDomainError(err)
	}

	rollKind := normalizeRollKind(in.GetRollKind())
	advantage := int(in.GetAdvantage())
	disadvantage := int(in.GetDisadvantage())
	if advantage < 0 {
		advantage = 0
	}
	if disadvantage < 0 {
		disadvantage = 0
	}
	if in.GetUnderwater() && rollKind == pb.RollKind_ROLL_KIND_ACTION {
		disadvantage++
	}

	hopeSpends := hopeSpendsFromModifiers(in.GetModifiers())
	spendEventCount, spendTotal := summarizeSpendAmounts(hopeSpends)
	if rollKind == pb.RollKind_ROLL_KIND_REACTION && spendEventCount > 0 {
		return actionRollContext{}, status.Error(codes.InvalidArgument, "reaction rolls cannot spend hope")
	}

	modifierTotal, modifierMetadata := normalizeActionModifiers(in.GetModifiers())
	return actionRollContext{
		CampaignID:        campaignID,
		SessionID:         sessionID,
		SceneID:           strings.TrimSpace(in.GetSceneId()),
		CharacterID:       characterID,
		Trait:             trait,
		RollKind:          rollKind,
		Difficulty:        int(in.GetDifficulty()),
		Advantage:         advantage,
		Disadvantage:      disadvantage,
		ModifierTotal:     modifierTotal,
		ModifierMetadata:  modifierMetadata,
		Underwater:        in.GetUnderwater(),
		HopeSpends:        hopeSpends,
		SpendEventCount:   spendEventCount,
		SpendTotal:        spendTotal,
		BreathCountdownID: strings.TrimSpace(in.GetBreathCountdownId()),
		CharacterState:    state,
	}, nil
}

func summarizeSpendAmounts(spends []hopeSpend) (int, int) {
	count := 0
	total := 0
	for _, spend := range spends {
		if spend.Amount <= 0 {
			continue
		}
		count++
		total += spend.Amount
	}
	return count, total
}
