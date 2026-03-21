package session

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/session/gate"

// --- Gate payload type aliases ---

type GateOpenedPayload = gate.GateOpenedPayload
type GateResolvedPayload = gate.GateResolvedPayload
type GateResponseRecordedPayload = gate.GateResponseRecordedPayload
type GateAbandonedPayload = gate.GateAbandonedPayload

// --- Gate status type and constant aliases ---

type GateStatus = gate.GateStatus

const (
	GateStatusOpen      = gate.GateStatusOpen
	GateStatusResolved  = gate.GateStatusResolved
	GateStatusAbandoned = gate.GateStatusAbandoned
)

// --- Gate progress type aliases ---

type GateProgress = gate.GateProgress
type GateProgressResponse = gate.GateProgressResponse

// --- Gate projection type aliases ---

type StoredGateMetadata = gate.StoredGateMetadata
type StoredGateResolution = gate.StoredGateResolution

// --- Gate progress constants ---

const (
	GateResponseAuthorityParticipant = gate.GateResponseAuthorityParticipant

	GateResolutionStatePendingResponses = gate.GateResolutionStatePendingResponses
	GateResolutionStateReadyToResolve   = gate.GateResolutionStateReadyToResolve
	GateResolutionStateBlocked          = gate.GateResolutionStateBlocked
	GateResolutionStateManualReview     = gate.GateResolutionStateManualReview
)

// --- Gate label functions ---

var (
	NormalizeGateType   = gate.NormalizeGateType
	NormalizeGateReason = gate.NormalizeGateReason
)

// --- Gate workflow API functions ---

var (
	NormalizeGateWorkflowMetadata = gate.NormalizeGateWorkflowMetadata
	ValidateGateResponse          = gate.ValidateGateResponse
)

// --- Gate progress API functions ---

var (
	BuildInitialGateProgress   = gate.BuildInitialGateProgress
	RecordGateResponseProgress = gate.RecordGateResponseProgress
)

// --- Gate projection metadata functions ---

var (
	MarshalGateMetadataJSON        = gate.MarshalGateMetadataJSON
	DecodeGateMetadataMap          = gate.DecodeGateMetadataMap
	ValidateGateResponseMetadata   = gate.ValidateGateResponseMetadata
	BuildStoredGateMetadata        = gate.BuildStoredGateMetadata
	BuildGateMetadataMapFromStored = gate.BuildGateMetadataMapFromStored
)

// --- Gate projection progress functions ---

var (
	BuildInitialGateProgressState   = gate.BuildInitialGateProgressState
	DecodeGateProgressMap           = gate.DecodeGateProgressMap
	DecodeGateProgress              = gate.DecodeGateProgress
	MarshalGateProgressJSON         = gate.MarshalGateProgressJSON
	RecordGateResponseProgressState = gate.RecordGateResponseProgressState
	BuildGateProgressFromResponses  = gate.BuildGateProgressFromResponses
)

// --- Gate projection resolution functions ---

var (
	MarshalGateResolutionJSON        = gate.MarshalGateResolutionJSON
	BuildGateResolutionMap           = gate.BuildGateResolutionMap
	MarshalGateResolutionMapJSON     = gate.MarshalGateResolutionMapJSON
	DecodeGateResolutionMap          = gate.DecodeGateResolutionMap
	BuildStoredGateResolution        = gate.BuildStoredGateResolution
	BuildGateResolutionMapFromStored = gate.BuildGateResolutionMapFromStored
)

// --- Gate projection JSON helpers ---

var JSONMapFromValue = gate.JSONMapFromValue
