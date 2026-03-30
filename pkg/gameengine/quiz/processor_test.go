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

func TestResultChannelAllocated(t *testing.T) {
	result := make(chan *pb.GameSummary)
	playObj := &PlayerObj{
		QuestionForPlayer: make(chan *data.QuizData),
		Result:            result,
		Player:            &pb.Player{},
	}
	if playObj.Result == nil {
		t.Fatal("expected Result channel to be allocated, got nil")
	}
}

func TestRemovePlayerFromRegistry_MissingGame(t *testing.T) {
	t.Cleanup(func() {
		GameRegistry.mu.Lock()
		GameRegistry.games = make(map[string]*Game)
		GameRegistry.mu.Unlock()
	})

	RemovePlayerFromRegistry("nonexistent-game", "player-1")
}

func TestDisconnectPlayer_SetsStatus(t *testing.T) {
	t.Cleanup(func() {
		GameRegistry.mu.Lock()
		GameRegistry.games = make(map[string]*Game)
		GameRegistry.mu.Unlock()
	})

	code := "DC-STATUS"
	playerID := "player-dc"

	player := &PlayerObj{
		Player:            &pb.Player{Id: playerID, Status: pb.PlayerStatus_PLAYING},
		QuestionForPlayer: make(chan *data.QuizData, 1),
		Result:            make(chan *pb.GameSummary, 1),
	}

	GameRegistry.mu.Lock()
	GameRegistry.games[code] = &Game{players: PlayersMap{playerID: player}}
	GameRegistry.mu.Unlock()

	DisconnectPlayer(code, playerID)

	got, err := GetPlayer(code, playerID)
	if err != nil {
		t.Fatalf("GetPlayer: %v", err)
	}
	if got.Player.Status != pb.PlayerStatus_DISCONNECTED {
		t.Fatalf("expected status DISCONNECTED (%d), got %d", pb.PlayerStatus_DISCONNECTED, got.Player.Status)
	}
}

func TestDisconnectPlayer_CancelsContext(t *testing.T) {
	t.Cleanup(func() {
		GameRegistry.mu.Lock()
		GameRegistry.games = make(map[string]*Game)
		GameRegistry.mu.Unlock()
	})

	code := "DC-CANCEL"
	playerID := "player-cancel"

	ctx, cancel := context.WithCancel(context.Background())

	player := &PlayerObj{
		Player:            &pb.Player{Id: playerID, Status: pb.PlayerStatus_PLAYING},
		QuestionForPlayer: make(chan *data.QuizData, 1),
		Result:            make(chan *pb.GameSummary, 1),
		cancelCtx:         cancel,
	}

	GameRegistry.mu.Lock()
	GameRegistry.games[code] = &Game{players: PlayersMap{playerID: player}}
	GameRegistry.mu.Unlock()

	DisconnectPlayer(code, playerID)

	select {
	case <-ctx.Done():
		// success — cancel func was called
	case <-time.After(time.Second):
		t.Fatal("expected context to be cancelled after DisconnectPlayer")
	}
}

