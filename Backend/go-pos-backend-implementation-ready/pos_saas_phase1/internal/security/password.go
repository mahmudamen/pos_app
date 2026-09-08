package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 2
	argonKeyLen  = 32
	argonSaltLen = 16
)

func HashPassword(password string) (string, error) {
	if len(password) < 12 {
		return "", errors.New("password must be at least 12 characters")
	}
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf(
		"argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func VerifyPassword(password, encoded string) bool {
	var m, t, p uint32
	var saltB64, hashB64 string
	n, err := fmt.Sscanf(encoded, "argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		&m, &t, &p, &saltB64, &hashB64)
	if err != nil || n != 5 {
		return false
	}
	salt, err1 := base64.RawStdEncoding.DecodeString(saltB64)
	expected, err2 := base64.RawStdEncoding.DecodeString(hashB64)
	if err1 != nil || err2 != nil {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, t, m, uint8(p), uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}
