package interactiontransport

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
)

const (
	commandTypeSessionSceneActivate          command.Type = commandids.SessionSceneActivate
	commandTypeSessionCharacterControllerSet command.Type = commandids.SessionCharacterControllerSet
	commandTypeSessionGMAuthoritySet         command.Type = commandids.SessionGMAuthoritySet
	commandTypeSessionOOCOpen                command.Type = commandids.SessionOOCOpen
	commandTypeSessionOOCPost                command.Type = commandids.SessionOOCPost
	commandTypeSessionOOCReadyMark           command.Type = commandids.SessionOOCReadyMark
	commandTypeSessionOOCReadyClear          command.Type = commandids.SessionOOCReadyClear
	commandTypeSessionOOCClose               command.Type = commandids.SessionOOCClose
	commandTypeSessionOOCResolve             command.Type = commandids.SessionOOCResolve
	commandTypeSessionAITurnQueue            command.Type = commandids.SessionAITurnQueue
	commandTypeSessionAITurnStart            command.Type = commandids.SessionAITurnStart
	commandTypeSessionAITurnFail             command.Type = commandids.SessionAITurnFail
	commandTypeSessionAITurnClear            command.Type = commandids.SessionAITurnClear

	commandTypeScenePlayerPhaseStart            command.Type = commandids.ScenePlayerPhaseStart
	commandTypeScenePlayerPhasePost             command.Type = commandids.ScenePlayerPhasePost
	commandTypeScenePlayerPhaseYield            command.Type = commandids.ScenePlayerPhaseYield
	commandTypeScenePlayerPhaseUnyield          command.Type = commandids.ScenePlayerPhaseUnyield
	commandTypeScenePlayerPhaseAccept           command.Type = commandids.ScenePlayerPhaseAccept
	commandTypeScenePlayerPhaseRequestRevisions command.Type = commandids.ScenePlayerPhaseRequestRevisions
	commandTypeScenePlayerPhaseEnd              command.Type = commandids.ScenePlayerPhaseEnd
	commandTypeSceneGMInteractionCommit         command.Type = commandids.SceneGMInteractionCommit
)
