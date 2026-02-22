-- name: AppendAuditEvent :exec
INSERT INTO audit_events (
  timestamp,
  event_name,
  severity,
  campaign_id,
  session_id,
  actor_type,
  actor_id,
  request_id,
  invocation_id,
  trace_id,
  span_id,
  attributes_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
