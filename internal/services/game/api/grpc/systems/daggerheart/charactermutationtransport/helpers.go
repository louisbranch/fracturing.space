package charactermutationtransport

import (
	"fmt"
	"strings"

	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
)

func campaignSupportsDaggerheart(record storage.CampaignRecord) bool {
	systemID, ok := systembridge.NormalizeSystemID(record.System.String())
	return ok && systemID == systembridge.SystemIDDaggerheart
}

func requireDaggerheartSystem(record storage.CampaignRecord, unsupportedMessage string) error {
	if campaignSupportsDaggerheart(record) {
		return nil
	}
	return status.Error(codes.FailedPrecondition, unsupportedMessage)
}

func requireDaggerheartSystemf(record storage.CampaignRecord, unsupportedFormat string, args ...any) error {
	return requireDaggerheartSystem(record, fmt.Sprintf(strings.TrimSpace(unsupportedFormat), args...))
}

func handleDomainError(err error) error {
	return grpcerror.HandleDomainError(err)
}

func tierForLevel(level int) int {
	switch {
	case level <= 1:
		return 1
	case level <= 4:
		return 2
	case level <= 7:
		return 3
	default:
		return 4
	}
}
