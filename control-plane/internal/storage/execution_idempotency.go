package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Agent-Field/agentfield/control-plane/pkg/types"
)

// AsyncExecutionIdempotencyResult describes the durable request binding.
type AsyncExecutionIdempotencyResult struct {
	Execution *types.Execution
	Created   bool
	Conflict  bool
}

// AsyncExecutionIdempotencyStore is the optional durable capability used by
// the async execute handler. Stores without it must reject opt-in idempotency
// rather than silently falling back to process-local deduplication.
type AsyncExecutionIdempotencyStore interface {
	LookupAsyncExecutionIdempotency(ctx context.Context, scopeKey, target, requestKey, bodyDigest string) (*AsyncExecutionIdempotencyResult, error)
	CreateAsyncExecutionIdempotent(ctx context.Context, scopeKey, target, requestKey, bodyDigest string, execution *types.Execution) (*AsyncExecutionIdempotencyResult, error)
}

// LookupAsyncExecutionIdempotency checks an existing durable binding without
// reserving queue capacity. A matching digest returns the canonical execution;
// a different digest returns Conflict.
func (ls *LocalStorage) LookupAsyncExecutionIdempotency(ctx context.Context, scopeKey, target, requestKey, bodyDigest string) (*AsyncExecutionIdempotencyResult, error) {
	row := ls.requireSQLDB().QueryRowContext(ctx, `
		SELECT target, body_digest, execution_id
		FROM execution_request_idempotency
		WHERE scope_key = ? AND request_key = ?`, scopeKey, requestKey)

	var storedTarget, storedDigest, executionID string
	if err := row.Scan(&storedTarget, &storedDigest, &executionID); err != nil {
		if err == sql.ErrNoRows {
			return &AsyncExecutionIdempotencyResult{}, nil
		}
		return nil, fmt.Errorf("lookup async execution idempotency: %w", err)
	}
	if storedTarget != target || storedDigest != bodyDigest {
		return &AsyncExecutionIdempotencyResult{Conflict: true}, nil
	}

	execution, err := ls.GetExecutionRecord(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("load idempotent execution %s: %w", executionID, err)
	}
	if execution == nil {
		return nil, fmt.Errorf("idempotency binding %s references missing execution %s", requestKey, executionID)
	}
	return &AsyncExecutionIdempotencyResult{Execution: execution}, nil
}

// CreateAsyncExecutionIdempotent atomically binds a request key and creates its
// execution row. The unique key is the database fence for concurrent requests;
// no second execution row is inserted for a winning binding.
func (ls *LocalStorage) CreateAsyncExecutionIdempotent(ctx context.Context, scopeKey, target, requestKey, bodyDigest string, execution *types.Execution) (*AsyncExecutionIdempotencyResult, error) {
	if execution == nil {
		return nil, fmt.Errorf("nil execution payload")
	}
	db := ls.requireSQLDB()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin async idempotency transaction: %w", err)
	}
	defer rollbackTx(tx, "CreateAsyncExecutionIdempotent")

	now := time.Now().UTC()
	insertResult, err := tx.ExecContext(ctx, `
		INSERT INTO execution_request_idempotency
			(scope_key, target, request_key, body_digest, execution_id, run_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (scope_key, request_key) DO NOTHING`,
		scopeKey, target, requestKey, bodyDigest, execution.ExecutionID, execution.RunID, now, now)
	if err != nil {
		return nil, fmt.Errorf("insert async idempotency binding: %w", err)
	}

	affected, err := insertResult.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("check async idempotency binding insert: %w", err)
	}
	if affected == 0 {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit existing async idempotency lookup: %w", err)
		}
		existing, err := ls.LookupAsyncExecutionIdempotency(ctx, scopeKey, target, requestKey, bodyDigest)
		if err != nil {
			return nil, err
		}
		return existing, nil
	}

	execution.CreatedAt = now
	execution.UpdatedAt = now
	if execution.StartedAt.IsZero() {
		execution.StartedAt = now
	}
	if err := insertExecutionRecordTx(ctx, tx, execution); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit async idempotent execution: %w", err)
	}
	return &AsyncExecutionIdempotencyResult{Execution: execution, Created: true}, nil
}

func insertExecutionRecordTx(ctx context.Context, tx *sqlTx, exec *types.Execution) error {
	var notesJSON []byte
	var err error
	if len(exec.Notes) > 0 {
		notesJSON, err = json.Marshal(exec.Notes)
		if err != nil {
			return fmt.Errorf("marshal notes: %w", err)
		}
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO executions (
			execution_id, run_id, parent_execution_id,
			agent_node_id, instance_id, reasoner_id, node_id,
			status, status_reason, input_payload, result_payload, error_message,
			input_uri, result_uri, session_id, actor_id,
			started_at, completed_at, duration_ms, notes, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		exec.ExecutionID, exec.RunID, exec.ParentExecutionID,
		exec.AgentNodeID, exec.InstanceID, exec.ReasonerID, exec.NodeID,
		exec.Status, exec.StatusReason, bytesOrNil(exec.InputPayload), bytesOrNil(exec.ResultPayload), exec.ErrorMessage,
		exec.InputURI, exec.ResultURI, exec.SessionID, exec.ActorID,
		exec.StartedAt, exec.CompletedAt, exec.DurationMS, notesJSON, exec.CreatedAt, exec.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert idempotent execution: %w", err)
	}
	return nil
}
