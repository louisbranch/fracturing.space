package statetransport

import (
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/conditiontransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
)

// CharacterStateToProto maps stored Daggerheart character state into the gRPC
// response shape shared by damage, condition, and recovery wrappers.
func CharacterStateToProto(state projectionstore.DaggerheartCharacterState) *pb.DaggerheartCharacterState {
	temporaryArmorBuckets := make([]*pb.DaggerheartTemporaryArmorBucket, 0, len(state.TemporaryArmor))
	for _, bucket := range state.TemporaryArmor {
		temporaryArmorBuckets = append(temporaryArmorBuckets, &pb.DaggerheartTemporaryArmorBucket{
			Source:   bucket.Source,
			Duration: bucket.Duration,
			SourceId: bucket.SourceID,
			Amount:   int32(bucket.Amount),
		})
	}

	return &pb.DaggerheartCharacterState{
		Hp:                    int32(state.Hp),
		Hope:                  int32(state.Hope),
		HopeMax:               int32(state.HopeMax),
		Stress:                int32(state.Stress),
		Armor:                 int32(state.Armor),
		Conditions:            conditiontransport.ConditionsToProto(state.Conditions),
		TemporaryArmorBuckets: temporaryArmorBuckets,
		LifeState:             conditiontransport.LifeStateToProto(state.LifeState),
	}
}

// OptionalInt32 preserves optional roll details in transport responses without
// teaching wrappers to repeat pointer conversion noise.
func OptionalInt32(value *int) *int32 {
	if value == nil {
		return nil
	}
	v := int32(*value)
	return &v
}
