package play

import (
	"sync"

	pb "github.com/chinmayb/brainiac-brawl/gen/go/api"
	"github.com/chinmayb/brainiac-brawl/pkg/data"
	"github.com/chinmayb/brainiac-brawl/pkg/gameengine/quiz"
)

type PlayerChan struct {
	Player            pb.Player
	QuestionForPlayer chan data.QuizData
}

type OnGoingGame struct {
	Wg *sync.WaitGroup

	// question
	Questions chan quiz.Question

	Answers chan quiz.Answer
	// map of players
	PlayersMap map[string]chan PlayerChan

	//
}

func BeginGame() {

	/*
		1) should send the question to all the players
		2) should recieve the answer from all the players
		3) validate the answer
		4) should send the answer to
	*/
	for {

	}

}

func broadCastQuestion(players []chan PlayerChan, quizData data.QuizData) {
	wg := &sync.WaitGroup{}
	for _, ch := range players {
		wg.Add(1)
		go func(cha chan PlayerChan) {
			playerC := <-cha
			playerC.QuestionForPlayer <- quizData
		}(ch)
	}
}
