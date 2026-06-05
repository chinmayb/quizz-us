# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run backend (starts gRPC on :9090 + HTTP/WS gateway on :8080)
make run-backend

# Run frontend dev server (serves on :3001)
make run-frontend

# Run all tests
make test

# Run a single test package
go test -v ./pkg/gameengine/quiz/...

# Format Go code and proto files
make fmt

# Regenerate protobuf/gRPC Go code from api/quizz-us.proto
make buf

# Tidy and vendor dependencies
make vendor

# Run the Go CLI client (for testing without the frontend)
make sample-client

# Run with Docker Compose (frontend at :3001, backend at :8080)
docker-compose up
```

## Architecture

This is a real-time multiplayer quiz game. The backend is Go; the frontend is vanilla Vue 3 (no build step, loaded from CDN via `<script>` tag).

### Request path

Browser → WebSocket (`/playws`) → gRPC bidirectional stream (`Play` RPC) → game processor goroutine

1. **Frontend** (`frontend/index-modular.html`, `frontend/components/`) — Vue 3 SPA that connects to the backend WebSocket. The host joins first, which initialises a game, then starts it with a BEGIN action. Players join via a game code.

2. **WebSocket bridge** (`cmd/ws.go`) — upgrades HTTP to WebSocket, then proxies JSON ↔ proto between the browser and the gRPC stream. JSON wire format is `websocketIncomingMessage` / `websocketOutgoingMessage`; these map to the `GamePlay` protobuf one-of.

3. **gRPC server** (`pkg/play/server.go`) — implements `Games.Play`, a bidirectional streaming RPC. On JOIN it calls `initGame`, adds the player to the in-memory registry, and spawns a goroutine that fans out questions and results to the player's channels. On BEGIN it signals the game processor.

4. **Game engine** (`pkg/gameengine/quiz/`) — two layers:
   - `processor.go`: `Game` / `GameRegistry` — in-memory registry of live games (protected by `sync.RWMutex`). Each game has a `gameProcessor` with channels for `BeginGame`, `AnswerChan`, and `areAllAnsweredRight`. `Game.Process` is the event loop run in a goroutine.
   - `engine.go`: `QuizEnginer` interface — picks a random question from `QuizDataRefined` and validates answers (case-insensitive string match).

5. **Quiz data** (`pkg/data/importer.go`, `quiz-data.yaml`) — YAML file parsed at startup into two maps: `QuizDataRefined` (id → QuizData) and `QuizDataByTag` (tag → []id). Questions are served from memory; no database.

6. **Protobuf API** (`api/quizz-us.proto`, generated into `gen/go/api/`) — defines `GamePlay`, `GamePlayCommand`, `GameSummary`, `Player`, and enums for `GamePlayAction`/`GamePlayStatus`. Edit the `.proto` then run `make buf` to regenerate.

7. **CLI** (`cmd/`) — Cobra CLI with `serve` (starts all servers) and `client` (test client) subcommands. Flags: `--port` (HTTP, default 8080), `--grpc-port` (gRPC, default 9090), `--log-level`.

### Key invariants

- `GameRegistry` is the single source of truth for live games; all access goes through the helpers in `processor.go` with the registry mutex.
- Each player has two channels: `QuestionForPlayer` and `Result`, written by the broadcaster goroutines in the processor and read by the per-player goroutine in `server.go`.
- The WebSocket bridge spawns two goroutines per connection (read loop, write loop) and uses a `done` channel to synchronise teardown when the read loop exits.
- `quiz-data.yaml` must be present in the working directory when the server starts; `serve.go` exits with code 1 if it cannot be parsed.
