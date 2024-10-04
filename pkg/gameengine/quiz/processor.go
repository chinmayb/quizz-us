package quiz

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	pb "github.com/chinmayb/brainiac-brawl/gen/go/api"
	"github.com/chinmayb/brainiac-brawl/pkg/data"
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

func AddPlayerToRegistry(gameID string, playerObj *PlayerObj) {
	if playerObj == nil {
		return
	}
	GameRegistry.mu.Lock()
	defer GameRegistry.mu.Unlock()

	playermap := make(map[string]*PlayerObj)

	playermap[playerObj.Player.Id] = playerObj
	if _, ok := GameRegistry.games[gameID]; !ok {
		return
	}
	// TODO introduce
	GameRegistry.games[gameID].players = playermap
}

func AddGame(gameID string, processor *Game) {
	GameRegistry.mu.Lock()
	defer GameRegistry.mu.Unlock()

	if processor == nil {
		return
	}

	// nothing to do if already exists
	if _, ok := GameRegistry.games[gameID]; ok {
		return
	}

	GameRegistry.games[gameID] = processor
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

	for {
		select {
		// TODO crashes needs to be handled from a reconciler if a game is running & stuck for a while
		case gameObj := <-g.GamePro.BeginGame:
			// check code and verify
			if p, ok := GetGame(gameObj.Code); !ok {
				return fmt.Errorf("no playing registry found")
			} else {
				fmt.Printf("Processor %#v", p)
			}
			g.Play(ctx, gameObj.Code)

		case <-ctx.Done():
			return nil
		}
	}
}

// Play
func (g *Game) Play(ctx context.Context, code string) error {
	// read it from db
	ticker := time.NewTicker(time.Second * 30)
	quizengine := NewQuizEnginer()
	log.Printf("begining game")
	if err := broadCastQuestion(ctx, code, quizengine); err != nil {
		return err
	}
	for {
		select {
		case <-ticker.C:
			log.Printf("sending the next question")
			if err := broadCastQuestion(ctx, code, quizengine); err != nil {
				return err
			}

		// Every player will post the answer to this channel, the player details are in the object
		case playerobj := <-g.GamePro.AnswerChan:
			log.Printf("received answer from %s", playerobj.Player.Name)
			if !quizengine.ValidateAnswer(ctx, playerobj.AnswerFromPlayer) {
				log.Printf("wrong answer")
				continue
			}
			// TODO update the score in the DB
			playerobj.Player.Score++
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
			log.Printf("context done")
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
			player.QuestionForPlayer <- data
		}()
	}
	return nil
}

func broadCastResult(ctx context.Context, code string, q QuizEnginer) error {
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
