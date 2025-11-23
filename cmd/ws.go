package cmd

import (
	"context"
	"encoding/json"
	"errors"
	log "log/slog"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"

	pb "github.com/chinmayb/quizz-us/gen/go/api"
	"google.golang.org/protobuf/encoding/protojson"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		switch origin {
		case "http://localhost:8090", // dev frontend
			"http://localhost:8080", // server port
			"http://127.0.0.1:8080", // server IP
			"https://quizz.us":      // production frontend
			return true
		case "": // Allow file:// origins (when opening HTML files directly)
			return true
		default:
			// For development, allow localhost and 127.0.0.1 with any port
			if r.Host == "127.0.0.1:8080" || r.Host == "localhost:8080" {
				return true
			}
			return false
		}
	},
}

// WSHandler is a websocket handler, that returns the mux object for handling web socket request
// this interacts with the grpc stream backend to return the question & answer from the user
func WSHandler(ctx context.Context, log log.Logger, client pb.GamesClient) *http.ServeMux {
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/playws", func(w http.ResponseWriter, r *http.Request) {
		// Upgrade HTTP connection to WebSocket
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error("Failed to upgrade WebSocket", "err", err)
			return
		}
		defer func() {
			if err := ws.Close(); err != nil {
				log.Error("failed to close websocket", "reason", err)
			}
		}()

		// Start Play when the user joins the game & UI hits the websocket
		stream, err := client.Play(ctx)
		if err != nil {
			log.Info("Failed to initiate gRPC stream: %v", "err", err)
			return
		}
		done := make(chan struct{})

		// Goroutine: user -> WebSocket -> game grpc stream
		go func() {
			defer close(done)
			for {
				// read messages from web socket
				_, data, err := ws.ReadMessage()
				if err != nil {
					log.Error("WebSocket read error", "reason", err)
					if closeErr := stream.CloseSend(); closeErr != nil {
						log.Error("failed to close grpc stream", "reason", closeErr)
					}
					return
				}

				gamePlay, err := parseWebSocketMessage(data)
				if err != nil {
					log.Error("invalid websocket payload", "reason", err)
					if writeErr := writeWebSocketError(ws, err.Error()); writeErr != nil {
						log.Error("failed to send error to websocket client", "reason", writeErr)
						if closeErr := stream.CloseSend(); closeErr != nil {
							log.Error("failed to close grpc stream", "reason", closeErr)
						}
						return
					}
					continue
				}

				if err := stream.Send(gamePlay); err != nil {
					log.Error("failed to send message to grpc server", "reason", err)
					return
				}
			}
		}()

		// Goroutine: game grpc stream -> WebSocket -> user
		go func() {
			for {
				resp, err := stream.Recv()
				if err != nil {
					if writeErr := ws.WriteMessage(websocket.CloseMessage, []byte("gRPC stream closed")); writeErr != nil {
						log.Error("WebSocket close message error", "reason", writeErr)
					}
					return
				}
				payload, err := buildWebSocketPayload(resp)
				if err != nil {
					log.Error("failed to marshal response payload", "reason", err)
					if writeErr := writeWebSocketError(ws, "internal server error"); writeErr != nil {
						log.Error("failed to notify websocket client about marshal error", "reason", writeErr)
						return
					}
					continue
				}

				err = ws.WriteMessage(websocket.TextMessage, payload)
				if err != nil {
					log.Error("WebSocket write error", "reason", err)
					return
				}
			}
		}()

		<-done
	})
	return httpMux
}

type websocketIncomingMessage struct {
	ID      string                    `json:"id"`
	Code    string                    `json:"code"`
	Name    string                    `json:"name,omitempty"`
	Action  string                    `json:"action,omitempty"`
	Command *websocketIncomingCommand `json:"command,omitempty"`
}

