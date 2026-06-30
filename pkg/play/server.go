package play

import (
	"context"
	"fmt"
	"io"
	log "log/slog"
	"strings"
	"time"

	pb "github.com/chinmayb/quizz-us/gen/go/api"
	"github.com/chinmayb/quizz-us/pkg/data"
	"github.com/chinmayb/quizz-us/pkg/gameengine/quiz"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func NewPlayServer(log *log.Logger) pb.GamesServer {
	s := &PlayServer{log: log}
	return s
}

// imageSrcString returns the question's image URL as a string, or empty if unset.
func imageSrcString(q *data.QuizData) string {
	if q == nil || q.ImageSrc == nil {
		return ""
	}
	return q.ImageSrc.String()
}

// validateRequest ensures the inbound message carries the required identifiers.
func validateRequest(in *pb.GamePlay) (string, error) {
	code := in.GetCode()
	if code == "" {
		return "", fmt.Errorf("code not found %s", code)
	}
	if in.GetId() == "" {
		return "", fmt.Errorf("player ID not found")
	}
	return code, nil
}

// buildQuestionGamePlay constructs the outbound question message for a player.
// CorrectAnswer is intentionally omitted — it is only exposed in the results/summary phase.
func buildQuestionGamePlay(playerID, code string, q *data.QuizData) *pb.GamePlay {
	return &pb.GamePlay{
		Id:   playerID,
		Code: code,
		Cmd: &pb.GamePlay_Command{Command: &pb.GamePlayCommand{
			Id:           q.Id,
			Question:     q.Question,
			QuestionTime: timestamppb.Now(),
			ImageSrc:     imageSrcString(q),
		}},
	}
}

// playerSession owns the outbound streaming loop for a single connected player.
type playerSession struct {
	log    *log.Logger
	stream pb.Games_PlayServer
	player *quiz.PlayerObj
	code   string
	ticker *time.Ticker
}

// run drives the player's outbound stream, handling rejoin catch-up first.
func (s *playerSession) run(playerID string, isRejoin bool) {
	if isRejoin {
		if !s.handleRejoin(playerID) {
			return
		}
	}
	s.runOutboundLoop(playerID)
}

// handleRejoin swaps the player's channels for fresh ones bound to this stream
// and replays the last question. Returns false if the rejoin could not complete.
func (s *playerSession) handleRejoin(playerID string) bool {
	newQ := make(chan *data.QuizData)
	newResult := make(chan *pb.GamePlay)
	_, newCancel := context.WithCancel(s.stream.Context())
	lastQ, ok := quiz.RejoinPlayer(s.code, playerID, newQ, newResult, newCancel)
	if !ok {
		s.log.Warn("rejoin failed", "player", playerID)
		return false
	}
	s.player.QuestionForPlayer = newQ
	s.player.Result = newResult

	if lastQ != nil {
		if err := s.stream.Send(buildQuestionGamePlay(playerID, s.code, lastQ)); err != nil {
			s.log.Error("error sending catch-up question", "err", err)
			return false
		}
	}
	return true
}

// runOutboundLoop fans questions, results, and keepalives to the player until
// the stream context is cancelled.
func (s *playerSession) runOutboundLoop(playerID string) {
	for {
		select {
		case quizQuestion := <-s.player.QuestionForPlayer:
			s.log.Info("Sending question to player", "player", s.player.Player.Id)
			if err := s.stream.Send(buildQuestionGamePlay(s.player.Player.Id, s.code, quizQuestion)); err != nil {
				s.log.Error("error while sending question", "err", err)
				return
			}
		case result := <-s.player.Result:
			if err := s.stream.Send(result); err != nil {
				return
			}
		// This is for heartbeat - includes current player list
		case <-s.ticker.C:
			s.log.Debug("Sending keepalive with player list")
			out := &pb.GamePlay{
				Cmd: &pb.GamePlay_Summary{Summary: buildGameSummary(s.code, pb.GamePlayStatus_NOT_STARTED)},
			}
			if err := s.stream.Send(out); err != nil {
				return
			}
		case <-s.stream.Context().Done():
			s.log.Info("player exited", "ID", playerID)
			quiz.DisconnectPlayer(s.code, s.player.Player.Id)

			players, err := quiz.GetAllPlayers(s.code)
			if err != nil {
				return
			}
			s.log.Info("game registry", "players", players)
			activeCount := 0
			for _, pl := range players {
				if pl.Player.Status != pb.PlayerStatus_DISCONNECTED {
					activeCount++
				}
			}
			if activeCount == 0 {
				s.log.Info("no active players left, removing the game")
				quiz.RemoveGame(s.code)
				return
			}

			// Let remaining players know this one left.
			broadcastLobby(s.code)
			return
		}
	}
}

type PlayServer struct {
	log *log.Logger
	pb.UnimplementedGamesServer
}

func initGame(_ context.Context, code string, hostID string, settings *quiz.GameSettings) (*quiz.Game, error) {
	gameChan := make(chan quiz.GamePro)
	ansChan := make(chan quiz.PlayerObj)
	p := quiz.NewGameProcessor(gameChan, ansChan)
	p.Code = code
	p.HostID = hostID
	if settings != nil {
		p.Settings = *settings
	}

	// ADD it to the in memory registry
	if alreadyExists := quiz.AddGame(code, p); alreadyExists {
		log.Warn("game already exists", "gameID", code)
		return nil, fmt.Errorf("game already exists")
	}

	gameCtx, cancel := context.WithCancel(context.Background())
	p.SetCancelFn(cancel)

	// should start only once
	go func() {
		if err := p.Process(gameCtx); err != nil {
			log.Error("game processor stopped", "err", err, "gameID", code)
		}
	}()

	return p, nil
}

func sanitizeCategories(categories []string) []string {
	seen := make(map[string]struct{}, len(categories))
	out := make([]string, 0, len(categories))
	for _, category := range categories {
		normalized := strings.TrimSpace(category)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func buildGameSettings(req *pb.CreateGameRequest) quiz.GameSettings {
	settings := quiz.GameSettings{}
	if req == nil {
		return settings
	}

	settings = quiz.GameSettings{
		Categories: sanitizeCategories(req.GetCategories()),
	}

	if req.GetQuestionDuration() != nil {
		if d := req.GetQuestionDuration().AsDuration(); d > 0 {
			settings.QuestionDuration = d
		}
	}
	if req.GetTargetScore() > 0 {
		settings.TargetScore = req.GetTargetScore()
	}

	if settings.QuestionDuration <= 0 {
		settings.QuestionDuration = 30 * time.Second
	}
	if settings.TargetScore <= 0 {
		settings.TargetScore = 10
	}

	return settings
}

func (p *PlayServer) CreateGame(ctx context.Context, in *pb.CreateGameRequest) (*pb.CreateGameResponse, error) {
	if in == nil {
		return nil, fmt.Errorf("request is required")
	}
	if strings.TrimSpace(in.GetHostId()) == "" {
		return nil, fmt.Errorf("host_id is required")
	}
	if strings.TrimSpace(in.GetHostName()) == "" {
		return nil, fmt.Errorf("host_name is required")
	}

	code, err := quiz.GenerateGameCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate game code: %w", err)
	}

	settings := buildGameSettings(in)
	gameKindID := strings.TrimSpace(in.GetGameKindId())
	if gameKindID == "" {
		gameKindID = "quiz"
	}

	game, err := initGame(ctx, code, in.GetHostId(), &settings)
	if err != nil {
		return nil, err
	}

	host := &pb.Player{
		Id:     in.GetHostId(),
		Name:   in.GetHostName(),
		Status: pb.PlayerStatus_WAITING,
		Score:  0,
	}

	hostObj := &quiz.PlayerObj{
		Player:            host,
		QuestionForPlayer: make(chan *data.QuizData, 1),
		Result:            make(chan *pb.GamePlay, 1),
	}
	quiz.AddPlayerToRegistry(code, hostObj)

	now := timestamppb.Now()
	resp := &pb.CreateGameResponse{
		Code: code,
		Game: &pb.Game{
			GameKindId: gameKindID,
			Code:       code,
			Status:     "NOT_STARTED",
			CreatedAt:  now,
			UpdatedAt:  now,
			Spec: &pb.Game_Spec{
				QuestionDuration: durationpb.New(game.Settings.QuestionDuration),
				TargetScore:      game.Settings.TargetScore,
				TargetTime:       in.GetTargetTime(),
			},
		},
		Host:    host,
		Summary: buildGameSummary(code, pb.GamePlayStatus_NOT_STARTED),
	}
	return resp, nil
}

// buildGameSummary creates a GameSummary with the current player list for a game
func buildGameSummary(code string, status pb.GamePlayStatus) *pb.GameSummary {
	players, err := quiz.GetAllPlayers(code)
	if err != nil {
		log.Error("failed to get players for summary", "gameID", code, "err", err)
		return &pb.GameSummary{Status: status}
	}

	// Build player list for the summary
	playerList := make([]*pb.Player, 0, len(players))
	for _, p := range players {
		playerList = append(playerList, &pb.Player{
			Id:     p.Player.Id,
			Name:   p.Player.Name,
			Score:  p.Player.Score,
			Status: pb.PlayerStatus_WAITING,
		})
	}

	return &pb.GameSummary{
		Players: playerList,
		Status:  status,
	}
}

// broadcastLobby pushes the current lobby player list to every connected player
// so all screens update promptly when someone joins or leaves. Sends are async
// with a timeout to avoid blocking on (or leaking goroutines for) dead streams.
func broadcastLobby(code string) {
	channels := quiz.ResultChannels(code)
	if len(channels) == 0 {
		return
	}

	out := &pb.GamePlay{
		Cmd: &pb.GamePlay_Summary{Summary: buildGameSummary(code, pb.GamePlayStatus_NOT_STARTED)},
	}

	for _, ch := range channels {
		resultCh := ch
		go func() {
			select {
			case resultCh <- out:
			case <-time.After(2 * time.Second):
			}
		}()
	}
}

/*
Play logic-
if p1/p2:
1) init the game
2) send heart beats
3) wait for the questions/game to begin
4) send answers
5) quit

if host:
1) p1 actions + begin the game
*/
func (p *PlayServer) Play(stream pb.Games_PlayServer) error {
	qForPlayChan := make(chan *data.QuizData)
	playObj := &quiz.PlayerObj{
		QuestionForPlayer: qForPlayChan,
		Result:            make(chan *pb.GamePlay),
		Player:            &pb.Player{},
	}
	ticker := time.NewTicker(30 * time.Second)
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			log.Info("client errored")
			return nil
		}
		if err != nil {
			return err
		}
		// check incoming request if the id exists during join or reconnection

		code, err := validateRequest(in)
		if err != nil {
			return err
		}
		log := p.log.With("gameID", code)
		log.Info("Received request", "request", in)
		// TODO check if client is exited if n heartbeats missed
		if in.GetAction() == pb.GamePlayAction_HEARTBEAT {
			_, ok := quiz.GetGame(code)
			if !ok {
				return fmt.Errorf("game ID not found")
			}
			continue
		}

		// If command is empty add treat it as heartbeat and add him to the registry
		if in.GetAction() == pb.GamePlayAction_JOIN {
			// shouldnt send the same
			log.Info("Player joined", "player", in.GetId(), "name", in.GetName())

			// Games are normally created via the CreateGame RPC. Fall back to
			// creating one on first JOIN if it does not exist yet (backward compat).
			if _, gameExists := quiz.GetGame(code); !gameExists {
				if _, err := initGame(stream.Context(), in.GetCode(), in.GetId(), nil); err != nil {
					log.Error("failed to initialize game", "err", err)
					return err
				}
			}

			// Bind this stream's player identity for logging/disconnect handling.
			playObj.Player.Id = in.GetId()
			if in.GetName() != "" {
				playObj.Player.Name = in.GetName()
			}

			_, playerErr := quiz.GetPlayer(code, in.GetId())
			isRejoin := playerErr == nil

			if !isRejoin {
				_, playerCancel := context.WithCancel(stream.Context())
				playObj.SetCancelFunc(playerCancel)

				quiz.AddPlayerToRegistry(code, playObj)
			}

			session := &playerSession{
				log:    log,
				stream: stream,
				player: playObj,
				code:   code,
				ticker: ticker,
			}
			go session.run(in.GetId(), isRejoin)

			// Notify all players (including the joiner) of the updated lobby.
			broadcastLobby(code)
		}

		if in.GetAction() == pb.GamePlayAction_BEGIN {
			p, ok := quiz.GetGame(code)
			if !ok {
				return fmt.Errorf("game ID not found")
			}
			if p.HostID != in.GetId() {
				return fmt.Errorf("only the host can begin the game")
			}
			gp := quiz.GamePro{Code: code}
			p.GamePro.BeginGame <- gp
			log.Info("sent")
		}

		// Check if the player left
		if in.GetAction() == pb.GamePlayAction_END {
			log.Info("player left, ta ta ", "ID", in.GetId())
			quiz.RemovePlayerFromRegistry(code, in.GetId())
			continue
		}

		if in.GetCommand().GetPlayerAnswer() != "" {
			// send the answer to processor queue

			p := quiz.PlayerObj{
				Player: &pb.Player{Id: in.GetId()},
				AnswerFromPlayer: &data.QuizData{
					Id:     in.GetCommand().GetId(),
					Answer: in.GetCommand().GetPlayerAnswer(),
				},
			}
			log.Info("Received answer from player", "answer", p.AnswerFromPlayer)
			proc, ok := quiz.GetGame(code)
			if !ok {
				return fmt.Errorf("game ID not found")
			}
			proc.GamePro.AnswerChan <- p
		}
	}
}
