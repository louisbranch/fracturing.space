package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
)

const (
	commandTypeSessionActiveSceneSet command.Type = commandids.SessionActiveSceneSet
	commandTypeSessionGMAuthoritySet command.Type = commandids.SessionGMAuthoritySet
	commandTypeSessionOOCPause       command.Type = commandids.SessionOOCPause
	commandTypeSessionOOCPost        command.Type = commandids.SessionOOCPost
	commandTypeSessionOOCReadyMark   command.Type = commandids.SessionOOCReadyMark
	commandTypeSessionOOCReadyClear  command.Type = commandids.SessionOOCReadyClear
	commandTypeSessionOOCResume      command.Type = commandids.SessionOOCResume
	commandTypeSessionAITurnQueue    command.Type = commandids.SessionAITurnQueue
	commandTypeSessionAITurnStart    command.Type = commandids.SessionAITurnStart
	commandTypeSessionAITurnFail     command.Type = commandids.SessionAITurnFail
	commandTypeSessionAITurnClear    command.Type = commandids.SessionAITurnClear

	commandTypeScenePlayerPhaseStart            command.Type = commandids.ScenePlayerPhaseStart
	commandTypeScenePlayerPhasePost             command.Type = commandids.ScenePlayerPhasePost
	commandTypeScenePlayerPhaseYield            command.Type = commandids.ScenePlayerPhaseYield
	commandTypeScenePlayerPhaseUnyield          command.Type = commandids.ScenePlayerPhaseUnyield
	commandTypeScenePlayerPhaseAccept           command.Type = commandids.ScenePlayerPhaseAccept
	commandTypeScenePlayerPhaseRequestRevisions command.Type = commandids.ScenePlayerPhaseRequestRevisions
	commandTypeScenePlayerPhaseEnd              command.Type = commandids.ScenePlayerPhaseEnd
	commandTypeSceneGMOutputCommit              command.Type = commandids.SceneGMOutputCommit
)