func TestBroadcastSkipsDisconnected(t *testing.T) {
	t.Cleanup(func() {
		GameRegistry.mu.Lock()
		GameRegistry.games = make(map[string]*Game)
		GameRegistry.mu.Unlock()
	})

	code := "DC-SKIP"
	question := &data.QuizData{Id: "99", Question: "Skip?", Answer: "Yes"}

	playing := &PlayerObj{
		Player:            &pb.Player{Id: "p-playing", Status: pb.PlayerStatus_PLAYING},
		QuestionForPlayer: make(chan *data.QuizData, 1),
		Result:            make(chan *pb.GameSummary, 1),
	}
	disconnected := &PlayerObj{
		Player:            &pb.Player{Id: "p-disconnected", Status: pb.PlayerStatus_DISCONNECTED},
		QuestionForPlayer: make(chan *data.QuizData, 1),
		Result:            make(chan *pb.GameSummary, 1),
	}

	GameRegistry.mu.Lock()
	GameRegistry.games[code] = &Game{
		players: PlayersMap{
			"p-playing":      playing,
			"p-disconnected": disconnected,
		},
	}
	GameRegistry.mu.Unlock()

	engine := &stubQuizEngine{question: question}
	if err := broadCastQuestion(context.Background(), code, engine); err != nil {
		t.Fatalf("broadCastQuestion: %v", err)
	}

	// PLAYING player should receive
	select {
	case q := <-playing.QuestionForPlayer:
		if q.Id != question.Id {
			t.Fatalf("playing player got wrong question: %s", q.Id)
		}
	case <-time.After(time.Second):
		t.Fatal("playing player did not receive question")
	}

	// DISCONNECTED player should NOT receive
	select {
	case <-disconnected.QuestionForPlayer:
		t.Fatal("disconnected player should NOT receive a question")
	case <-time.After(200 * time.Millisecond):
		// success — no message sent
	}
}

func TestDisconnectPlayer_MissingGame(t *testing.T) {
	t.Cleanup(func() {
		GameRegistry.mu.Lock()
		GameRegistry.games = make(map[string]*Game)
		GameRegistry.mu.Unlock()
	})

	// Should not panic
	DisconnectPlayer("nonexistent-game", "player-1")
}

func TestScoreTrackedInRegistry(t *testing.T) {
	t.Cleanup(func() {
		GameRegistry.mu.Lock()
		GameRegistry.games = make(map[string]*Game)
		GameRegistry.mu.Unlock()
	})

	code := "SCORE01"
	playerID := "player-score"

	player := &PlayerObj{
		Player:            &pb.Player{Id: playerID, Score: 0},
		QuestionForPlayer: make(chan *data.QuizData, 1),
		Result:            make(chan *pb.GameSummary, 1),
	}

	GameRegistry.mu.Lock()
	GameRegistry.games[code] = &Game{
		players: PlayersMap{playerID: player},
	}
	GameRegistry.mu.Unlock()

	updatePlayerScore(code, playerID, 3)

	got, err := GetPlayer(code, playerID)
	if err != nil {
		t.Fatalf("GetPlayer: %v", err)
	}
	if got.Player.Score != 3 {
		t.Fatalf("expected score 3, got %d", got.Player.Score)
	}
}

func TestLastQuestionTracking(t *testing.T) {
	t.Cleanup(func() {
		GameRegistry.mu.Lock()
		GameRegistry.games = make(map[string]*Game)
		GameRegistry.mu.Unlock()
	})

	code := "LASTQ01"
	question := &data.QuizData{Id: "99", Question: "Test?", Answer: "Yes"}

	player := &PlayerObj{
		Player:            &pb.Player{Id: "p1"},
		QuestionForPlayer: make(chan *data.QuizData, 1),
	}

	GameRegistry.mu.Lock()
	GameRegistry.games[code] = &Game{
		players: PlayersMap{"p1": player},
	}
	GameRegistry.mu.Unlock()

	engine := &stubQuizEngine{question: question}
	if err := broadCastQuestion(context.Background(), code, engine); err != nil {
		t.Fatalf("broadCastQuestion returned error: %v", err)
	}

	select {
	case <-player.QuestionForPlayer:
	case <-time.After(time.Second):
		t.Fatal("player did not receive question")
	}

	got := GetLastQuestion(code)
	if got == nil {
		t.Fatal("expected GetLastQuestion to return non-nil, got nil")
	}
	if got != question {
		t.Fatalf("expected GetLastQuestion to return the same pointer, got different")
	}
}

func TestGetLastQuestion_NoGame(t *testing.T) {
	t.Cleanup(func() {
		GameRegistry.mu.Lock()
		GameRegistry.games = make(map[string]*Game)
		GameRegistry.mu.Unlock()
	})

	got := GetLastQuestion("nonexistent-game-xyz")
	if got != nil {
		t.Fatalf("expected nil for nonexistent game, got %v", got)
	}
}
