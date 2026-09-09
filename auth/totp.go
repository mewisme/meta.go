package auth

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidTOTPSecret = errors.New("invalid TOTP secret")

func TOTP(secret string, now time.Time) (string, error) {
	normalized := strings.ToUpper(strings.NewReplacer(" ", "", "-", "").Replace(secret))
	if normalized == "" {
		return "", ErrInvalidTOTPSecret
	}
	padding := (8 - len(normalized)%8) % 8
	key, err := base32.StdEncoding.DecodeString(normalized + strings.Repeat("=", padding))
	if err != nil || len(key) == 0 {
		return "", ErrInvalidTOTPSecret
	}
	counter := uint64(now.Unix() / 30)
	message := make([]byte, 8)
	for index := 7; index >= 0; index-- {
		message[index] = byte(counter)
		counter >>= 8
	}
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(message)
	digest := mac.Sum(nil)
	offset := digest[len(digest)-1] & 0x0f
	code := (uint32(digest[offset])&0x7f)<<24 | uint32(digest[offset+1])<<16 | uint32(digest[offset+2])<<8 | uint32(digest[offset+3])
	return fmt.Sprintf("%06d", code%1_000_000), nil
}

func ResolveOTP(value string, now time.Time) (string, error) {
	compact := strings.ReplaceAll(strings.TrimSpace(value), " ", "")
	if len(compact) >= 6 && len(compact) <= 8 {
		if _, err := strconv.ParseUint(compact, 10, 32); err == nil {
			return compact, nil
		}
	}
	return TOTP(compact, now)
}
