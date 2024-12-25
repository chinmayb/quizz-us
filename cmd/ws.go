package cmd

import (
	"context"
	log "log/slog"
	"net/http"

	"github.com/gorilla/websocket"

	pb "github.com/chinmayb/brainiac-brawl/gen/go/api"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		switch origin {
		case "http://localhost:8090", // dev frontend
			"https://brainiac-brawl.com": // production frontend
			return true
		default:
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
		defer ws.Close()

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
				// FIXME
				_, _, err := ws.ReadMessage()
				if err != nil {
					log.Error("WebSocket: %v", "read error", err)
					stream.CloseSend()
					return
				}
				err = stream.Send(nil)
				if err != nil {
					log.Info("failed to send to grpc server", "reason", err)
					return
				}
			}
		}()

		// Goroutine: game grpc stream -> WebSocket -> user
		go func() {
			for {
				resp, err := stream.Recv()
				if err != nil {
					log.Error("gRPC receive error: %v", "", err)
					ws.WriteMessage(websocket.CloseMessage, []byte("gRPC stream closed"))
					return
				}
				// write to websocket
				// FIXME
				err = ws.WriteMessage(websocket.TextMessage, []byte(resp.String()))
				if err != nil {
					log.Error("WebSocket write error: %v", "", err)
					return
				}
			}
		}()

		<-done
	})
	return httpMux
}
