// Package argon provides password hashing and verification using Argon2id.
package argon

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	defaultMemory      uint32 = 64 * 1024
	defaultIterations  uint32 = 3
	defaultParallelism uint8  = 2
	defaultSaltLength  uint32 = 16
	defaultKeyLength   uint32 = 32

	maxMemory      uint32 = 256 * 1024
	maxIterations  uint32 = 10
	maxParallelism uint8  = 16
	maxSaltLength  uint32 = 64
	maxKeyLength   uint32 = 64
)

var ErrInvalidHash = errors.New("invalid argon2id hash")

type Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

func DefaultParams() Params {
	return Params{
		Memory:      defaultMemory,
		Iterations:  defaultIterations,
		Parallelism: defaultParallelism,
		SaltLength:  defaultSaltLength,
		KeyLength:   defaultKeyLength,
	}
}

// Hash returns a PHC-formatted Argon2id password hash using secure defaults.
func Hash(password string) (string, error) {
	return hash(password, DefaultParams(), rand.Reader)
}

func hash(password string, params Params, randomness io.Reader) (string, error) {
	if err := validateParams(params); err != nil {
		return "", err
	}

	salt := make([]byte, params.SaltLength)
	if _, err := io.ReadFull(randomness, salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}

	key := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		params.Memory,
		params.Iterations,
		params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// Verify compares a password with a PHC-formatted Argon2id hash in constant
// time. Malformed or unsafe hashes return ErrInvalidHash.
func Verify(password, encodedHash string) (bool, error) {
	params, salt, expectedKey, err := parse(encodedHash)
	if err != nil {
		return false, err
	}

	actualKey := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		uint32(len(expectedKey)),
	)

	return subtle.ConstantTimeCompare(actualKey, expectedKey) == 1, nil
}

func parse(encodedHash string) (Params, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return Params{}, nil, nil, ErrInvalidHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return Params{}, nil, nil, ErrInvalidHash
	}

	var params Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.Memory, &params.Iterations, &params.Parallelism); err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	key, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}

	params.SaltLength = uint32(len(salt))
	params.KeyLength = uint32(len(key))
	if err := validateParams(params); err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}

	return params, salt, key, nil
}

func validateParams(params Params) error {
	if params.Memory < 8*uint32(params.Parallelism) || params.Memory > maxMemory ||
		params.Iterations == 0 || params.Iterations > maxIterations ||
		params.Parallelism == 0 || params.Parallelism > maxParallelism ||
		params.SaltLength < 8 || params.SaltLength > maxSaltLength ||
		params.KeyLength < 16 || params.KeyLength > maxKeyLength {
		return ErrInvalidHash
	}

	return nil
}
