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

	delete(GameRegistry.games, gameID)
}

func RemovePlayerFromRegistry(gameID string, playerID string) {
	GameRegistry.mu.Lock()
	defer GameRegistry.mu.Unlock()

	delete(GameRegistry.games[gameID].players, playerID)
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
	Result            chan *pb.GameSummary
	AnswerFromPlayer  *data.QuizData
}

func NewGameProcessor(gameChan chan GamePro, ansChan chan PlayerObj) *Game {
	areAllAnsweredRight := make(chan bool)
	players := make(map[string]*PlayerObj)

	gp := gameProcessor{
		BeginGame:           gameChan,
		AnswerChan:          ansChan,
		areAllAnsweredRight: areAllAnsweredRight,
	}
	return &Game{
		gp,
		players,
	}
}

type Game struct {
	GamePro gameProcessor
	players PlayersMap
}

type gameProcessor struct {
	AnswerChan          chan PlayerObj
	areAllAnsweredRight chan bool
	IsGameEnded         chan bool
	BeginGame           chan GamePro
}

func (g *Game) Process(ctx context.Context) error {
	log.Info("Game processor started")
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
			g.Play(ctx, gameObj.Code)

		case <-ctx.Done():
			log.Info("context done in game processor")
			return nil
		}
	}
}

// Play
func (g *Game) Play(ctx context.Context, code string) error {
	// read it from db
	ticker := time.NewTicker(time.Second * 30)
	quizengine := NewQuizEnginer()
	time.Sleep(1 * time.Second)
	log.Debug("Begining game")
	if err := broadCastQuestion(ctx, code, quizengine); err != nil {
		return err
	}
	for {
		select {
		case <-ticker.C:
			log.Info("sending the next question")
			if err := broadCastQuestion(ctx, code, quizengine); err != nil {
				return err
			}

		// Every player will post the answer to this channel, the player details are in the object
		case playerobj := <-g.GamePro.AnswerChan:
			log.Info("received", "answer", playerobj.AnswerFromPlayer, "player", playerobj.Player)
			if !quizengine.ValidateAnswer(ctx, playerobj.AnswerFromPlayer) {
				log.Info("Wrong answer")
				continue
			}
			log.Info("Right answer")
			// TODO update the score in the DB
			// add a logic to give more points for faster answers
			playerobj.Player.Score++
			// TODO notify all the players that they one player has answered the question
			// TODO check if all answered right if so send it to that channel
			//

		// this is added to send the question immediately after every one has given the right answer
		case <-g.GamePro.areAllAnsweredRight:
			if err := broadCastQuestion(ctx, code, quizengine); err != nil {
				return err
			}

		case <-g.GamePro.IsGameEnded:
			if err := broadCastResult(ctx, code, quizengine); err != nil {
				return err
			}
		case <-ctx.Done():
			log.Info("context done")
			return nil
		}
	}
}

func broadCastQuestion(ctx context.Context, code string, q QuizEnginer) error {
	players, _ := GetAllPlayers(code)
	quizdata, err := q.ProduceQuestions(ctx, nil)
	if err != nil {
		return fmt.Errorf("error in play %v", err)
	}
	data := <-quizdata
	for _, player := range players {
		// fan out the quiz questions to players
		go func() {
			log.Debug("sending question to", "player", player.Player.Id)
			player.QuestionForPlayer <- data
		}()
	}
	return nil
}

func broadCastResult(_ context.Context, code string, q QuizEnginer) error {
	players, _ := GetAllPlayers(code)

	// TODO add result from DB
	for _, player := range players {
		// fan out the quiz questions to players
		go func() {
			player.Result <- nil
		}()
	}
	return nil
}
