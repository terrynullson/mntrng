// Package main seeds a test company and a test user for screenshot automation.
// Only for local/dev. Credentials: test_screenshot_admin / TestScreenshot1
package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	"github.com/example/hls-monitoring-platform/internal/config"
)

const (
	testCompanyName = "Screenshot Test Company"
	testLogin       = "test_screenshot_admin"
	testEmail       = "test_screenshot@localhost"
	testPassword    = "TestScreenshot1"
	testRole        = "company_admin"
)

func main() {
	databaseURL := config.GetString("DATABASE_URL", os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	var companyID int64
	err = db.QueryRowContext(ctx,
		`INSERT INTO companies (name) VALUES ($1) ON CONFLICT (name) DO NOTHING RETURNING id`,
		testCompanyName,
	).Scan(&companyID)
	if err != nil {
		if err == sql.ErrNoRows {
			err = db.QueryRowContext(ctx, `SELECT id FROM companies WHERE name = $1`, testCompanyName).Scan(&companyID)
			if err != nil {
				log.Fatalf("get company id: %v", err)
			}
			log.Printf("company already exists: id=%d", companyID)
		} else {
			log.Fatalf("insert company: %v", err)
		}
	} else {
		log.Printf("created company: id=%d name=%s", companyID, testCompanyName)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	_, err = db.ExecContext(ctx,
		`INSERT INTO users (company_id, email, login, password_hash, role, status)
		 VALUES ($1, $2, $3, $4, $5, 'active')`,
		companyID, testEmail, testLogin, string(hash), testRole,
	)
	if err != nil {
		if isUniqueViolation(err) {
			log.Printf("user already exists: login=%s (seed idempotent)", testLogin)
			return
		}
		log.Fatalf("insert user: %v", err)
	}

	log.Printf("seed ok: login=%s role=%s company_id=%d (use for screenshot automation only)", testLogin, testRole, companyID)
}

func isUniqueViolation(err error) bool {
	var pgErr *pq.Error
	return errors.As(err, &pgErr) && string(pgErr.Code) == "23505"
}
