package play

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	pb "github.com/chinmayb/quizz-us/gen/go/api"
	"github.com/chinmayb/quizz-us/pkg/data"
	"github.com/chinmayb/quizz-us/pkg/gameengine/quiz"
	"google.golang.org/grpc/metadata"
)

// fakePlayStream is a minimal in-memory pb.Games_PlayServer for unit tests.
type fakePlayStream struct {
	ctx     context.Context
	mu      sync.Mutex
	sent    []*pb.GamePlay
	sendErr error
}

func newFakePlayStream(ctx context.Context) *fakePlayStream {
	return &fakePlayStream{ctx: ctx}
}

func (f *fakePlayStream) Send(m *pb.GamePlay) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, m)
	return nil
}

func (f *fakePlayStream) Recv() (*pb.GamePlay, error)  { return nil, io.EOF }
func (f *fakePlayStream) SetHeader(metadata.MD) error  { return nil }
func (f *fakePlayStream) SendHeader(metadata.MD) error { return nil }
func (f *fakePlayStream) SetTrailer(metadata.MD)       {}
func (f *fakePlayStream) Context() context.Context     { return f.ctx }
func (f *fakePlayStream) SendMsg(m any) error          { return nil }
func (f *fakePlayStream) RecvMsg(m any) error          { return nil }

func (f *fakePlayStream) sentCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.sent)
}

func (f *fakePlayStream) lastSent() *pb.GamePlay {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.sent) == 0 {
		return nil
	}
	return f.sent[len(f.sent)-1]
}

func registerGameWithPlayer(t *testing.T, code, playerID string, status pb.PlayerStatus) *quiz.PlayerObj {
	t.Helper()
	game := quiz.NewGameProcessor(make(chan quiz.GamePro, 1), make(chan quiz.PlayerObj, 1))
	game.Code = code
	quiz.AddGame(code, game)

	player := &quiz.PlayerObj{
		Player:            &pb.Player{Id: playerID, Status: status},
		QuestionForPlayer: make(chan *data.QuizData, 1),
		Result:            make(chan *pb.GamePlay, 1),
	}
	quiz.AddPlayerToRegistry(code, player)
	t.Cleanup(func() { quiz.RemoveGame(code) })
	return player
}

