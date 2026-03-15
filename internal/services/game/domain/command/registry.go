// Package command defines the command envelope and validation entry points.
package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	coreencoding "github.com/louisbranch/fracturing.space/internal/services/game/core/encoding"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/naming"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

var (
	// ErrCampaignIDRequired indicates a missing campaign id.
	// Canonical definition in domain/ids; re-exported for caller compatibility.
	ErrCampaignIDRequired = ids.ErrCampaignIDRequired
	// ErrTypeRequired indicates a missing command type.
	ErrTypeRequired = errors.New("command type is required")
	// ErrTypeUnknown indicates an unregistered command type.
	ErrTypeUnknown = errors.New("command type is not registered")
	// ErrSystemMetadataRequired indicates missing system metadata for system commands.
	ErrSystemMetadataRequired = errors.New("system metadata is required for system commands")
	// ErrSystemMetadataForbidden indicates system metadata on core commands.
	ErrSystemMetadataForbidden = errors.New("system metadata must be empty for core commands")
	// ErrSystemTypeNamespaceMismatch indicates system metadata does not match type namespace.
	ErrSystemTypeNamespaceMismatch = errors.New("system id must match command type namespace")
	// ErrActorTypeInvalid indicates an unknown actor type.
	ErrActorTypeInvalid = errors.New("actor type is invalid")
	// ErrActorIDRequired indicates a missing actor id for participant/gm.
	ErrActorIDRequired = errors.New("actor id is required for participant or gm")
	// ErrPayloadInvalid indicates malformed payload JSON.
	ErrPayloadInvalid = errors.New("payload json must be valid")
)

// Type identifies a stable command semantic used by both API transport and
// domain deciders. A change in this value is a behavior contract.
type Type string

// Owner identifies whether a command type is handled by core game rules or a
// pluggable game-system module.
type Owner string

const (
	// OwnerCore indicates a core domain command.
	OwnerCore Owner = "core"
	// OwnerSystem indicates a system-owned command.
	OwnerSystem Owner = "system"
)

// GateScope declares when a command is subject to a session decision gate.
type GateScope string

const (
	// GateScopeNone indicates the command is never gated.
	GateScopeNone GateScope = "none"
	// GateScopeSession indicates the command is gated when a session gate is open.
	GateScopeSession GateScope = "session"
	// GateScopeScene indicates the command is gated when a scene gate is open.
	GateScopeScene GateScope = "scene"
)

// GatePolicy declares how a command behaves when a session gate is open.
type GatePolicy struct {
	Scope         GateScope
	AllowWhenOpen bool
}

// ActiveSessionClassification declares how a command behaves while a campaign
// has an active session.
type ActiveSessionClassification string

const (
	// ActiveSessionClassificationAllowed permits command execution while active.
	ActiveSessionClassificationAllowed ActiveSessionClassification = "allowed"
	// ActiveSessionClassificationBlocked rejects command execution while active.
	ActiveSessionClassificationBlocked ActiveSessionClassification = "blocked"
)

// ActiveSessionPolicy declares active-session behavior for a command type.
type ActiveSessionPolicy struct {
	Classification         ActiveSessionClassification
	AllowInGameSystemActor bool
}

// TargetEntityPolicy declares how a command identifies the aggregate entity it
// targets before the engine routes it.
type TargetEntityPolicy struct {
	EntityType   string
	PayloadField string
}

// AllowedDuringActiveSession returns metadata for a command that stays allowed
// while a campaign session is active.
func AllowedDuringActiveSession() ActiveSessionPolicy {
	return ActiveSessionPolicy{Classification: ActiveSessionClassificationAllowed}
}

// BlockedDuringActiveSession returns metadata for a command that is blocked
// while a campaign session is active.
func BlockedDuringActiveSession() ActiveSessionPolicy {
	return ActiveSessionPolicy{Classification: ActiveSessionClassificationBlocked}
}

// BlockedDuringActiveSessionExceptInGameSystemActor returns metadata for a
// command that is blocked during active sessions except when executed by a
// system actor already scoped to the active session.
func BlockedDuringActiveSessionExceptInGameSystemActor() ActiveSessionPolicy {
	return ActiveSessionPolicy{
		Classification:         ActiveSessionClassificationBlocked,
		AllowInGameSystemActor: true,
	}
}

