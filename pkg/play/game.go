package play

import (
	"sync"

	pb "github.com/chinmayb/brainiac-brawl/gen/go/api"
)

type PlayerChan struct {
	Player pb.Player
}

type Question struct {
	quest string
}

type Answer struct {
	ans string
}

type OnGoingGame struct {
	Wg *sync.WaitGroup

	Questions chan Question

	Answers chan Answer
	// map of players
	PlayersMap map[string]chan PlayerChan

	//
}

func BeginGame() {

	for {
		select {}
	}

}
