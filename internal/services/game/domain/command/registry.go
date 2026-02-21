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
)

var (
	// ErrCampaignIDRequired indicates a missing campaign id.
	ErrCampaignIDRequired = errors.New("campaign id is required")
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
)

// GatePolicy declares how a command behaves when a session gate is open.
type GatePolicy struct {
	Scope         GateScope
	AllowWhenOpen bool
}

// ActorType identifies the actor who initiated the command.
type ActorType string

const (
	// ActorTypeSystem indicates a system-originated command.
	ActorTypeSystem ActorType = "system"
	// ActorTypeParticipant indicates a participant-originated command.
	ActorTypeParticipant ActorType = "participant"
	// ActorTypeGM indicates a GM-originated command.
	ActorTypeGM ActorType = "gm"
)

// Command captures the canonical envelope used by the domain engine.
//
// Commands are normalized and validated before reaching deciders so business
// rules are applied to stable inputs instead of transport-shaped payloads.
type Command struct {
	CampaignID    string
	Type          Type
	ActorType     ActorType
	ActorID       string
	SessionID     string
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
}

// PayloadValidator validates a payload JSON document.
type PayloadValidator func(json.RawMessage) error

// Registry stores command definitions and validates commands.
//
// Validation here is intentionally strict: malformed commands are rejected once,
// before policy deciders run, to keep behavior deterministic.
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
	cmd.CampaignID = strings.TrimSpace(cmd.CampaignID)
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
		if expectedSystemNamespace, ok := naming.NamespaceFromType(string(cmd.Type)); ok {
			if naming.NormalizeSystemNamespace(cmd.SystemID) != expectedSystemNamespace {
				return Command{}, fmt.Errorf(
					"%w: type %s expects %s, got %s",
					ErrSystemTypeNamespaceMismatch,
					cmd.Type,
					expectedSystemNamespace,
					cmd.SystemID,
				)
			}
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
	return cmd, nil
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
