-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS execution_request_idempotency (
    scope_key TEXT NOT NULL,
    target TEXT NOT NULL,
    request_key TEXT NOT NULL,
    body_digest TEXT NOT NULL,
    execution_id TEXT NOT NULL,
    run_id TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    PRIMARY KEY (scope_key, request_key),
    UNIQUE (execution_id)
);
CREATE INDEX IF NOT EXISTS idx_execution_request_idempotency_execution
    ON execution_request_idempotency(execution_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_execution_request_idempotency_execution;
DROP TABLE IF EXISTS execution_request_idempotency;
-- +goose StatementEnd
