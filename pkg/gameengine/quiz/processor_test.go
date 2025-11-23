package quiz

import (
	"context"
	"testing"
	"time"

	pb "github.com/chinmayb/quizz-us/gen/go/api"
	"github.com/chinmayb/quizz-us/pkg/data"
)

type stubQuizEngine struct {
	question *data.QuizData
}

func (s *stubQuizEngine) ProduceQuestions(context.Context, any) (chan *data.QuizData, chan error) {
	qChan := make(chan *data.QuizData, 1)
	qChan <- s.question
	return qChan, nil
}

func (s *stubQuizEngine) ValidateAnswer(context.Context, *data.QuizData) bool {
	return true
}

func TestBroadCastQuestionFansOutToAllPlayers(t *testing.T) {
	t.Cleanup(func() {
		GameRegistry.mu.Lock()
		GameRegistry.games = make(map[string]*Game)
		GameRegistry.mu.Unlock()
	})

	code := "TESTCODE"
	question := &data.QuizData{Id: "42", Question: "Life?", Answer: "Yes"}

	// prepare players
	players := map[string]*PlayerObj{
		"p1": {Player: &pb.Player{Id: "p1"}, QuestionForPlayer: make(chan *data.QuizData, 1)},
		"p2": {Player: &pb.Player{Id: "p2"}, QuestionForPlayer: make(chan *data.QuizData, 1)},
		"p3": {Player: &pb.Player{Id: "p3"}, QuestionForPlayer: make(chan *data.QuizData, 1)},
	}

	GameRegistry.mu.Lock()
	GameRegistry.games[code] = &Game{players: players}
	GameRegistry.mu.Unlock()

	engine := &stubQuizEngine{question: question}

	if err := broadCastQuestion(context.Background(), code, engine); err != nil {
		t.Fatalf("broadCastQuestion returned error: %v", err)
	}

	received := make([]*data.QuizData, 0, len(players))
	for id, pl := range players {
		select {
		case q := <-pl.QuestionForPlayer:
			if q == nil {
				t.Fatalf("player %s received nil question", id)
			}
			if q != question {
				t.Fatalf("player %s received different question pointer", id)
			}
			received = append(received, q)
		case <-time.After(time.Second):
			t.Fatalf("player %s did not receive question", id)
		}
	}

	if len(received) != len(players) {
		t.Fatalf("expected %d questions, got %d", len(players), len(received))
	}

	for i := 1; i < len(received); i++ {
		if received[i].Id != received[0].Id {
			t.Fatalf("player %d received different question id %s vs %s", i, received[i].Id, received[0].Id)
		}
	}
}
