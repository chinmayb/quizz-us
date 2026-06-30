package quiz

import (
	"context"
	"fmt"
	log "log/slog"
	"sync"
	"time"

	pb "github.com/chinmayb/quizz-us/gen/go/api"
	"github.com/chinmayb/quizz-us/pkg/data"
)

// Declare the global variable
var GameRegistry = GameRegistryObj{
	games: make(map[string]*Game),
}

type GameProcessor interface {
	Process(context.Context) error
}

type GamePro struct {
	Code string
}

type GameStatus int

const (
	GameStatusNotStarted GameStatus = iota
	GameStatusInProgress
	GameStatusFinished
)

type GameSettings struct {
	Categories       []string
	QuestionDuration time.Duration
	TargetScore      int32
}

func defaultGameSettings() GameSettings {
	return GameSettings{
		QuestionDuration: 30 * time.Second,
		TargetScore:      10,
	}
}

// playersMap for id and player
type PlayersMap map[string]*PlayerObj

// GameRegistry is a map with a read-write lock that contains games queues
type GameRegistryObj struct {
	mu sync.RWMutex
	// TODO add per game level locking inside this
	games map[string]*Game
}

func GetGame(code string) (*Game, bool) {
	GameRegistry.mu.RLock()
	defer GameRegistry.mu.RUnlock()

	game, exists := GameRegistry.games[code]
	return game, exists
}

// get all the players for the current game with code
func GetAllPlayers(code string) (PlayersMap, error) {
	GameRegistry.mu.RLock()
	defer GameRegistry.mu.RUnlock()

	game, exists := GameRegistry.games[code]
	if !exists {
		return nil, fmt.Errorf("game not found")
	}
	return game.players, nil
}

// ResultChannels returns a lock-protected snapshot of every connected player's
// Result channel for a game. Snapshotting under the registry lock avoids racing
// with concurrent channel swaps performed during a player rejoin.
func ResultChannels(code string) []chan *pb.GamePlay {
	GameRegistry.mu.RLock()
	defer GameRegistry.mu.RUnlock()

	game, exists := GameRegistry.games[code]
	if !exists {
		return nil
	}

	channels := make([]chan *pb.GamePlay, 0, len(game.players))
	for _, p := range game.players {
		if p != nil && p.Result != nil {
			channels = append(channels, p.Result)
		}
	}
	return channels
}

// GetPlayer get player by ID for a running game
func GetPlayer(code string, playerID string) (*PlayerObj, error) {
	GameRegistry.mu.RLock()
	defer GameRegistry.mu.RUnlock()

	g, exists := GameRegistry.games[code]
	if !exists {
		return nil, fmt.Errorf("game not found")
	}

	playerChan, exists := g.players[playerID]
	if !exists {
		return nil, fmt.Errorf("player not found")
	}
	return playerChan, nil
}

func RemoveGame(gameID string) {
	GameRegistry.mu.Lock()
	defer GameRegistry.mu.Unlock()

	if game, ok := GameRegistry.games[gameID]; ok {
		if game.cancelFn != nil {
			game.cancelFn()
		}
	}
	delete(GameRegistry.games, gameID)
}

func RemovePlayerFromRegistry(gameID string, playerID string) {
	GameRegistry.mu.Lock()
	defer GameRegistry.mu.Unlock()

	game, ok := GameRegistry.games[gameID]
	if !ok {
		return
	}
	delete(game.players, playerID)
}

func DisconnectPlayer(gameID string, playerID string) {
	var cancelFn context.CancelFunc

	GameRegistry.mu.Lock()
	game, ok := GameRegistry.games[gameID]
	if !ok {
		GameRegistry.mu.Unlock()
		return
	}
	p, ok := game.players[playerID]
	if !ok {
		GameRegistry.mu.Unlock()
		return
	}
	p.Player.Status = pb.PlayerStatus_DISCONNECTED
	cancelFn = p.cancelCtx
	GameRegistry.mu.Unlock()

	if cancelFn != nil {
		cancelFn()
	}
}

func updatePlayerScore(gameID string, playerID string, score int32) {
	GameRegistry.mu.Lock()
	defer GameRegistry.mu.Unlock()

	game, ok := GameRegistry.games[gameID]
	if !ok {
		return
	}
	if p, ok := game.players[playerID]; ok {
		p.Player.Score = score
	}
}

