-- Telemetry events are stored separately from game events.
CREATE TABLE IF NOT EXISTS telemetry_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  timestamp TEXT NOT NULL,
  event_name TEXT NOT NULL,
  severity TEXT NOT NULL,
  campaign_id TEXT,
  session_id TEXT,
  actor_type TEXT,
  actor_id TEXT,
  request_id TEXT,
  invocation_id TEXT,
  trace_id TEXT,
  span_id TEXT,
  attributes_json BLOB
);

CREATE INDEX IF NOT EXISTS idx_telemetry_events_campaign_id ON telemetry_events (campaign_id);
CREATE INDEX IF NOT EXISTS idx_telemetry_events_timestamp ON telemetry_events (timestamp);
