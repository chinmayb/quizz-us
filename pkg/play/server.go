package play

import (
	"io"
	log "log/slog"

	pb "github.com/chinmayb/brainiac-brawl/gen/go/api"
)

type PlayServer struct {
	pb.BrainiacBrawlServer
}

func NewPlayServer() *PlayServer {
	s := &PlayServer{}
	return s
}

func (p *PlayServer) Play(stream pb.BrainiacBrawl_PlayServer) error {

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
		// in.GetCode()

		// check the action & see if its from the hosted player
		if in.GetAction() == pb.GamePlayAction_BEGIN {
			// update db with game play status
			BeginGame()
		}
		if err := stream.Send(in); err != nil {
			return err
		}
	}
}
