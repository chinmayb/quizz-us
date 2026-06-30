package quiz

import (
	"errors"
	"regexp"
	"testing"
)

func TestGenerateGameCode_Format(t *testing.T) {
	t.Cleanup(resetGameRegistry)

	code, err := GenerateGameCode()
	if err != nil {
		t.Fatalf("GenerateGameCode returned error: %v", err)
	}

	matched, err := regexp.MatchString("^[A-Z0-9]{6}$", code)
	if err != nil {
		t.Fatalf("regex error: %v", err)
	}
	if !matched {
		t.Fatalf("expected 6-char uppercase alphanumeric code, got %q", code)
	}
}

func TestGenerateUniqueGameCode_RetriesOnCollision(t *testing.T) {
	t.Cleanup(resetGameRegistry)

	AddGame("ABC123", &Game{players: PlayersMap{}})

	calls := 0
	candidateFn := func() (string, error) {
		calls++
		if calls == 1 {
			return "ABC123", nil
		}
		return "XYZ789", nil
	}

	code, err := generateUniqueGameCode(3, candidateFn)
	if err != nil {
		t.Fatalf("generateUniqueGameCode returned error: %v", err)
	}
	if code != "XYZ789" {
		t.Fatalf("expected non-colliding code XYZ789, got %q", code)
	}
	if calls != 2 {
		t.Fatalf("expected 2 candidate attempts, got %d", calls)
	}
}

func TestGenerateUniqueGameCode_Exhausted(t *testing.T) {
	t.Cleanup(resetGameRegistry)

	AddGame("AAAAAA", &Game{players: PlayersMap{}})

	candidateFn := func() (string, error) {
		return "AAAAAA", nil
	}

	_, err := generateUniqueGameCode(2, candidateFn)
	if !errors.Is(err, errCodeGenExhausted) {
		t.Fatalf("expected errCodeGenExhausted, got %v", err)
	}
}

func TestGenerateUniqueGameCode_PropagatesCandidateError(t *testing.T) {
	t.Cleanup(resetGameRegistry)

	candidateErr := errors.New("entropy unavailable")
	candidateFn := func() (string, error) {
		return "", candidateErr
	}

	_, err := generateUniqueGameCode(2, candidateFn)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, candidateErr) {
		t.Fatalf("expected wrapped candidate error, got %v", err)
	}
}

func TestGenerateGameCode_NotAlwaysSameValue(t *testing.T) {
	t.Cleanup(resetGameRegistry)

	seen := make(map[string]struct{})
	for i := 0; i < 20; i++ {
		code, err := GenerateGameCode()
		if err != nil {
			t.Fatalf("GenerateGameCode returned error: %v", err)
		}
		seen[code] = struct{}{}
	}

	if len(seen) < 2 {
		t.Fatalf("expected at least 2 distinct codes across runs, got %d", len(seen))
	}
}

func resetGameRegistry() {
	GameRegistry.mu.Lock()
	defer GameRegistry.mu.Unlock()
	GameRegistry.games = make(map[string]*Game)
}
