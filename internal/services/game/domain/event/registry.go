package event

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	coreencoding "github.com/louisbranch/fracturing.space/internal/services/game/core/encoding"
)

var (
	// ErrCampaignIDRequired indicates a missing campaign id.
	ErrCampaignIDRequired = errors.New("campaign id is required")
	// ErrTypeRequired indicates a missing event type.
	ErrTypeRequired = errors.New("event type is required")
	// ErrTypeUnknown indicates an unregistered event type.
	ErrTypeUnknown = errors.New("event type is not registered")
	// ErrActorTypeInvalid indicates an unknown actor type.
	ErrActorTypeInvalid = errors.New("actor type is invalid")
	// ErrActorIDRequired indicates a missing actor id for participant/gm.
	ErrActorIDRequired = errors.New("actor id is required for participant or gm")
	// ErrEntityTypeRequired indicates a missing entity type for addressed events.
	ErrEntityTypeRequired = errors.New("entity type is required")
	// ErrEntityIDRequired indicates a missing entity id for addressed events.
	ErrEntityIDRequired = errors.New("entity id is required")
	// ErrSystemMetadataRequired indicates missing system metadata for system events.
	ErrSystemMetadataRequired = errors.New("system metadata is required for system events")
	// ErrSystemMetadataForbidden indicates system metadata on core events.
	ErrSystemMetadataForbidden = errors.New("system metadata must be empty for core events")
	// ErrPayloadInvalid indicates malformed payload JSON.
	ErrPayloadInvalid = errors.New("payload json must be valid")
	// ErrStorageFieldsSet indicates storage-assigned fields were pre-set.
	ErrStorageFieldsSet = errors.New("storage-assigned fields must be empty")
)

// Type identifies a stable event semantic used by API transport and projections.
//
// Event names are part of the write-path contract; changing one affects
// replay, projections, and downstream integrations.
type Type string

// Owner identifies whether an event type is core or system-owned.
//
// Core events are always managed by the common campaign/session/participant
// aggregate logic; system events are owned by pluggable modules.
type Owner string

const (
	// OwnerCore indicates a core domain event.
	OwnerCore Owner = "core"
	// OwnerSystem indicates a system-owned event.
	OwnerSystem Owner = "system"
)

// ActorType identifies the actor who initiated the event.
type ActorType string

const (
	// ActorTypeSystem indicates a system-originated event.
	ActorTypeSystem ActorType = "system"
	// ActorTypeParticipant indicates a participant-originated event.
	ActorTypeParticipant ActorType = "participant"
	// ActorTypeGM indicates a GM-originated event.
	ActorTypeGM ActorType = "gm"
)

// Event captures the canonical event envelope.
//
// The envelope is immutable metadata + business payload: storage appends
// integrity hashes and chain fields after validation, preserving replay order.
type Event struct {
	CampaignID     string
	Seq            uint64
	Hash           string
	PrevHash       string
	ChainHash      string
	Signature      string
	SignatureKeyID string
	Type           Type
	Timestamp      time.Time
	ActorType      ActorType
	ActorID        string
	SessionID      string
	RequestID      string
	InvocationID   string
	EntityType     string
	EntityID       string
	SystemID       string
	SystemVersion  string
	CorrelationID  string
	CausationID    string
	PayloadJSON    []byte
}

// PayloadValidator validates a payload JSON document.
type PayloadValidator func(json.RawMessage) error

// Definition registers metadata for an event type.
//
// Metadata declares how strict the registry should be around entity addressing and
// validation. This keeps projections honest about which aggregate subtree each event
// affects.
type Definition struct {
	Type            Type
	Owner           Owner
	Addressing      AddressingPolicy
	ValidatePayload PayloadValidator
}

// AddressingPolicy declares required entity-addressing fields for an event type.
type AddressingPolicy uint8

const (
	// AddressingPolicyNone does not require entity addressing fields.
	AddressingPolicyNone AddressingPolicy = iota
	// AddressingPolicyEntityType requires only entity_type.
	AddressingPolicyEntityType
	// AddressingPolicyEntityTarget requires both entity_type and entity_id.
	AddressingPolicyEntityTarget
)

// Registry stores event definitions and validates events for append.
//
// Validation happens at the edge of storage. The goal is to reject malformed
// events before persistence and before they can contaminate replay.
type Registry struct {
	definitions map[Type]Definition
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{definitions: make(map[Type]Definition)}
}

