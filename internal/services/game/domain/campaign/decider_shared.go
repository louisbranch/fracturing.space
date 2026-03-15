package campaign

import (
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
)

func commandDecodeMessage(cmd command.Command, err error) string {
	return fmt.Sprintf("decode %s payload: %v", cmd.Type, err)
}