// TargetEntity returns metadata for commands that target one aggregate entity
// type and may need payload fallback when the transport did not set EntityID.
func TargetEntity(entityType, payloadField string) TargetEntityPolicy {
	return TargetEntityPolicy{
		EntityType:   strings.TrimSpace(entityType),
		PayloadField: strings.TrimSpace(payloadField),
	}
}

// ActorType identifies the actor who initiated the command.
// The canonical definition lives in the event package; this alias keeps
// existing command-layer callers working without import changes.
type ActorType = event.ActorType

const (
	// ActorTypeSystem indicates a system-originated command.
	ActorTypeSystem = event.ActorTypeSystem
	// ActorTypeParticipant indicates a participant-originated command.
	ActorTypeParticipant = event.ActorTypeParticipant
	// ActorTypeGM indicates a GM-originated command.
	ActorTypeGM = event.ActorTypeGM
)

// Command captures the canonical envelope used by the domain engine.
//
// Commands are normalized and validated before reaching deciders so business
// rules are applied to stable inputs instead of transport-shaped payloads.
type Command struct {
	CampaignID    ids.CampaignID
	Type          Type
	ActorType     ActorType
	ActorID       string
	SessionID     ids.SessionID
	SceneID       ids.SceneID
	RequestID     string
	InvocationID  string
	EntityType    string
	EntityID      string
	SystemID      string
	SystemVersion string
	CorrelationID string
	CausationID   string
	PayloadJSON   []byte
}

// Definition registers metadata for a command type.
//
// The definition is the single place that declares:
//   - who owns the command (core/system),
//   - which payload validator runs,
//   - and whether session gates apply.
type Definition struct {
	Type            Type
	Owner           Owner
	ValidatePayload PayloadValidator
	Gate            GatePolicy
	ActiveSession   ActiveSessionPolicy
	Target          TargetEntityPolicy
}

// PayloadValidator validates a payload JSON document.
type PayloadValidator func(json.RawMessage) error

// Registry stores command definitions and validates commands.
//
// Validation here is intentionally strict: malformed commands are rejected once,
// before policy deciders run, to keep behavior deterministic.
//
// A Registry must be fully populated before use. After initialization,
// all methods are safe for concurrent read access.
type Registry struct {
	definitions map[Type]Definition
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{definitions: make(map[Type]Definition)}
}

// Register adds a new command type definition to the registry.
func (r *Registry) Register(def Definition) error {
	if r == nil {
		return errors.New("registry is required")
	}
	def.Type = Type(strings.TrimSpace(string(def.Type)))
	if def.Type == "" {
		return ErrTypeRequired
	}
	switch def.Owner {
	case OwnerCore, OwnerSystem:
		// allowed
	default:
		return fmt.Errorf("owner must be core or system")
	}
	def.Target = TargetEntity(def.Target.EntityType, def.Target.PayloadField)
	if r.definitions == nil {
		r.definitions = make(map[Type]Definition)
	}
	if _, exists := r.definitions[def.Type]; exists {
		return fmt.Errorf("command type already registered: %s", def.Type)
	}
	r.definitions[def.Type] = def
	return nil
}

