-- +migrate Up

CREATE TABLE ai_access_requests (
    id TEXT PRIMARY KEY,
    requester_user_id TEXT NOT NULL,
    owner_user_id TEXT NOT NULL,
    agent_id TEXT NOT NULL,
    scope TEXT NOT NULL,
    request_note TEXT NOT NULL,
    status TEXT NOT NULL,
    reviewer_user_id TEXT NOT NULL,
    review_note TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    reviewed_at INTEGER
);

CREATE INDEX ai_access_requests_requester_id_idx ON ai_access_requests(requester_user_id, id);
CREATE INDEX ai_access_requests_owner_id_idx ON ai_access_requests(owner_user_id, id);

-- +migrate Down
DROP TABLE IF EXISTS ai_access_requests;
