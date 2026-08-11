// Package algorithm contains cryptographic helpers used by GooseForum.
package algorithm

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const (
	saltLength     = 32
	hashIterations = 10000
	hashKeyLen     = 32
)

// MakePassword hashes password and returns a storable hash:salt string.
func MakePassword(password string) (string, error) {
	hash, salt, err := EncryptPassword(password)
	return hash + ":" + salt, err
}

// VerifyEncryptPassword verifies inputPassword against a stored hash:salt value.
func VerifyEncryptPassword(secretPassword, inputPassword string) error {
	passwordStore := strings.Split(secretPassword, ":")
	if len(passwordStore) != 2 {
		return errors.New("no pass")
	}
	return VerifyPassword(passwordStore[0], passwordStore[1], inputPassword)
}

// IsWellFormedPasswordHash reports whether secretPassword is a hash:salt
// value that VerifyEncryptPassword will fully process: exactly two segments,
// both valid base64, decoding to the expected hash/salt lengths. Malformed
// values fail fast inside VerifyEncryptPassword without running PBKDF2, so
// callers that equalize verification timing must detect them up front.
func IsWellFormedPasswordHash(secretPassword string) bool {
	passwordStore := strings.Split(secretPassword, ":")
	if len(passwordStore) != 2 {
		return false
	}
	hash, err := base64.StdEncoding.DecodeString(passwordStore[0])
	if err != nil || len(hash) != hashKeyLen {
		return false
	}
	salt, err := base64.StdEncoding.DecodeString(passwordStore[1])
	if err != nil || len(salt) != saltLength {
		return false
	}
	return true
}

// EncryptPassword hashes password with a random salt.
func EncryptPassword(password string) (string, string, error) {
	salt := make([]byte, saltLength)
	_, err := rand.Read(salt)
	if err != nil {
		return "", "", err
	}

	hash := pbkdf2SHA256([]byte(password), salt, hashIterations, hashKeyLen)
	encodedHash := base64.StdEncoding.EncodeToString(hash)
	encodedSalt := base64.StdEncoding.EncodeToString(salt)

	return encodedHash, encodedSalt, nil
}

// VerifyPassword verifies inputPassword against encodedHash and encodedSalt.
func VerifyPassword(encodedHash, encodedSalt, inputPassword string) error {
	hash, err := base64.StdEncoding.DecodeString(encodedHash)
	if err != nil {
		return errors.New("invalid password hash")
	}
	salt, err := base64.StdEncoding.DecodeString(encodedSalt)
	if err != nil {
		return errors.New("invalid password salt")
	}

	inputHash := pbkdf2SHA256([]byte(inputPassword), salt, hashIterations, hashKeyLen)

	if !equalHashes(hash, inputHash) {
		return errors.New("incorrect password")
	}

	return nil
}

func equalHashes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}

func pbkdf2SHA256(password []byte, salt []byte, iterations int, keyLen int) []byte {
	hashFunc := sha256.New
	dk := pbkdf2.Key(password, salt, iterations, keyLen, hashFunc)
	return dk
}
