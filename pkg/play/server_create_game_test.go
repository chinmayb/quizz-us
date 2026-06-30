package play

import (
	"context"
	"log/slog"
	"regexp"
	"testing"
	"time"

	pb "github.com/chinmayb/quizz-us/gen/go/api"
	"github.com/chinmayb/quizz-us/pkg/gameengine/quiz"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestCreateGame_Success(t *testing.T) {
	srv := &PlayServer{log: slog.Default()}
	req := &pb.CreateGameRequest{
		GameKindId:       "trivia",
		HostId:           "host-1",
		HostName:         "Alice",
		Categories:       []string{"history", "  science  ", "history", ""},
		QuestionDuration: durationpb.New(15 * time.Second),
		TargetScore:      7,
	}

	resp, err := srv.CreateGame(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateGame returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	matched, err := regexp.MatchString("^[A-Z0-9]{6}$", resp.GetCode())
	if err != nil {
		t.Fatalf("regex error: %v", err)
	}
	if !matched {
		t.Fatalf("expected generated 6-char code, got %q", resp.GetCode())
	}

	if resp.GetHost().GetId() != req.GetHostId() {
		t.Fatalf("expected host id %q, got %q", req.GetHostId(), resp.GetHost().GetId())
	}
	if resp.GetGame().GetGameKindId() != "trivia" {
		t.Fatalf("expected game kind id %q, got %q", "trivia", resp.GetGame().GetGameKindId())
	}
	if resp.GetSummary() == nil || len(resp.GetSummary().GetPlayers()) != 1 {
		t.Fatalf("expected one host player in initial summary, got %+v", resp.GetSummary())
	}

	game, ok := quiz.GetGame(resp.GetCode())
	if !ok {
		t.Fatalf("expected game %q to exist in registry", resp.GetCode())
	}
	if game.HostID != req.GetHostId() {
		t.Fatalf("expected game host id %q, got %q", req.GetHostId(), game.HostID)
	}
	if game.Status != quiz.GameStatusNotStarted {
		t.Fatalf("expected game status NOT_STARTED, got %v", game.Status)
	}
	if game.Settings.QuestionDuration != 15*time.Second {
		t.Fatalf("expected question duration 15s, got %v", game.Settings.QuestionDuration)
	}
	if game.Settings.TargetScore != 7 {
		t.Fatalf("expected target score 7, got %d", game.Settings.TargetScore)
	}
	if len(game.Settings.Categories) != 2 || game.Settings.Categories[0] != "history" || game.Settings.Categories[1] != "science" {
		t.Fatalf("unexpected categories: %+v", game.Settings.Categories)
	}
}

func TestCreateGame_Validation(t *testing.T) {
	srv := &PlayServer{log: slog.Default()}

	_, err := srv.CreateGame(context.Background(), &pb.CreateGameRequest{HostName: "Alice"})
	if err == nil {
		t.Fatal("expected error for missing host_id")
	}

	_, err = srv.CreateGame(context.Background(), &pb.CreateGameRequest{HostId: "h1"})
	if err == nil {
		t.Fatal("expected error for missing host_name")
	}
}

func TestCreateGame_DefaultGameKindId(t *testing.T) {
	srv := &PlayServer{log: slog.Default()}

	resp, err := srv.CreateGame(context.Background(), &pb.CreateGameRequest{
		HostId:   "host-default",
		HostName: "Default Kind",
	})
	if err != nil {
		t.Fatalf("CreateGame returned error: %v", err)
	}
	if resp.GetGame().GetGameKindId() != "quiz" {
		t.Fatalf("expected default game kind id %q, got %q", "quiz", resp.GetGame().GetGameKindId())
	}
}