func TestValidateRequest(t *testing.T) {
	tests := []struct {
		name     string
		in       *pb.GamePlay
		wantErr  bool
		wantCode string
	}{
		{name: "valid", in: &pb.GamePlay{Id: "p1", Code: "ABC123"}, wantErr: false, wantCode: "ABC123"},
		{name: "missing code", in: &pb.GamePlay{Id: "p1"}, wantErr: true},
		{name: "missing id", in: &pb.GamePlay{Code: "ABC123"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := validateRequest(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if code != tt.wantCode {
				t.Fatalf("expected code %q, got %q", tt.wantCode, code)
			}
		})
	}
}

func TestBuildQuestionGamePlay(t *testing.T) {
	q := &data.QuizData{Id: "42", Question: "Capital of France?", Answer: "Paris"}

	out := buildQuestionGamePlay("p1", "ABC123", q)

	if out.GetId() != "p1" || out.GetCode() != "ABC123" {
		t.Fatalf("unexpected envelope: id=%q code=%q", out.GetId(), out.GetCode())
	}
	cmd := out.GetCommand()
	if cmd == nil {
		t.Fatal("expected command payload")
	}
	if cmd.GetId() != "42" || cmd.GetQuestion() != "Capital of France?" {
		t.Fatalf("unexpected question payload: %+v", cmd)
	}
	if cmd.GetCorrectAnswer() != "" {
		t.Fatalf("correct answer must be omitted, got %q", cmd.GetCorrectAnswer())
	}
	if cmd.GetQuestionTime() == nil {
		t.Fatal("expected question time to be set")
	}
}

func TestPlayerSession_SendsQuestionThenStops(t *testing.T) {
	const code = "SESS-Q"
	player := registerGameWithPlayer(t, code, "p1", pb.PlayerStatus_PLAYING)

	ctx, cancel := context.WithCancel(context.Background())
	stream := newFakePlayStream(ctx)
	session := &playerSession{
		log:    slog.Default(),
		stream: stream,
		player: player,
		code:   code,
		ticker: time.NewTicker(time.Hour), // effectively disabled for this test
	}

	// Queue a question for delivery.
	player.QuestionForPlayer <- &data.QuizData{Id: "7", Question: "2+2?", Answer: "4"}

	done := make(chan struct{})
	go func() {
		session.runOutboundLoop("p1")
		close(done)
	}()

	// Wait until the question is delivered.
	deadline := time.After(2 * time.Second)
	for stream.sentCount() == 0 {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for question to be sent")
		case <-time.After(5 * time.Millisecond):
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("outbound loop did not exit after context cancel")
	}

	sent := stream.sent[0]
	if sent.GetCommand().GetId() != "7" {
		t.Fatalf("expected question id 7, got %q", sent.GetCommand().GetId())
	}
}

func TestPlayerSession_DisconnectRemovesEmptyGame(t *testing.T) {
	const code = "SESS-DC"
	player := registerGameWithPlayer(t, code, "p1", pb.PlayerStatus_PLAYING)

	ctx, cancel := context.WithCancel(context.Background())
	stream := newFakePlayStream(ctx)
	session := &playerSession{
		log:    slog.Default(),
		stream: stream,
		player: player,
		code:   code,
		ticker: time.NewTicker(time.Hour),
	}

	done := make(chan struct{})
	go func() {
		session.runOutboundLoop("p1")
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("outbound loop did not exit after context cancel")
	}

	if _, ok := quiz.GetGame(code); ok {
		t.Fatal("expected game to be removed after last active player disconnected")
	}
}

func TestPlayerSession_HandleRejoinSwapsChannels(t *testing.T) {
	const code = "SESS-REJOIN"
	player := registerGameWithPlayer(t, code, "p1", pb.PlayerStatus_DISCONNECTED)
	oldQ := player.QuestionForPlayer
	oldResult := player.Result

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := newFakePlayStream(ctx)
	session := &playerSession{
		log:    slog.Default(),
		stream: stream,
		player: player,
		code:   code,
		ticker: time.NewTicker(time.Hour),
	}

	if ok := session.handleRejoin("p1"); !ok {
		t.Fatal("expected handleRejoin to succeed")
	}

	if session.player.QuestionForPlayer == oldQ {
		t.Fatal("expected QuestionForPlayer channel to be replaced")
	}
	if session.player.Result == oldResult {
		t.Fatal("expected Result channel to be replaced")
	}
	if player.Player.Status != pb.PlayerStatus_PLAYING {
		t.Fatalf("expected player status PLAYING after rejoin, got %v", player.Player.Status)
	}
}

func TestBroadcastLobby_DeliversToAllPlayers(t *testing.T) {
	const code = "LOBBY-BC"

	// Two players with buffered Result channels so the async sends complete.
	p1 := registerGameWithPlayer(t, code, "p1", pb.PlayerStatus_WAITING)
	p2 := &quiz.PlayerObj{
		Player:            &pb.Player{Id: "p2", Status: pb.PlayerStatus_WAITING},
		QuestionForPlayer: make(chan *data.QuizData, 1),
		Result:            make(chan *pb.GamePlay, 1),
	}
	quiz.AddPlayerToRegistry(code, p2)

	broadcastLobby(code)

	assertLobby := func(name string, ch <-chan *pb.GamePlay) {
		select {
		case msg := <-ch:
			summary := msg.GetSummary()
			if summary == nil {
				t.Fatalf("%s: expected summary message", name)
			}
			if summary.GetStatus() != pb.GamePlayStatus_NOT_STARTED {
				t.Fatalf("%s: expected NOT_STARTED, got %v", name, summary.GetStatus())
			}
			if len(summary.GetPlayers()) != 2 {
				t.Fatalf("%s: expected 2 players, got %d", name, len(summary.GetPlayers()))
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("%s: timed out waiting for lobby broadcast", name)
		}
	}

	assertLobby("p1", p1.Result)
	assertLobby("p2", p2.Result)
}
