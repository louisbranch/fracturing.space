---
title: "Minimal game system skeleton"
parent: "Guides"
nav_order: 10
status: canonical
owner: engineering
last_reviewed: "2026-03-27"
---

# Minimal game system skeleton

Smallest possible game system: one command, one event, one state field. For a
production reference, see `domain/systems/daggerheart/`.

## Identity, types, and state

```go
package mysystem

import ("github.com/.../domain/command"; "github.com/.../domain/event")

const (
    SystemID      = "mysystem"
    SystemVersion = "1.0.0"
    CommandTypeScoreSet   command.Type = "mysystem.score.set"
    EventTypeScoreChanged event.Type   = "mysystem.score.changed"
)

type SnapshotState struct {
    CampaignID string
    Score      int
}

type scoreSetPayload struct{ Score int `json:"score"` }
```

## State assertion helper

Every system needs a function to recover typed state from `any`:

```go
func assertState(raw any) (*SnapshotState, error) {
    if raw == nil { return &SnapshotState{}, nil }
    s, ok := raw.(*SnapshotState)
    if !ok { return nil, fmt.Errorf("unexpected state type %T", raw) }
    return s, nil
}
```

## Decider

Uses `TypedDecider[S]` so the decide function receives `*SnapshotState`:

```go
func newDecider() module.Decider {
    return module.TypedDecider[*SnapshotState]{
        Assert: assertState,
        Fn: func(s *SnapshotState, cmd command.Command, now func() time.Time) command.Decision {
            var p scoreSetPayload
            if err := json.Unmarshal(cmd.PayloadJSON, &p); err != nil {
                return command.Reject(command.Rejection{
                    Code: command.RejectionCodePayloadDecodeFailed, Message: err.Error()})
            }
            payloadJSON, _ := json.Marshal(p)
            return command.Accept(event.Event{Type: EventTypeScoreChanged, PayloadJSON: payloadJSON})
        },
    }
}
```

## Folder

Uses `FoldRouter[S]` with `HandleFold` for automatic payload unmarshaling:

```go
func newFolder() module.Folder {
    r := module.NewFoldRouter[*SnapshotState](assertState)
    module.HandleFold(r, EventTypeScoreChanged,
        func(s *SnapshotState, p scoreSetPayload) error {
            s.Score = p.Score
            return nil
        })
    return r
}
```

## State factory

```go
type stateFactory struct{}

func (f stateFactory) NewSnapshotState(id ids.CampaignID) (any, error) {
    return &SnapshotState{CampaignID: string(id)}, nil
}
func (f stateFactory) NewCharacterState(_ ids.CampaignID, _ ids.CharacterID, _ string) (any, error) {
    return nil, nil // no character state
}
```

## Module (wires everything together)

```go
type Module struct {
    decider domainmodule.Decider
    folder  domainmodule.Folder
    factory domainmodule.StateFactory
}

func NewModule() *Module {
    return &Module{decider: newDecider(), folder: newFolder(), factory: stateFactory{}}
}

func (m *Module) ID() string      { return SystemID }
func (m *Module) Version() string { return SystemVersion }
func (m *Module) RegisterCommands(r *command.Registry) error {
    return r.Register(command.Definition{Type: CommandTypeScoreSet, Owner: command.OwnerSystem})
}
func (m *Module) RegisterEvents(r *event.Registry) error {
    return r.Register(event.Definition{
        Type: EventTypeScoreChanged, Owner: event.OwnerSystem,
        Intent: event.IntentProjectionAndReplay})
}
func (m *Module) EmittableEventTypes() []event.Type { return []event.Type{EventTypeScoreChanged} }
func (m *Module) Decider() domainmodule.Decider           { return m.decider }
func (m *Module) Folder() domainmodule.Folder             { return m.folder }
func (m *Module) StateFactory() domainmodule.StateFactory { return m.factory }

var _ domainmodule.Module = (*Module)(nil)
```

## Manifest entry

Register in `domain/systems/manifest/manifest.go`:

```go
{
    ID:          mysystem.SystemID,
    Version:     mysystem.SystemVersion,
    BuildModule: func() domainsystem.Module { return mysystem.NewModule() },
    // Add BuildMetadataSystem / BuildAdapter when the system needs
    // API-layer metadata or projection-side event handling.
},
```

`BuildRegistries` reads the manifest, registers commands/events, and runs
startup validation (decider command coverage, fold event coverage, emittable
type registration).
