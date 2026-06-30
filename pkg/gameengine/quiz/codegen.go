package quiz

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
)

const (
	gameCodeLength         = 6
	defaultCodeGenAttempts = 10
)

var (
	gameCodeAlphabet   = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	errCodeGenExhausted = errors.New("unable to generate a unique game code")
)

// GenerateGameCode returns a 6-character alphanumeric game code and retries on collisions.
func GenerateGameCode() (string, error) {
	return generateUniqueGameCode(defaultCodeGenAttempts, generateRandomCodeCandidate)
}

func generateUniqueGameCode(maxAttempts int, candidateFn func() (string, error)) (string, error) {
	if maxAttempts <= 0 {
		return "", errCodeGenExhausted
	}

	for i := 0; i < maxAttempts; i++ {
		candidate, err := candidateFn()
		if err != nil {
			return "", fmt.Errorf("generate game code candidate: %w", err)
		}

		if _, exists := GetGame(candidate); !exists {
			return candidate, nil
		}
	}

	return "", errCodeGenExhausted
}

func generateRandomCodeCandidate() (string, error) {
	b := make([]byte, gameCodeLength)
	max := big.NewInt(int64(len(gameCodeAlphabet)))

	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = gameCodeAlphabet[n.Int64()]
	}

	return string(b), nil
}
