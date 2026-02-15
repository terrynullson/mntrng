package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestVerifyTelegramPayload(t *testing.T) {
	botToken := "123456:telegram_token"
	now := time.Now().UTC()
	authDate := now.Unix()

	payload := map[string]string{
		"id":         "777",
		"first_name": "Alice",
		"username":   "alice_dev",
		"auth_date":  fmt.Sprintf("%d", authDate),
	}
	payload["hash"] = signTelegramPayload(botToken, payload)

	userID, username, err := verifyTelegramPayload(payload, botToken, now, 10*time.Minute)
	if err != nil {
		t.Fatalf("expected valid payload, got error: %v", err)
	}
	if userID != 777 {
		t.Fatalf("expected user id 777, got %d", userID)
	}
	if username == nil || *username != "alice_dev" {
		t.Fatalf("expected username alice_dev, got %+v", username)
	}
}

func TestVerifyTelegramPayloadInvalidSignature(t *testing.T) {
	botToken := "123456:telegram_token"
	now := time.Now().UTC()

	payload := map[string]string{
		"id":         "777",
		"first_name": "Alice",
		"username":   "alice_dev",
		"auth_date":  fmt.Sprintf("%d", now.Unix()),
		"hash":       "invalid_hash",
	}

	_, _, err := verifyTelegramPayload(payload, botToken, now, 10*time.Minute)
	if err == nil {
		t.Fatal("expected hash mismatch error")
	}
	if err != errTelegramHashMismatch {
		t.Fatalf("expected errTelegramHashMismatch, got %v", err)
	}
}

func signTelegramPayload(botToken string, payload map[string]string) string {
	keys := make([]string, 0, len(payload))
	for key, value := range payload {
		if key == "hash" {
			continue
		}
		if strings.TrimSpace(value) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, payload[key]))
	}
	dataCheck := strings.Join(parts, "\n")

	secret := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secret[:])
	mac.Write([]byte(dataCheck))
	return hex.EncodeToString(mac.Sum(nil))
}
