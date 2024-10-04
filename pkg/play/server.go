package play

import (
	"context"
	"fmt"
	"io"
	log "log/slog"
	"time"

	pb "github.com/chinmayb/brainiac-brawl/gen/go/api"
	"github.com/chinmayb/brainiac-brawl/pkg/data"
	"github.com/chinmayb/brainiac-brawl/pkg/gameengine/quiz"
)

func NewPlayServer(log *log.Logger) pb.GamesServer {
	s := &PlayServer{log: log}
	return s
}

type PlayServer struct {
	log *log.Logger
	pb.UnimplementedGamesServer
}

func initGame(ctx context.Context, code string) {
	log.Info("initializing game: ", "code", code)
	gameChan := make(chan quiz.GamePro)
	ansChan := make(chan quiz.PlayerObj)
	p := quiz.NewGameProcessor(gameChan, ansChan)

	// ADD it to the in memory registry
	quiz.AddGame(code, p)

	// should start only once
	go p.Process(ctx)
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
		// in.GetCode()

		code := in.GetCode()
		if code == "" {
			return fmt.Errorf("code not found %s", code)
		}
		if in.GetId() == "" {
			return fmt.Errorf("player ID not found")
		}
		log := p.log.With("gameID", code)

		// TODO check if client is exited if n heartbeat missed
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
			log.Info("Player joined", "player", in.GetId())
			initGame(stream.Context(), in.GetCode())
			playObj.Player.Id = in.GetId()

			quiz.AddPlayerToRegistry(code, playObj)

			pl, err := quiz.GetPlayer(code, in.GetId())
			if err != nil {
				return err
			}
			// waiting for the question/result & send it to player
			// should happen only once when the player joins
			go func() {
				for {
					select {
					case quizQuestion := <-pl.QuestionForPlayer:
						log.Info("Sending question to player")
						out := &pb.GamePlay{
							Cmd: &pb.GamePlay_Command{Command: &pb.GamePlayCommand{
								Question:      quizQuestion.Question,
								CorrectAnswer: quizQuestion.Answer,
							},
							}}
						if err := stream.Send(out); err != nil {
							log.Error("error sending question", err)
							return
						}
					case result := <-pl.Result:
						out := &pb.GamePlay{
							Cmd: &pb.GamePlay_Summary{Summary: result},
						}
						if err := stream.Send(out); err != nil {
							return
						}
					// This is for heartbeat
					case <-ticker.C:
						log.Debug("Sending keepalive")
						out := &pb.GamePlay{
							Cmd: &pb.GamePlay_Summary{Summary: &pb.GameSummary{Status: pb.GamePlayStatus_NOT_STARTED}},
						}
						if err := stream.Send(out); err != nil {
							return
						}
					case <-stream.Context().Done():
						log.Info("client exiting")
						return
					}
				}
			}()
		}

		// check the action & see if its from the hosted player
		if in.GetAction() == pb.GamePlayAction_BEGIN {
			// update db with game play status
			// add begin logic
			p, ok := quiz.GetGame(code)
			if !ok {
				return fmt.Errorf("game ID not found")
			}
			gp := quiz.GamePro{Code: code}
			p.GamePro.BeginGame <- gp
			log.Info("sent")
		}

		if in.GetCommand().GetPlayerAnswer() != "" {
			// send the answer to processor queue

			p := quiz.PlayerObj{
				Player:            &pb.Player{Id: in.GetCommand().Id},
				QuestionForPlayer: nil,
				AnswerFromPlayer: &data.QuizData{
					Answer: in.GetCommand().GetPlayerAnswer(),
				},
			}
			log.Info("Received answer from player", "answer", p.AnswerFromPlayer)
			proc, ok := quiz.GetGame(code)
			if !ok {
				return fmt.Errorf("game ID not found")
			}
			fmt.Printf("Processor in sender %#v", proc)

			proc.GamePro.AnswerChan <- p
			log.Info("sent")
		}
	}
}
