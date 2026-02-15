package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	errTelegramBotTokenMissing = errors.New("bot_token_missing")
	errTelegramPayloadMissing  = errors.New("payload_missing")
	errTelegramHashMissing     = errors.New("hash_missing")
	errTelegramHashMismatch    = errors.New("hash_mismatch")
	errTelegramAuthDateMissing = errors.New("auth_date_missing")
	errTelegramAuthDateInvalid = errors.New("auth_date_invalid")
	errTelegramAuthDateExpired = errors.New("auth_date_expired")
	errTelegramUserIDMissing   = errors.New("id_missing")
	errTelegramUserIDInvalid   = errors.New("id_invalid")
)

func verifyTelegramPayload(payload map[string]string, botToken string, now time.Time, maxAge time.Duration) (int64, *string, error) {
	if strings.TrimSpace(botToken) == "" {
		return 0, nil, errTelegramBotTokenMissing
	}
	if len(payload) == 0 {
		return 0, nil, errTelegramPayloadMissing
	}

	hashValue := strings.TrimSpace(payload["hash"])
	if hashValue == "" {
		return 0, nil, errTelegramHashMissing
	}

	dataCheck := buildTelegramDataCheckString(payload)
	if !isTelegramHashValid(botToken, dataCheck, hashValue) {
		return 0, nil, errTelegramHashMismatch
	}

	authDateRaw := strings.TrimSpace(payload["auth_date"])
	if authDateRaw == "" {
		return 0, nil, errTelegramAuthDateMissing
	}
	authDateUnix, err := strconv.ParseInt(authDateRaw, 10, 64)
	if err != nil {
		return 0, nil, errTelegramAuthDateInvalid
	}
	authTime := time.Unix(authDateUnix, 0).UTC()
	if now.Sub(authTime) > maxAge || authTime.Sub(now) > time.Minute {
		return 0, nil, errTelegramAuthDateExpired
	}

	idRaw := strings.TrimSpace(payload["id"])
	if idRaw == "" {
		return 0, nil, errTelegramUserIDMissing
	}
	telegramUserID, err := strconv.ParseInt(idRaw, 10, 64)
	if err != nil || telegramUserID <= 0 {
		return 0, nil, errTelegramUserIDInvalid
	}

	username := strings.TrimSpace(payload["username"])
	if username == "" {
		return telegramUserID, nil, nil
	}
	return telegramUserID, &username, nil
}

func buildTelegramDataCheckString(payload map[string]string) string {
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
	return strings.Join(parts, "\n")
}

func isTelegramHashValid(botToken string, dataCheck string, hashValue string) bool {
	secret := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secret[:])
	mac.Write([]byte(dataCheck))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(strings.ToLower(hashValue)))
}