func AddPlayerToRegistry(gameID string, playerObj *PlayerObj) {
	if playerObj == nil {
		return
	}
	GameRegistry.mu.Lock()
	defer GameRegistry.mu.Unlock()

	if game, ok := GameRegistry.games[gameID]; ok {
		game.players[playerObj.Player.Id] = playerObj
	}
}

func AddGame(gameID string, processor *Game) (exists bool) {
	GameRegistry.mu.Lock()
	defer GameRegistry.mu.Unlock()

	if processor == nil {
		return !exists
	}

	// nothing to do if already exists
	if _, ok := GameRegistry.games[gameID]; ok {
		return true
	}

	log.Debug("initializing game: ", "code", gameID)
	GameRegistry.games[gameID] = processor
	return false
}

// PlayerObj player obj
type PlayerObj struct {
	Player            *pb.Player
	QuestionForPlayer chan *data.QuizData
	Result            chan *pb.GamePlay
	AnswerFromPlayer  *data.QuizData
	cancelCtx         context.CancelFunc
}

func (p *PlayerObj) SetCancelFunc(fn context.CancelFunc) {
	p.cancelCtx = fn
}

func NewGameProcessor(gameChan chan GamePro, ansChan chan PlayerObj) *Game {
	areAllAnsweredRight := make(chan bool, 1)
	players := make(map[string]*PlayerObj)

	gp := gameProcessor{
		BeginGame:           gameChan,
		AnswerChan:          ansChan,
		areAllAnsweredRight: areAllAnsweredRight,
	}
	return &Game{
		GamePro:   gp,
		players:   players,
		Code:      "",
		Status:    GameStatusNotStarted,
		Settings:  defaultGameSettings(),
		StartTime: time.Time{},
		cancelFn:  nil,
	}
}

type Game struct {
	GamePro      gameProcessor
	players      PlayersMap
	Code         string
	HostID       string
	Status       GameStatus
	Settings     GameSettings
	StartTime    time.Time
	cancelFn     context.CancelFunc
	lastQuestion *data.QuizData
	lastQMu      sync.RWMutex
}

func (g *Game) SetCancelFn(fn context.CancelFunc) {
	g.cancelFn = fn
}

func (g *Game) markStarted(now time.Time) {
	if now.IsZero() {
		now = time.Now()
	}
	g.Status = GameStatusInProgress
	g.StartTime = now
}

func (g *Game) markFinished() {
	g.Status = GameStatusFinished
}

type gameProcessor struct {
	AnswerChan          chan PlayerObj
	areAllAnsweredRight chan bool
	IsGameEnded         chan bool
	BeginGame           chan GamePro
}

func (g *Game) Process(ctx context.Context) error {
	log.Debug("Game about to begin", "code", g.Code)
	for {
		select {
		// TODO crashes needs to be handled from a reconciler if a game is running & stuck for a while
		case gameObj := <-g.GamePro.BeginGame:
			// check code and verify
			if p, ok := GetGame(gameObj.Code); !ok {
				return fmt.Errorf("no playing registry found")
			} else {
				log.Info("Processor", "object", p)
			}
			g.markStarted(time.Now())
			g.Play(ctx, gameObj.Code)

		case <-ctx.Done():
			log.Info("context done in game processor")
			return nil
		}
	}
}

