package quiz

import (
	"context"
	"testing"
	"time"

	pb "github.com/chinmayb/quizz-us/gen/go/api"
)

func TestNewGameProcessor_DefaultState(t *testing.T) {
	game := NewGameProcessor(make(chan GamePro, 1), make(chan PlayerObj, 1))

	if game.Status != GameStatusNotStarted {
		t.Fatalf("expected default status NOT_STARTED, got %v", game.Status)
	}
	if !game.StartTime.IsZero() {
		t.Fatalf("expected zero StartTime before game starts, got %v", game.StartTime)
	}
	if game.Settings.QuestionDuration != 30*time.Second {
		t.Fatalf("expected default question duration 30s, got %v", game.Settings.QuestionDuration)
	}
	if game.Settings.TargetScore != 10 {
		t.Fatalf("expected default target score 10, got %d", game.Settings.TargetScore)
	}
	if len(game.Settings.Categories) != 0 {
		t.Fatalf("expected empty default categories, got %d", len(game.Settings.Categories))
	}
}

func TestGameStatusTransitions(t *testing.T) {
	game := NewGameProcessor(make(chan GamePro, 1), make(chan PlayerObj, 1))

	startedAt := time.Now()
	game.markStarted(startedAt)
	if game.Status != GameStatusInProgress {
		t.Fatalf("expected IN_PROGRESS after start, got %v", game.Status)
	}
	if !game.StartTime.Equal(startedAt) {
		t.Fatalf("expected StartTime to be set to %v, got %v", startedAt, game.StartTime)
	}

	game.markFinished()
	if game.Status != GameStatusFinished {
		t.Fatalf("expected FINISHED after markFinished, got %v", game.Status)
	}
}

func TestMarkStarted_WithZeroTimeSetsNow(t *testing.T) {
	game := NewGameProcessor(make(chan GamePro, 1), make(chan PlayerObj, 1))

	before := time.Now()
	game.markStarted(time.Time{})
	after := time.Now()

	if game.Status != GameStatusInProgress {
		t.Fatalf("expected IN_PROGRESS after markStarted with zero time, got %v", game.Status)
	}
	if game.StartTime.IsZero() {
		t.Fatal("expected StartTime to be set when markStarted receives zero time")
	}
	if game.StartTime.Before(before) || game.StartTime.After(after) {
		t.Fatalf("expected StartTime between %v and %v, got %v", before, after, game.StartTime)
	}
}

func TestBroadCastResult_MarksFinishedAndBroadcastsSummary(t *testing.T) {
	resetGameRegistryForStateTests()
	t.Cleanup(resetGameRegistryForStateTests)

	const code = "STATE-RESULT"

	p1Result := make(chan *pb.GamePlay, 1)
	p2Result := make(chan *pb.GamePlay, 1)

	game := &Game{
		Status: GameStatusInProgress,
		players: PlayersMap{
			"p1": {
				Player: &pb.Player{Id: "p1", Name: "Alice", Score: 3},
				Result: p1Result,
			},
			"p2": {
				Player: &pb.Player{Id: "p2", Name: "Bob", Score: 5},
				Result: p2Result,
			},
		},
	}

	GameRegistry.mu.Lock()
	GameRegistry.games[code] = game
	GameRegistry.mu.Unlock()

	if err := broadCastResult(context.Background(), code); err != nil {
		t.Fatalf("broadCastResult returned error: %v", err)
	}

	if game.Status != GameStatusFinished {
		t.Fatalf("expected game status FINISHED after broadCastResult, got %v", game.Status)
	}

	assertSummary := func(ch <-chan *pb.GamePlay) {
		select {
		case msg := <-ch:
			if msg == nil || msg.GetSummary() == nil {
				t.Fatal("expected summary message, got nil")
			}
			summary := msg.GetSummary()
			if summary.GetStatus() != pb.GamePlayStatus_GAME_OVER {
				t.Fatalf("expected GAME_OVER status, got %v", summary.GetStatus())
			}
			if len(summary.GetPlayers()) != 2 {
				t.Fatalf("expected 2 players in summary, got %d", len(summary.GetPlayers()))
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for summary broadcast")
		}
	}

	assertSummary(p1Result)
	assertSummary(p2Result)
}

func resetGameRegistryForStateTests() {
	GameRegistry.mu.Lock()
	defer GameRegistry.mu.Unlock()
	GameRegistry.games = make(map[string]*Game)
}
