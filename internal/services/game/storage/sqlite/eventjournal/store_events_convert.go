package eventjournal

import (
	"errors"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

func isConstraintError(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	code := sqliteErr.Code()
	return code == sqlite3.SQLITE_CONSTRAINT || code == sqlite3.SQLITE_CONSTRAINT_UNIQUE || code == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY
}

// Domain conversion helpers for events

type eventRowData struct {
	CampaignID     string
	Seq            int64
	EventHash      string
	PrevEventHash  string
	ChainHash      string
	SignatureKeyID string
	EventSignature string
	Timestamp      int64
	EventType      string
	SessionID      string
	SceneID        string
	RequestID      string
	InvocationID   string
	ActorType      string
	ActorID        string
	EntityType     string
	EntityID       string
	SystemID       string
	SystemVersion  string
	CorrelationID  string
	CausationID    string
	PayloadJSON    []byte
}

func eventRowDataToDomain(row eventRowData) (event.Event, error) {
	return event.Event{
		CampaignID:     ids.CampaignID(row.CampaignID),
		Seq:            uint64(row.Seq),
		Hash:           row.EventHash,
		PrevHash:       row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		Signature:      row.EventSignature,
		Timestamp:      sqliteutil.FromMillis(row.Timestamp),
		Type:           event.Type(row.EventType),
		SessionID:      ids.SessionID(row.SessionID),
		SceneID:        ids.SceneID(row.SceneID),
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      event.ActorType(row.ActorType),
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		CorrelationID:  row.CorrelationID,
		CausationID:    row.CausationID,
		PayloadJSON:    row.PayloadJSON,
	}, nil
}

func eventRowDataFromEvent(row db.Event) eventRowData {
	return eventRowData{
		CampaignID:     row.CampaignID,
		Seq:            row.Seq,
		EventHash:      row.EventHash,
		PrevEventHash:  row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		EventSignature: row.EventSignature,
		Timestamp:      row.Timestamp,
		EventType:      row.EventType,
		SessionID:      row.SessionID,
		SceneID:        row.SceneID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      row.ActorType,
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		CorrelationID:  row.CorrelationID,
		CausationID:    row.CausationID,
		PayloadJSON:    row.PayloadJson,
	}
}

func eventRowDataFromGetEventByHashRow(row db.GetEventByHashRow) eventRowData {
	return eventRowData{
		CampaignID:     row.CampaignID,
		Seq:            row.Seq,
		EventHash:      row.EventHash,
		PrevEventHash:  row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		EventSignature: row.EventSignature,
		Timestamp:      row.Timestamp,
		EventType:      row.EventType,
		SessionID:      row.SessionID,
		SceneID:        row.SceneID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      row.ActorType,
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		CorrelationID:  row.CorrelationID,
		CausationID:    row.CausationID,
		PayloadJSON:    row.PayloadJson,
	}
}

func eventRowDataFromGetEventBySeqRow(row db.GetEventBySeqRow) eventRowData {
	return eventRowData{
		CampaignID:     row.CampaignID,
		Seq:            row.Seq,
		EventHash:      row.EventHash,
		PrevEventHash:  row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		EventSignature: row.EventSignature,
		Timestamp:      row.Timestamp,
		EventType:      row.EventType,
		SessionID:      row.SessionID,
		SceneID:        row.SceneID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      row.ActorType,
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		CorrelationID:  row.CorrelationID,
		CausationID:    row.CausationID,
		PayloadJSON:    row.PayloadJson,
	}
}

func eventRowDataFromListEventsRow(row db.ListEventsRow) eventRowData {
	return eventRowData{
		CampaignID:     row.CampaignID,
		Seq:            row.Seq,
		EventHash:      row.EventHash,
		PrevEventHash:  row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		EventSignature: row.EventSignature,
		Timestamp:      row.Timestamp,
		EventType:      row.EventType,
		SessionID:      row.SessionID,
		SceneID:        row.SceneID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      row.ActorType,
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		CorrelationID:  row.CorrelationID,
		CausationID:    row.CausationID,
		PayloadJSON:    row.PayloadJson,
	}
}

func eventRowDataFromListEventsBySessionRow(row db.ListEventsBySessionRow) eventRowData {
	return eventRowData{
		CampaignID:     row.CampaignID,
		Seq:            row.Seq,
		EventHash:      row.EventHash,
		PrevEventHash:  row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		EventSignature: row.EventSignature,
		Timestamp:      row.Timestamp,
		EventType:      row.EventType,
		SessionID:      row.SessionID,
		SceneID:        row.SceneID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      row.ActorType,
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		CorrelationID:  row.CorrelationID,
		CausationID:    row.CausationID,
		PayloadJSON:    row.PayloadJson,
	}
}

func eventRowsToDomain(rows []db.ListEventsRow) ([]event.Event, error) {
	events := make([]event.Event, 0, len(rows))
	for _, row := range rows {
		evt, err := eventRowDataToDomain(eventRowDataFromListEventsRow(row))
		if err != nil {
			return nil, err
		}
		events = append(events, evt)
	}
	return events, nil
}

func eventRowsBySessionToDomain(rows []db.ListEventsBySessionRow) ([]event.Event, error) {
	events := make([]event.Event, 0, len(rows))
	for _, row := range rows {
		evt, err := eventRowDataToDomain(eventRowDataFromListEventsBySessionRow(row))
		if err != nil {
			return nil, err
		}
		events = append(events, evt)
	}
	return events, nil
}
