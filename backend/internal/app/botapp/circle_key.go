package botapp

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path"
	"strings"
	"time"
)

func buildCircleObjectKey(userID int64, fileName string) (string, error) {
	rnd := make([]byte, 8)
	if _, err := rand.Read(rnd); err != nil {
		return "", err
	}

	ext := strings.ToLower(path.Ext(strings.TrimSpace(fileName)))
	if ext == "" {
		ext = ".mp4"
	}

	stamp := time.Now().UTC().Format("20060102T150405")
	return fmt.Sprintf("users/%d/circle/%s_%s%s", userID, stamp, hex.EncodeToString(rnd), ext), nil
}
