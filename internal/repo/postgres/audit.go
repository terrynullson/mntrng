package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
)

// InsertAuditLogTx writes one audit log row in the given transaction.
// Used by all API repos for approve/reject, role/status change, and entity mutations.
func InsertAuditLogTx(
	ctx context.Context,
	tx *sql.Tx,
	companyID int64,
	actorType string,
	actorID string,
	entityType string,
	entityID int64,
	action string,
	payload map[string]interface{},
) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO audit_log (company_id, actor_type, actor_id, entity_type, entity_id, action, payload)
         VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)`,
		companyID,
		actorType,
		actorID,
		entityType,
		entityID,
		action,
		string(payloadJSON),
	)
	return err
}