// Register adds a new event type definition to the registry.
func (r *Registry) Register(def Definition) error {
	if r == nil {
		return fmt.Errorf("registry is required")
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
	switch def.Addressing {
	case AddressingPolicyNone, AddressingPolicyEntityType, AddressingPolicyEntityTarget:
		// allowed
	default:
		return fmt.Errorf("event addressing policy is invalid")
	}
	if r.definitions == nil {
		r.definitions = make(map[Type]Definition)
	}
	if _, exists := r.definitions[def.Type]; exists {
		return fmt.Errorf("event type already registered: %s", def.Type)
	}
	r.definitions[def.Type] = def
	return nil
}

// ValidateForAppend validates and normalizes an event prior to storage append.
//
// It enforces ownership boundaries (core/system), canonical payload shape, and
// pre-append invariants (for example, hash/sequence fields are initially empty).
// This protects event history integrity from the first write.
func (r *Registry) ValidateForAppend(evt Event) (Event, error) {
	if r == nil {
		return Event{}, fmt.Errorf("registry is required")
	}
	if evt.Seq != 0 || strings.TrimSpace(evt.Hash) != "" || strings.TrimSpace(evt.PrevHash) != "" ||
		strings.TrimSpace(evt.ChainHash) != "" || strings.TrimSpace(evt.Signature) != "" ||
		strings.TrimSpace(evt.SignatureKeyID) != "" {
		return Event{}, ErrStorageFieldsSet
	}

	evt.CampaignID = strings.TrimSpace(evt.CampaignID)
	if evt.CampaignID == "" {
		return Event{}, ErrCampaignIDRequired
	}

	evt.Type = Type(strings.TrimSpace(string(evt.Type)))
	if evt.Type == "" {
		return Event{}, ErrTypeRequired
	}
	def, ok := r.definitions[evt.Type]
	if !ok {
		return Event{}, ErrTypeUnknown
	}

	evt.ActorType = ActorType(strings.TrimSpace(string(evt.ActorType)))
	if evt.ActorType == "" {
		evt.ActorType = ActorTypeSystem
	}
	switch evt.ActorType {
	case ActorTypeSystem, ActorTypeParticipant, ActorTypeGM:
		// allowed
	default:
		return Event{}, ErrActorTypeInvalid
	}
	evt.ActorID = strings.TrimSpace(evt.ActorID)
	if (evt.ActorType == ActorTypeParticipant || evt.ActorType == ActorTypeGM) && evt.ActorID == "" {
		return Event{}, ErrActorIDRequired
	}

	evt.SessionID = strings.TrimSpace(evt.SessionID)
	evt.RequestID = strings.TrimSpace(evt.RequestID)
	evt.InvocationID = strings.TrimSpace(evt.InvocationID)
	evt.EntityType = strings.TrimSpace(evt.EntityType)
	evt.EntityID = strings.TrimSpace(evt.EntityID)
	evt.SystemID = strings.TrimSpace(evt.SystemID)
	evt.SystemVersion = strings.TrimSpace(evt.SystemVersion)
	evt.CorrelationID = strings.TrimSpace(evt.CorrelationID)
	evt.CausationID = strings.TrimSpace(evt.CausationID)

	requireEntityType := false
	requireEntityID := false
	switch def.Addressing {
	case AddressingPolicyEntityType:
		requireEntityType = true
	case AddressingPolicyEntityTarget:
		requireEntityType = true
		requireEntityID = true
	}

	switch def.Owner {
	case OwnerSystem:
		if evt.SystemID == "" || evt.SystemVersion == "" {
			return Event{}, ErrSystemMetadataRequired
		}
		requireEntityType = true
		requireEntityID = true
	case OwnerCore:
		if evt.SystemID != "" || evt.SystemVersion != "" {
			return Event{}, ErrSystemMetadataForbidden
		}
	default:
		return Event{}, fmt.Errorf("event owner is invalid")
	}
	if requireEntityType && evt.EntityType == "" {
		return Event{}, ErrEntityTypeRequired
	}
	if requireEntityID && evt.EntityID == "" {
		return Event{}, ErrEntityIDRequired
	}
	if evt.EntityID != "" && evt.EntityType == "" {
		return Event{}, ErrEntityTypeRequired
	}

	if len(evt.PayloadJSON) == 0 {
		evt.PayloadJSON = []byte("{}")
	}
	if !json.Valid(evt.PayloadJSON) {
		return Event{}, ErrPayloadInvalid
	}

	canonical, err := coreencoding.CanonicalJSON(json.RawMessage(evt.PayloadJSON))
	if err != nil {
		return Event{}, fmt.Errorf("canonical payload json: %w", err)
	}
	evt.PayloadJSON = canonical

	if def.ValidatePayload != nil {
		if err := def.ValidatePayload(json.RawMessage(evt.PayloadJSON)); err != nil {
			return Event{}, fmt.Errorf("payload invalid: %w", err)
		}
	}

	return evt, nil
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