// Play
func (g *Game) Play(ctx context.Context, code string) error {
	ticker := time.NewTicker(time.Second * 30)
	quizengine := NewQuizEnginer()
	time.Sleep(1 * time.Second)
	log.Debug("Begining game")

	currentQuestion, err := broadCastQuestion(ctx, code, quizengine)
	if err != nil {
		return err
	}
	answeredCorrectly := make(map[string]bool)

	for {
		select {
		case <-ticker.C:
			log.Info("sending the next question")
			broadCastAnswerReveal(ctx, code, currentQuestion)
			currentQuestion, err = broadCastQuestion(ctx, code, quizengine)
			if err != nil {
				return err
			}
			answeredCorrectly = make(map[string]bool)

		case playerobj := <-g.GamePro.AnswerChan:
			log.Info("received", "answer", playerobj.AnswerFromPlayer, "player", playerobj.Player)
			if !quizengine.ValidateAnswer(ctx, playerobj.AnswerFromPlayer) {
				log.Info("Wrong answer")
				continue
			}
			log.Info("Right answer from player", "player", playerobj.Player.Id)
			playerobj.Player.Score++
			updatePlayerScore(g.Code, playerobj.Player.Id, playerobj.Player.Score)
			answeredCorrectly[playerobj.Player.Id] = true
			if players, err := GetAllPlayers(code); err == nil && len(players) > 0 && len(answeredCorrectly) >= len(players) {
				select {
				case g.GamePro.areAllAnsweredRight <- true:
				default:
				}
			}

		case <-g.GamePro.areAllAnsweredRight:
			broadCastAnswerReveal(ctx, code, currentQuestion)
			currentQuestion, err = broadCastQuestion(ctx, code, quizengine)
			if err != nil {
				return err
			}
			answeredCorrectly = make(map[string]bool)

		case <-g.GamePro.IsGameEnded:
			if err := broadCastResult(ctx, code); err != nil {
				return err
			}

		case <-ctx.Done():
			log.Info("context done")
			return nil
		}
	}
}

func broadCastQuestion(ctx context.Context, code string, q QuizEnginer) (*data.QuizData, error) {
	players, _ := GetAllPlayers(code)
	quizdata, err := q.ProduceQuestions(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error in play %v", err)
	}
	question := <-quizdata
	if game, ok := GetGame(code); ok {
		game.lastQMu.Lock()
		game.lastQuestion = question
		game.lastQMu.Unlock()
	}
	for _, player := range players {
		p := player
		if p.Player.Status == pb.PlayerStatus_DISCONNECTED {
			continue
		}
		go func(pl *PlayerObj) {
			log.Debug("sending question to", "player", pl.Player.Id)
			pl.QuestionForPlayer <- question
		}(p)
	}
	return question, nil
}

func GetLastQuestion(gameID string) *data.QuizData {
	game, ok := GetGame(gameID)
	if !ok {
		return nil
	}
	game.lastQMu.RLock()
	defer game.lastQMu.RUnlock()
	return game.lastQuestion
}

func RejoinPlayer(gameID string, playerID string, newQ chan *data.QuizData, newResult chan *pb.GamePlay, newCancel context.CancelFunc) (*data.QuizData, bool) {
	GameRegistry.mu.Lock()
	game, ok := GameRegistry.games[gameID]
	if !ok {
		GameRegistry.mu.Unlock()
		return nil, false
	}
	p, ok := game.players[playerID]
	if !ok {
		GameRegistry.mu.Unlock()
		return nil, false
	}
	if p.cancelCtx != nil {
		p.cancelCtx()
	}
	p.QuestionForPlayer = newQ
	p.Result = newResult
	p.cancelCtx = newCancel
	p.Player.Status = pb.PlayerStatus_PLAYING
	GameRegistry.mu.Unlock()

	lastQ := GetLastQuestion(gameID)
	return lastQ, true
}

func broadCastAnswerReveal(_ context.Context, code string, question *data.QuizData) {
	players, _ := GetAllPlayers(code)
	reveal := &pb.GamePlay{
		Cmd: &pb.GamePlay_Command{Command: &pb.GamePlayCommand{
			Id:            question.Id,
			CorrectAnswer: question.Answer,
		}},
	}
	for _, player := range players {
		p := player
		if p.Player.Status == pb.PlayerStatus_DISCONNECTED {
			continue
		}
		go func(pl *PlayerObj) {
			pl.Result <- reveal
		}(p)
	}
}

func broadCastResult(_ context.Context, code string) error {
	players, _ := GetAllPlayers(code)

	if game, ok := GetGame(code); ok {
		game.markFinished()
	}

	playerList := make([]*pb.Player, 0, len(players))
	for _, p := range players {
		playerList = append(playerList, &pb.Player{
			Id:    p.Player.Id,
			Name:  p.Player.Name,
			Score: p.Player.Score,
		})
	}

	result := &pb.GamePlay{
		Cmd: &pb.GamePlay_Summary{Summary: &pb.GameSummary{
			Players: playerList,
			Status:  pb.GamePlayStatus_GAME_OVER,
		}},
	}

	for _, player := range players {
		p := player
		go func(pl *PlayerObj) {
			pl.Result <- result
		}(p)
	}
	return nil
}
