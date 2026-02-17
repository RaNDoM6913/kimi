package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidTelegramData = errors.New("invalid telegram init data")
)

type TelegramIdentity struct {
	UserID   int64
	Username string
}

func ParseAndValidateInitData(initData, botToken string, maxAge time.Duration, devMode bool) (TelegramIdentity, error) {
	trimmed := strings.TrimSpace(initData)
	if trimmed == "" {
		return TelegramIdentity{}, ErrInvalidTelegramData
	}

	if devMode {
		if id, err := strconv.ParseInt(trimmed, 10, 64); err == nil && id > 0 {
			return TelegramIdentity{UserID: id}, nil
		}
	}

	query, err := url.ParseQuery(trimmed)
	if err != nil {
		return TelegramIdentity{}, fmt.Errorf("parse init data: %w", err)
	}

	hash := strings.TrimSpace(query.Get("hash"))
	if hash == "" {
		return TelegramIdentity{}, ErrInvalidTelegramData
	}
	query.Del("hash")

	dataCheckString := buildDataCheckString(query)
	if !verifyTelegramHash(dataCheckString, hash, botToken) {
		return TelegramIdentity{}, ErrInvalidTelegramData
	}

	authDateRaw := strings.TrimSpace(query.Get("auth_date"))
	authDate, err := strconv.ParseInt(authDateRaw, 10, 64)
	if err != nil || authDate <= 0 {
		return TelegramIdentity{}, ErrInvalidTelegramData
	}
	now := time.Now().UTC()
	authTime := time.Unix(authDate, 0).UTC()
	if authTime.After(now.Add(2 * time.Minute)) {
		return TelegramIdentity{}, ErrInvalidTelegramData
	}
	if maxAge > 0 && now.Sub(authTime) > maxAge {
		return TelegramIdentity{}, ErrInvalidTelegramData
	}

	identity, err := parseTelegramIdentity(query)
	if err != nil {
		return TelegramIdentity{}, err
	}
	return identity, nil
}

func parseTelegramIdentity(query url.Values) (TelegramIdentity, error) {
	if rawUser := strings.TrimSpace(query.Get("user")); rawUser != "" {
		var payload struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
		}
		if err := json.Unmarshal([]byte(rawUser), &payload); err == nil && payload.ID > 0 {
			return TelegramIdentity{UserID: payload.ID, Username: payload.Username}, nil
		}
	}

	username := strings.TrimSpace(query.Get("username"))
	for _, key := range []string{"id", "user_id", "tg_user_id"} {
		v := strings.TrimSpace(query.Get(key))
		if v == "" {
			continue
		}
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err == nil && parsed > 0 {
			return TelegramIdentity{UserID: parsed, Username: username}, nil
		}
	}

	return TelegramIdentity{}, ErrInvalidTelegramData
}

func buildDataCheckString(query url.Values) string {
	pairs := make([]string, 0, len(query))
	for k, v := range query {
		if len(v) == 0 {
			pairs = append(pairs, k+"=")
			continue
		}
		pairs = append(pairs, k+"="+v[0])
	}
	sort.Strings(pairs)
	return strings.Join(pairs, "\n")
}

func verifyTelegramHash(dataCheckString, incomingHash, botToken string) bool {
	hashLower := strings.ToLower(strings.TrimSpace(incomingHash))
	if hashLower == "" {
		return false
	}

	webAppSecret := hmacSHA256([]byte("WebAppData"), []byte(botToken))
	webAppHash := hex.EncodeToString(hmacSHA256(webAppSecret, []byte(dataCheckString)))
	if hmac.Equal([]byte(hashLower), []byte(webAppHash)) {
		return true
	}

	loginSecret := sha256.Sum256([]byte(botToken))
	loginHash := hex.EncodeToString(hmacSHA256(loginSecret[:], []byte(dataCheckString)))
	return hmac.Equal([]byte(hashLower), []byte(loginHash))
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}
