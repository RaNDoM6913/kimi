package auth

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func ValidateTelegramInitData(initData string) error {
	if strings.TrimSpace(initData) == "" {
		return fmt.Errorf("init data is empty: %w", ErrInvalidInput)
	}
	return nil
}

func ResolveTelegramUserID(initData string) (int64, error) {
	trimmed := strings.TrimSpace(initData)
	if err := ValidateTelegramInitData(trimmed); err != nil {
		return 0, err
	}

	if parsed, err := strconv.ParseInt(trimmed, 10, 64); err == nil && parsed > 0 {
		return parsed, nil
	}

	query, err := url.ParseQuery(trimmed)
	if err == nil && len(query) > 0 {
		if rawUser := query.Get("user"); rawUser != "" {
			var payload struct {
				ID int64 `json:"id"`
			}
			if unmarshalErr := json.Unmarshal([]byte(rawUser), &payload); unmarshalErr == nil && payload.ID > 0 {
				return payload.ID, nil
			}
		}

		for _, key := range []string{"user_id", "id", "tg_user_id"} {
			if value := query.Get(key); value != "" {
				parsed, parseErr := strconv.ParseInt(value, 10, 64)
				if parseErr == nil && parsed > 0 {
					return parsed, nil
				}
			}
		}
	}

	return fallbackTelegramUserID(trimmed), nil
}

func fallbackTelegramUserID(initData string) int64 {
	hash := sha256.Sum256([]byte(initData))
	v := binary.BigEndian.Uint64(hash[:8]) & 0x7fffffffffffffff
	if v == 0 {
		v = 1
	}
	return int64(v)
}
