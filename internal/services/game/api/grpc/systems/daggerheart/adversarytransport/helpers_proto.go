package adversarytransport

import (
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func adversaryToProto(adversary projectionstore.DaggerheartAdversary) *pb.DaggerheartAdversary {
	var sessionID *wrapperspb.StringValue
	if strings.TrimSpace(adversary.SessionID) != "" {
		sessionID = wrapperspb.String(adversary.SessionID)
	}
	return &pb.DaggerheartAdversary{
		Id:              adversary.AdversaryID,
		CampaignId:      adversary.CampaignID,
		Name:            adversary.Name,
		Kind:            adversary.Kind,
		SessionId:       sessionID,
		Notes:           adversary.Notes,
		Hp:              int32(adversary.HP),
		HpMax:           int32(adversary.HPMax),
		Stress:          int32(adversary.Stress),
		StressMax:       int32(adversary.StressMax),
		Evasion:         int32(adversary.Evasion),
		MajorThreshold:  int32(adversary.Major),
		SevereThreshold: int32(adversary.Severe),
		Armor:           int32(adversary.Armor),
		Conditions:      daggerheartConditionsToProto(adversary.Conditions),
		CreatedAt:       timestamppb.New(adversary.CreatedAt),
		UpdatedAt:       timestamppb.New(adversary.UpdatedAt),
	}
}

func daggerheartConditionsToProto(in []string) []pb.DaggerheartCondition {
	out := make([]pb.DaggerheartCondition, 0, len(in))
	for _, condition := range in {
		switch strings.ToLower(strings.TrimSpace(condition)) {
		case "hidden":
			out = append(out, pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN)
		case "vulnerable":
			out = append(out, pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE)
		}
	}
	return out
}
