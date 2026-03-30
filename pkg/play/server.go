package play

import (
	"context"
	"fmt"
	"io"
	log "log/slog"
	"time"

	pb "github.com/chinmayb/quizz-us/gen/go/api"
	"github.com/chinmayb/quizz-us/pkg/data"
	"github.com/chinmayb/quizz-us/pkg/gameengine/quiz"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func NewPlayServer(log *log.Logger) pb.GamesServer {
	s := &PlayServer{log: log}
	return s
}

type PlayServer struct {
	log *log.Logger
	pb.UnimplementedGamesServer
}

func initGame(_ context.Context, code string) {
	gameChan := make(chan quiz.GamePro)
	ansChan := make(chan quiz.PlayerObj)
	p := quiz.NewGameProcessor(gameChan, ansChan)
	p.Code = code

	// ADD it to the in memory registry
	if alreadyExists := quiz.AddGame(code, p); alreadyExists {
		log.Warn("game already exists", "gameID", code)
		return
	}

	gameCtx, cancel := context.WithCancel(context.Background())
	p.SetCancelFn(cancel)

	// should start only once
	go func() {
		if err := p.Process(gameCtx); err != nil {
			log.Error("game processor stopped", "err", err, "gameID", code)
		}
	}()
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
		Result:            make(chan *pb.GameSummary),
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

		code := in.GetCode()
		if code == "" {
			return fmt.Errorf("code not found %s", code)
		}
		if in.GetId() == "" {
			return fmt.Errorf("player ID not found")
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

			_, playerExists := quiz.GetPlayer(code, in.GetId())
			isRejoin := playerExists == nil

			if !isRejoin {
				// TODO only host can initialize the game
				initGame(stream.Context(), in.GetCode())
				playObj.Player.Id = in.GetId()
				playObj.Player.Name = in.GetName()

				_, playerCancel := context.WithCancel(stream.Context())
				playObj.SetCancelFunc(playerCancel)

				quiz.AddPlayerToRegistry(code, playObj)
			}

			go func() {
				if isRejoin {
					newQ := make(chan *data.QuizData)
					newResult := make(chan *pb.GameSummary)
					_, newCancel := context.WithCancel(stream.Context())
					lastQ, ok := quiz.RejoinPlayer(code, in.GetId(), newQ, newResult, newCancel)
					if !ok {
						log.Warn("rejoin failed", "player", in.GetId())
						return
					}
					playObj.QuestionForPlayer = newQ
					playObj.Result = newResult

					if lastQ != nil {
						out := &pb.GamePlay{
							Id:   in.GetId(),
							Code: code,
							Cmd: &pb.GamePlay_Command{Command: &pb.GamePlayCommand{
								Id:           lastQ.Id,
								Question:     lastQ.Question,
								QuestionTime: timestamppb.Now(),
							}},
						}
						if err := stream.Send(out); err != nil {
							log.Error("error sending catch-up question", "err", err)
							return
						}
					}
				}

				for {
					select {
					case quizQuestion := <-playObj.QuestionForPlayer:
						log.Info("Sending question to player", "player", playObj.Player.Id)
						out := &pb.GamePlay{
							Id:   playObj.Player.Id,
							Code: code,
							Cmd: &pb.GamePlay_Command{Command: &pb.GamePlayCommand{
								Id:           quizQuestion.Id,
								Question:     quizQuestion.Question,
								QuestionTime: timestamppb.Now(),
								// CorrectAnswer intentionally omitted — only exposed in results/summary phase
							}},
						}
						if err := stream.Send(out); err != nil {
							log.Error("error while sending question", "err", err)
							return
						}
					case result := <-playObj.Result:
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
						log.Info("player exited", "ID", in.GetId())
						quiz.DisconnectPlayer(code, playObj.Player.Id)

						players, err := quiz.GetAllPlayers(code)
						if err != nil {
							return
						}
						log.Info("game registry", "players", players)
						activeCount := 0
						for _, pl := range players {
							if pl.Player.Status != pb.PlayerStatus_DISCONNECTED {
								activeCount++
							}
						}
						if activeCount == 0 {
							log.Info("no active players left, removing the game")
							quiz.RemoveGame(code)
						}
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