type websocketIncomingCommand struct {
	ID            string `json:"id"`
	PlayerAnswer  string `json:"player_answer,omitempty"`
	Question      string `json:"question,omitempty"`
	CorrectAnswer string `json:"correct_answer,omitempty"`
}

type websocketOutgoingMessage struct {
	ID      string          `json:"id,omitempty"`
	Code    string          `json:"code,omitempty"`
	Action  string          `json:"action,omitempty"`
	Command json.RawMessage `json:"command,omitempty"`
	Summary json.RawMessage `json:"summary,omitempty"`
	Error   *websocketError `json:"error,omitempty"`
}

type websocketError struct {
	Message string `json:"message"`
}

var protoMarshalOptions = protojson.MarshalOptions{
	UseProtoNames: true,
}

func parseWebSocketMessage(data []byte) (*pb.GamePlay, error) {
	var msg websocketIncomingMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	if msg.ID == "" {
		return nil, errors.New("player id is required")
	}
	if msg.Code == "" {
		return nil, errors.New("game code is required")
	}

	gamePlay := &pb.GamePlay{
		Id:   msg.ID,
		Code: msg.Code,
		Name: msg.Name,
	}

	if msg.Command != nil {
		gamePlay.Cmd = &pb.GamePlay_Command{Command: &pb.GamePlayCommand{
			Id:            msg.Command.ID,
			PlayerAnswer:  msg.Command.PlayerAnswer,
			Question:      msg.Command.Question,
			CorrectAnswer: msg.Command.CorrectAnswer,
		}}
		return gamePlay, nil
	}

	if msg.Action == "" {
		return nil, errors.New("action or command is required")
	}

	action, err := actionFromString(msg.Action)
	if err != nil {
		return nil, err
	}

	gamePlay.Cmd = &pb.GamePlay_Action{Action: action}

	return gamePlay, nil
}

func buildWebSocketPayload(msg *pb.GamePlay) ([]byte, error) {
	if msg == nil {
		return nil, errors.New("gameplay response is nil")
	}

	out := websocketOutgoingMessage{
		ID:   msg.GetId(),
		Code: msg.GetCode(),
	}

	switch cmd := msg.GetCmd().(type) {
	case *pb.GamePlay_Action:
		out.Action = actionToString(cmd.Action)
	case *pb.GamePlay_Command:
		if cmd.Command != nil {
			raw, err := protoMarshalOptions.Marshal(cmd.Command)
			if err != nil {
				return nil, err
			}
			out.Command = raw
		}
	case *pb.GamePlay_Summary:
		if cmd.Summary != nil {
			raw, err := protoMarshalOptions.Marshal(cmd.Summary)
			if err != nil {
				return nil, err
			}
			out.Summary = raw
		}
	default:
		// no-op for unknown or unset command types
	}

	return json.Marshal(out)
}

func actionFromString(action string) (pb.GamePlayAction, error) {
	normalized := strings.ToUpper(strings.TrimSpace(action))
	switch normalized {
	case "JOIN":
		return pb.GamePlayAction_JOIN, nil
	case "HEARTBEAT":
		return pb.GamePlayAction_HEARTBEAT, nil
	case "BEGIN":
		return pb.GamePlayAction_BEGIN, nil
	case "END":
		return pb.GamePlayAction_END, nil
	case "UNKNOWN", "":
		return pb.GamePlayAction_UNKNOWN, nil
	default:
		return pb.GamePlayAction_UNKNOWN, errors.New("unsupported action: " + action)
	}
}

func actionToString(action pb.GamePlayAction) string {
	if name, ok := pb.GamePlayAction_name[int32(action)]; ok {
		return name
	}
	return "UNKNOWN"
}

func writeWebSocketError(ws *websocket.Conn, message string) error {
	payload, err := json.Marshal(websocketOutgoingMessage{
		Error: &websocketError{Message: message},
	})
	if err != nil {
		return err
	}

	return ws.WriteMessage(websocket.TextMessage, payload)
}
