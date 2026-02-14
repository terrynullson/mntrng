package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCompanyAuditHistoryPersistsAfterDeletion(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping db: %v", err)
	}

	schemaName := fmt.Sprintf("audit_history_%d", time.Now().UTC().UnixNano())
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`CREATE SCHEMA "%s"`, schemaName)); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	defer db.ExecContext(context.Background(), fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, schemaName))

	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("acquire conn: %v", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, fmt.Sprintf(`SET search_path TO "%s"`, schemaName)); err != nil {
		t.Fatalf("set search_path: %v", err)
	}

	applyMigrationFromFile(t, ctx, conn, filepath.Join("..", "..", "migrations", "0001_baseline_schema.up.sql"))
	applyMigrationFromFile(t, ctx, conn, filepath.Join("..", "..", "migrations", "0002_telegram_delivery_settings.up.sql"))
	applyMigrationFromFile(t, ctx, conn, filepath.Join("..", "..", "migrations", "0003_preserve_company_audit_history.up.sql"))

	var companyID int64
	if err := conn.QueryRowContext(
		ctx,
		`INSERT INTO companies (name) VALUES ($1) RETURNING id`,
		"Acme Retention Audit",
	).Scan(&companyID); err != nil {
		t.Fatalf("insert company: %v", err)
	}

	actions := []string{"create", "update", "delete"}
	for _, action := range actions {
		if _, err := conn.ExecContext(
			ctx,
			`INSERT INTO audit_log (company_id, actor_type, actor_id, entity_type, entity_id, action, payload)
             VALUES ($1, 'api', 'system', 'company', $1, $2, '{"test":true}'::jsonb)`,
			companyID,
			action,
		); err != nil {
			t.Fatalf("insert audit log action=%s: %v", action, err)
		}
	}

	if _, err := conn.ExecContext(ctx, `DELETE FROM companies WHERE id = $1`, companyID); err != nil {
		t.Fatalf("delete company: %v", err)
	}

	var companyCount int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(1) FROM companies WHERE id = $1`, companyID).Scan(&companyCount); err != nil {
		t.Fatalf("count companies: %v", err)
	}
	if companyCount != 0 {
		t.Fatalf("expected company to be deleted, got count=%d", companyCount)
	}

	var auditCount int
	if err := conn.QueryRowContext(
		ctx,
		`SELECT COUNT(1)
         FROM audit_log
         WHERE company_id = $1
           AND entity_type = 'company'
           AND action IN ('create', 'update', 'delete')`,
		companyID,
	).Scan(&auditCount); err != nil {
		t.Fatalf("count audit logs: %v", err)
	}
	if auditCount != 3 {
		t.Fatalf("expected 3 company audit entries to persist, got %d", auditCount)
	}
}

func applyMigrationFromFile(t *testing.T, ctx context.Context, conn *sql.Conn, path string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration file %s: %v", path, err)
	}

	if _, err := conn.ExecContext(ctx, string(content)); err != nil {
		t.Fatalf("apply migration %s: %v", path, err)
	}
}
