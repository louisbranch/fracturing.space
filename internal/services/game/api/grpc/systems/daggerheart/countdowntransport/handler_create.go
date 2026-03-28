package countdowntransport

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"strings"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) CreateSceneCountdown(ctx context.Context, in *pb.DaggerheartCreateSceneCountdownRequest) (CreateResult, error) {
	if in == nil {
		return CreateResult{}, status.Error(codes.InvalidArgument, "create scene countdown request is required")
	}
	if err := h.requireDependencies(); err != nil {
		return CreateResult{}, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return CreateResult{}, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return CreateResult{}, err
	}
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return CreateResult{}, err
	}
	name, err := validate.RequiredID(in.GetName(), "name")
	if err != nil {
		return CreateResult{}, err
	}
	tone, err := countdownToneFromProto(in.GetTone())
	if err != nil {
		return CreateResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	policy, err := countdownPolicyFromProto(in.GetAdvancementPolicy())
	if err != nil {
		return CreateResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	loopBehavior, err := countdownLoopBehaviorFromProto(in.GetLoopBehavior())
	if err != nil {
		return CreateResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := h.validateCampaignSession(ctx, campaignID, sessionID, campaign.CampaignOpSessionAction, "campaign system does not support daggerheart scene countdowns"); err != nil {
		return CreateResult{}, err
	}
	countdownID, err := h.resolveCountdownID(strings.TrimSpace(in.GetCountdownId()))
	if err != nil {
		return CreateResult{}, err
	}
	if err := h.ensureCountdownMissing(ctx, campaignID, countdownID); err != nil {
		return CreateResult{}, err
	}
	startingValue, startingRoll, err := resolveStartingValue(in.GetFixedStartingValue(), in.GetRandomizedStart())
	if err != nil {
		return CreateResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	payloadJSON, err := json.Marshal(daggerheartpayload.SceneCountdownCreatePayload{
		SessionID:         ids.SessionID(sessionID),
		SceneID:           ids.SceneID(sceneID),
		CountdownID:       dhids.CountdownID(countdownID),
		Name:              name,
		Tone:              tone,
		AdvancementPolicy: policy,
		StartingValue:     startingValue,
		RemainingValue:    startingValue,
		LoopBehavior:      loopBehavior,
		Status:            rules.CountdownStatusActive,
		LinkedCountdownID: dhids.CountdownID(strings.TrimSpace(in.GetLinkedCountdownId())),
		StartingRoll:      startingRoll,
	})
	if err != nil {
		return CreateResult{}, grpcerror.Internal("encode scene countdown payload", err)
	}
	if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartSceneCountdownCreate,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "scene_countdown",
		EntityID:        countdownID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "scene countdown create did not emit an event",
		ApplyErrMessage: "apply scene countdown created event",
	}); err != nil {
		return CreateResult{}, err
	}
	countdown, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if err != nil {
		return CreateResult{}, grpcerror.Internal("load scene countdown", err)
	}
	return CreateResult{Countdown: countdown}, nil
}

func (h *Handler) CreateCampaignCountdown(ctx context.Context, in *pb.DaggerheartCreateCampaignCountdownRequest) (CreateResult, error) {
	if in == nil {
		return CreateResult{}, status.Error(codes.InvalidArgument, "create campaign countdown request is required")
	}
	if err := h.requireDependencies(); err != nil {
		return CreateResult{}, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return CreateResult{}, err
	}
	name, err := validate.RequiredID(in.GetName(), "name")
	if err != nil {
		return CreateResult{}, err
	}
	tone, err := countdownToneFromProto(in.GetTone())
	if err != nil {
		return CreateResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	policy, err := countdownPolicyFromProto(in.GetAdvancementPolicy())
	if err != nil {
		return CreateResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	loopBehavior, err := countdownLoopBehaviorFromProto(in.GetLoopBehavior())
	if err != nil {
		return CreateResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := h.validateCampaignMutate(ctx, campaignID, "campaign system does not support daggerheart campaign countdowns"); err != nil {
		return CreateResult{}, err
	}
	countdownID, err := h.resolveCountdownID(strings.TrimSpace(in.GetCountdownId()))
	if err != nil {
		return CreateResult{}, err
	}
	if err := h.ensureCountdownMissing(ctx, campaignID, countdownID); err != nil {
		return CreateResult{}, err
	}
	startingValue, startingRoll, err := resolveStartingValue(in.GetFixedStartingValue(), in.GetRandomizedStart())
	if err != nil {
		return CreateResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	payloadJSON, err := json.Marshal(daggerheartpayload.CampaignCountdownCreatePayload{
		CountdownID:       dhids.CountdownID(countdownID),
		Name:              name,
		Tone:              tone,
		AdvancementPolicy: policy,
		StartingValue:     startingValue,
		RemainingValue:    startingValue,
		LoopBehavior:      loopBehavior,
		Status:            rules.CountdownStatusActive,
		LinkedCountdownID: dhids.CountdownID(strings.TrimSpace(in.GetLinkedCountdownId())),
		StartingRoll:      startingRoll,
	})
	if err != nil {
		return CreateResult{}, grpcerror.Internal("encode campaign countdown payload", err)
	}
	if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartCampaignCountdownCreate,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "campaign_countdown",
		EntityID:        countdownID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "campaign countdown create did not emit an event",
		ApplyErrMessage: "apply campaign countdown created event",
	}); err != nil {
		return CreateResult{}, err
	}
	countdown, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if err != nil {
		return CreateResult{}, grpcerror.Internal("load campaign countdown", err)
	}
	return CreateResult{Countdown: countdown}, nil
}

func (h *Handler) resolveCountdownID(countdownID string) (string, error) {
	if countdownID != "" {
		return countdownID, nil
	}
	value, err := h.deps.NewID()
	if err != nil {
		return "", grpcerror.Internal("generate countdown id", err)
	}
	return value, nil
}

func (h *Handler) ensureCountdownMissing(ctx context.Context, campaignID, countdownID string) error {
	_, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "load campaign countdown"); lookupErr != nil {
		return lookupErr
	}
	if err == nil {
		return status.Error(codes.FailedPrecondition, "countdown already exists")
	}
	return nil
}

func resolveStartingValue(fixed int32, randomized *pb.DaggerheartCountdownRandomizedStart) (int, *daggerheartpayload.CountdownStartingRollPayload, error) {
	if randomized == nil {
		if fixed <= 0 {
			return 0, nil, errors.New("fixed_starting_value must be positive")
		}
		return int(fixed), nil, nil
	}
	min := int(randomized.GetMin())
	max := int(randomized.GetMax())
	if min <= 0 || max < min {
		return 0, nil, errors.New("randomized_start range is invalid")
	}
	seed := time.Now().UnixNano()
	if randomized.Rng != nil && randomized.Rng.Seed != nil {
		seed = int64(randomized.Rng.GetSeed())
	}
	r := rand.New(rand.NewSource(seed))
	value := min + r.Intn(max-min+1)
	return value, &daggerheartpayload.CountdownStartingRollPayload{Min: min, Max: max, Value: value}, nil
}