// ValidateForDecision validates and normalizes a command before decision handling.
//
// It is the boundary that protects deciders from transport noise:
// canonical JSON, ownership checks, actor identity defaults, and payload
// validation all happen before domain logic sees the command.
func (r *Registry) ValidateForDecision(cmd Command) (Command, error) {
	if r == nil {
		return Command{}, errors.New("registry is required")
	}
	cmd.CampaignID = ids.CampaignID(strings.TrimSpace(string(cmd.CampaignID)))
	if cmd.CampaignID == "" {
		return Command{}, ErrCampaignIDRequired
	}
	cmd.Type = Type(strings.TrimSpace(string(cmd.Type)))
	if cmd.Type == "" {
		return Command{}, ErrTypeRequired
	}
	def, ok := r.definitions[cmd.Type]
	if !ok {
		return Command{}, ErrTypeUnknown
	}

	cmd.SystemID = strings.TrimSpace(cmd.SystemID)
	cmd.SystemVersion = strings.TrimSpace(cmd.SystemVersion)
	switch def.Owner {
	case OwnerSystem:
		if cmd.SystemID == "" || cmd.SystemVersion == "" {
			return Command{}, ErrSystemMetadataRequired
		}
		if err := naming.ValidateSystemNamespace(string(cmd.Type), cmd.SystemID); err != nil {
			return Command{}, fmt.Errorf("%w: %w", ErrSystemTypeNamespaceMismatch, err)
		}
	case OwnerCore:
		if cmd.SystemID != "" || cmd.SystemVersion != "" {
			return Command{}, ErrSystemMetadataForbidden
		}
	}

	cmd.ActorType = ActorType(strings.TrimSpace(string(cmd.ActorType)))
	if cmd.ActorType == "" {
		cmd.ActorType = ActorTypeSystem
	}
	switch cmd.ActorType {
	case ActorTypeSystem, ActorTypeParticipant, ActorTypeGM:
		// allowed
	default:
		return Command{}, ErrActorTypeInvalid
	}
	cmd.ActorID = strings.TrimSpace(cmd.ActorID)
	if (cmd.ActorType == ActorTypeParticipant || cmd.ActorType == ActorTypeGM) && cmd.ActorID == "" {
		return Command{}, ErrActorIDRequired
	}

	if len(cmd.PayloadJSON) == 0 {
		cmd.PayloadJSON = []byte("{}")
	}
	if !json.Valid(cmd.PayloadJSON) {
		return Command{}, ErrPayloadInvalid
	}

	canonical, err := coreencoding.CanonicalJSON(json.RawMessage(cmd.PayloadJSON))
	if err != nil {
		return Command{}, fmt.Errorf("canonical payload json: %w", err)
	}
	cmd.PayloadJSON = canonical
	if def.ValidatePayload != nil {
		if err := def.ValidatePayload(json.RawMessage(cmd.PayloadJSON)); err != nil {
			return Command{}, fmt.Errorf("payload invalid: %w", err)
		}
	}
	normalizeTargetEntity(&cmd, def.Target)
	return cmd, nil
}

func normalizeTargetEntity(cmd *Command, target TargetEntityPolicy) {
	if cmd == nil {
		return
	}
	cmd.EntityID = strings.TrimSpace(cmd.EntityID)
	cmd.EntityType = strings.TrimSpace(cmd.EntityType)
	if cmd.EntityID == "" && target.PayloadField != "" {
		cmd.EntityID = payloadStringField(cmd.PayloadJSON, target.PayloadField)
	}
	if cmd.EntityID != "" && cmd.EntityType == "" && target.EntityType != "" {
		cmd.EntityType = target.EntityType
	}
}

func payloadStringField(payloadJSON []byte, field string) string {
	field = strings.TrimSpace(field)
	if field == "" {
		return ""
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payloadJSON, &raw); err != nil {
		return ""
	}
	fieldJSON, ok := raw[field]
	if !ok {
		return ""
	}
	var value string
	if err := json.Unmarshal(fieldJSON, &value); err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

// Definition returns the command definition for a given type.
func (r *Registry) Definition(cmdType Type) (Definition, bool) {
	if r == nil {
		return Definition{}, false
	}
	cmdType = Type(strings.TrimSpace(string(cmdType)))
	if cmdType == "" {
		return Definition{}, false
	}
	def, ok := r.definitions[cmdType]
	return def, ok
}

// ListDefinitions returns a stable, sorted snapshot of registered definitions.
func (r *Registry) ListDefinitions() []Definition {
	if r == nil || len(r.definitions) == 0 {
		return nil
	}
	definitions := make([]Definition, 0, len(r.definitions))
	for _, definition := range r.definitions {
		definitions = append(definitions, definition)
	}
	sort.Slice(definitions, func(i, j int) bool {
		return string(definitions[i].Type) < string(definitions[j].Type)
	})
	return definitions
}
