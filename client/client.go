package client

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"

	pb "github.com/chinmayb/brainiac-brawl/gen/go/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Play
// Needs refactoring
func Play(ctx context.Context, opts ...grpc.CallOption) error {

	conn, err := grpc.NewClient("localhost:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	gc := pb.NewGamesClient(conn)
	stream, err := gc.Play(ctx)
	if err != nil {
		log.Println("err ", err)
		return err
	}
	done := make(chan struct{})
	playerID := fmt.Sprintf("%d", rand.Int31n(100))
	var (
		// use channel to send question id to the player
		// questionIdChan = make(chan string)
		isStarted  bool
		questionId string
	)
	code := "123"
	go func() {
		for {
			in, err := stream.Recv()
			if err != nil {
				log.Fatalf("Failed to receive message: %v", err)
				close(done)
				return
			}
			if in.GetCmd() == nil {
				// send heart beat
				fmt.Print("empty from server")
				continue
			}
			if in.GetSummary().GetStatus() == pb.GamePlayStatus_NOT_STARTED {
				fmt.Println("Received keepalive from server")
				out := &pb.GamePlay{Id: playerID, Code: code, Cmd: &pb.GamePlay_Action{Action: pb.GamePlayAction_HEARTBEAT}}
				stream.Send(out)
				continue
			}
			isStarted = true
			questionId = in.GetCommand().GetId()
			fmt.Println("Received question: ", in.GetCommand().GetQuestion(), in.GetCommand().GetId())
		}
	}()
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		// send init message to join
		out := &pb.GamePlay{Id: playerID, Code: code, Cmd: &pb.GamePlay_Action{Action: pb.GamePlayAction_JOIN}}
		if err := stream.Send(out); err != nil {
			log.Println("error while sending action", err)
			return
		}
		for {
			fmt.Println("Enter command: ")
			var message string
			for scanner.Scan() {
				message = scanner.Text()
				if message == "\n" {
					continue
				}
				break
			}
			if err := scanner.Err(); err != nil {
				fmt.Fprintln(os.Stderr, "reading standard input:", err)
			}
			switch message {
			// host says "begin"
			case "begin":
				fmt.Println("begining game")
				out = &pb.GamePlay{Id: playerID, Code: code, Cmd: &pb.GamePlay_Action{Action: pb.GamePlayAction_BEGIN}}
				if err = stream.Send(out); err != nil {
					log.Println("error while sending action", err)
					return
				}
				continue
			// player says `I quit!`
			case "quit":
				_ = stream.CloseSend() // Close the sending side of the stream
				return
			default:
				if !isStarted {
					fmt.Println("Game not started yet")
					continue
				}
				if message == "" {
					continue
				}
				// Do not accept answer to the same question if the answer was correct
				fmt.Println("sending answer ", message)
				out = &pb.GamePlay{Id: playerID, Code: code, Cmd: &pb.GamePlay_Command{Command: &pb.GamePlayCommand{Id: questionId, PlayerAnswer: message}}}
				if err = stream.Send(out); err != nil {
					log.Println("error while sending answer", err)
					return
				}
			}
		}
	}()
	<-done
	return nil
}
