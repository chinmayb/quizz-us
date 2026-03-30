# Player Rejoin Feature

## TL;DR

> **Quick Summary**: When a player's stream drops and they reconnect with the same player ID + game code, they are restored into the running game with their score intact and immediately receive the current question.
>
> **Deliverables**:
> - `PlayerObj` gains a per-player cancel context so old goroutines can be torn down on rejoin
> - `PlayerStatus.DISCONNECTED` is set on stream close (instead of deleting the registry entry)
> - `Game` struct tracks the last-broadcast question for catch-up delivery
> - `broadCastQuestion`/`broadCastResult` skip DISCONNECTED players (prevents goroutine leaks)
> - JOIN handler detects existing player ID → branches into rejoin or kick-and-rejoin flow
> - Two pre-existing bugs fixed as prerequisites: nil `Result` channel and broken score tracking
>
> **Estimated Effort**: Medium
> **Parallel Execution**: YES — 3 waves
> **Critical Path**: Task 1 → Task 2 → Task 3 → Task 4 → Task 5 → Task 6

---

## Context

### Original Request
"User exits and joins back with the same gameID, she should be put back into the game."

### Interview Summary
**Key Discussions**:
- **Score on rejoin**: Keep score intact, resume mid-game
- **Slot hold duration**: Hold slot until game ends — no timeout for v1
- **Catch-up**: Push current active question immediately on reconnect
- **Duplicate connection**: Kick old stream, accept new (last-write wins)

**Research Findings**:
- `PlayerStatus.DISCONNECTED` already defined in proto (unused) — use it, no proto changes needed
- `Result chan *pb.GameSummary` is NEVER allocated — silent goroutine leak when `broadCastResult` fires; fix alongside this work
- Score tracking is broken: answer handler creates throwaway `PlayerObj` with `&pb.Player{Id: ...}`, increments score on that copy, discards it — registry `PlayerObj` always has score=0
- `broadCastQuestion` uses unbuffered channels and fans out to ALL players — if a DISCONNECTED player stays in registry without filtering, the fan-out goroutine blocks forever (critical goroutine leak)
- `RemovePlayerFromRegistry` has a nil-dereference: `GameRegistry.games[gameID].players` panics if `gameID` doesn't exist

### Metis Review
**Identified Gaps (addressed)**:
- Fan-out goroutine leak: DISCONNECTED players MUST be filtered out of broadcast loops — squash Tasks 4+5 to prevent unsafe ordering
- Score tracking is currently broken — fix as prerequisite or "score intact" guarantee is vacuous
- Old goroutine race window on rejoin: per-player `context.CancelFunc` needed on `PlayerObj`
- `RemovePlayerFromRegistry` nil-dereference on missing game — add guard
- `initGame` called on every JOIN (idempotent but wasteful on rejoin) — rejoin path skips it

---

## Work Objectives

### Core Objective
Allow a player who disconnects (stream drops) to reconnect with the same ID and game code, be re-inserted into the running game with their score preserved, and immediately receive the current active question.

### Concrete Deliverables
- `pkg/gameengine/quiz/processor.go` — updated `PlayerObj`, `Game`, `RemovePlayerFromRegistry`, `broadCastQuestion`, `broadCastResult`, new `RejoinPlayer()` helper, `DisconnectPlayer()` helper
- `pkg/play/server.go` — updated `Play()` JOIN handler with rejoin detection, updated answer handler for correct score tracking, `Result` channel allocation
- `pkg/gameengine/quiz/processor_test.go` — new tests for all new behaviours

### Definition of Done
- [ ] `go test ./pkg/... -v` passes — all new tests GREEN
- [ ] `go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] A player that disconnects and reconnects with same ID+code re-enters game

### Must Have
- DISCONNECTED status set on stream close (not delete)
- Score preserved across disconnect/rejoin cycle
- Last question delivered immediately on rejoin
- Old goroutine cancelled on duplicate rejoin
- DISCONNECTED players skipped in broadcast loops
- `Result` channel allocated on player creation
- Nil-guard in `RemovePlayerFromRegistry`

### Must NOT Have (Guardrails)
- **NO** heartbeat timeout / eviction of DISCONNECTED players (out of scope v1)
- **NO** proto changes — do not add fields to `GamePlay`, `Player`, or any proto message
- **NO** new `GamePlayAction` enum value — rejoin reuses JOIN
- **NO** DB / persistence — all in-memory
- **NO** changes to END action behavior — intentional leave still fully deletes
- **NO** refactoring the global `GameRegistry` locking model beyond what this feature requires
- **NO** changes to WebSocket handler, transport layer, or any frontend code
- **NO** fixing unrelated TODOs (`areAllAnsweredRight`, host-only BEGIN, heartbeat counting)
- **NO** over-abstraction — no new interfaces, no new packages

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed.

### Test Decision
- **Infrastructure exists**: YES (`go test`, `pkg/gameengine/quiz/processor_test.go`)
- **Automated tests**: TDD — RED → GREEN → REFACTOR per task
- **Framework**: `go test` (stdlib)
- **Pattern**: Each task writes a failing test first, then implements to make it pass

### QA Policy
Every task includes agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.txt`.

- **Library/Unit**: `go test ./pkg/... -v -run TestName` — exact assertions, concrete data

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Prerequisite bug fixes — must land before anything else):
├── Task 1: Fix Result channel allocation + nil-guard RemovePlayerFromRegistry [quick]
└── Task 2: Fix score tracking — update registry PlayerObj on correct answer [quick]

Wave 2 (Infrastructure — Tasks 3+4 must land together):
├── Task 3: Add per-player context to PlayerObj + DisconnectPlayer() helper [unspecified-low]
└── Task 4: Add lastQuestion tracking to Game struct + update broadCastQuestion/Result [unspecified-low]

Wave 3 (Feature — depends on Wave 2):
└── Task 5: JOIN handler rejoin detection + RejoinPlayer() registry helper [unspecified-high]

Wave FINAL (after all tasks):
├── Task F1: Full test suite + build verification
└── Task F2: Scope fidelity check
```

**Critical ordering note**: Task 3 (DISCONNECTED status) and broadcast filtering MUST land together. Task 3 changes stream-close to set DISCONNECTED; if broadcast filtering isn't active, DISCONNECTED players in registry will cause goroutine leaks. They are therefore combined into a single atomic task.

### Dependency Matrix
- **Task 1**: None → blocks nothing specifically (prerequisite safety)
- **Task 2**: None → blocks Task 5 (score correctness needed before rejoin)
- **Task 3**: None → blocks Task 5 (per-player context needed for rejoin kick)
- **Task 4**: None → blocks Task 5 (lastQuestion needed for catch-up delivery)
- **Task 5**: Tasks 2, 3, 4 → blocks Final

### Agent Dispatch Summary
- **Wave 1**: 2 × `quick`
- **Wave 2**: 2 × `unspecified-low`
- **Wave 3**: 1 × `unspecified-high`
- **Final**: 1 × `unspecified-high`

---

## TODOs

- [x] 1. Fix Result channel allocation and nil-guard RemovePlayerFromRegistry

  **What to do**:
  - In `pkg/play/server.go`, find where `playObj` is created (lines 62-66). Add `Result: make(chan *pb.GameSummary)` to the `PlayerObj` literal so Result is never nil.
  - In `pkg/gameengine/quiz/processor.go`, `RemovePlayerFromRegistry` (lines 86-91): add a nil-guard so it doesn't panic if `gameID` doesn't exist in the map. Current: `delete(GameRegistry.games[gameID].players, playerID)`. Fix:
    ```go
    func RemovePlayerFromRegistry(gameID string, playerID string) {
        GameRegistry.mu.Lock()
        defer GameRegistry.mu.Unlock()
        game, ok := GameRegistry.games[gameID]
        if !ok {
            return
        }
        delete(game.players, playerID)
    }
    ```
  - Write unit test `TestResultChannelAllocated` in `processor_test.go`:
    - Create a `PlayerObj` the same way `server.go` does, assert `Result != nil`
  - Write unit test `TestRemovePlayerFromRegistry_MissingGame`:
    - Call `RemovePlayerFromRegistry("nonexistent", "p1")` — must not panic

  **Must NOT do**:
  - Do not change any other fields of PlayerObj
  - Do not change broadCastResult logic
  - Do not add buffering to the Result channel (unbuffered is correct — matches existing pattern)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (Wave 1, parallel with Task 2)
  - **Blocks**: Task 5 (safety prerequisite)
  - **Blocked By**: None

  **References**:
  - `pkg/play/server.go:59-66` — PlayerObj creation (where to add Result channel allocation)
  - `pkg/gameengine/quiz/processor.go:86-91` — RemovePlayerFromRegistry current impl
  - `pkg/gameengine/quiz/processor.go:118-127` — PlayerObj struct definition
  - `pkg/gameengine/quiz/processor.go:245-263` — broadCastResult (uses Result channel)
  - `pkg/gameengine/quiz/processor_test.go` — existing test patterns to follow

  **Acceptance Criteria**:
  - [ ] `go build ./...` exits 0
  - [ ] `go test ./pkg/gameengine/quiz/ -v -run TestResultChannelAllocated` PASS
  - [ ] `go test ./pkg/gameengine/quiz/ -v -run TestRemovePlayerFromRegistry_MissingGame` PASS

  **QA Scenarios**:
  ```
  Scenario: Result channel is allocated
    Tool: Bash (go test)
    Steps:
      1. go test ./pkg/gameengine/quiz/ -v -run TestResultChannelAllocated
    Expected Result: PASS, "--- PASS: TestResultChannelAllocated"
    Evidence: .sisyphus/evidence/task-1-result-channel.txt

  Scenario: RemovePlayerFromRegistry does not panic on missing game
    Tool: Bash (go test)
    Steps:
      1. go test ./pkg/gameengine/quiz/ -v -run TestRemovePlayerFromRegistry_MissingGame
    Expected Result: PASS (no panic)
    Evidence: .sisyphus/evidence/task-1-remove-missing-game.txt
  ```

  **Commit**: YES
  - Message: `fix(game): allocate Result channel and guard RemovePlayerFromRegistry against missing game`
  - Files: `pkg/play/server.go`, `pkg/gameengine/quiz/processor.go`, `pkg/gameengine/quiz/processor_test.go`
  - Pre-commit: `go build ./... && go test ./pkg/...`

---

- [x] 2. Fix score tracking — update registry PlayerObj on correct answer

  **What to do**:
  - In `pkg/play/server.go` lines 186-201: the answer handler creates a throwaway `PlayerObj{Player: &pb.Player{Id: in.GetId()}, ...}`. The processor increments `.Score++` on this copy, which is immediately discarded. The registry entry always stays at score=0.
  - Fix: in the `AnswerChan` case inside `processor.go` `Play()` (lines 206-215), after `playerobj.Player.Score++`, look up the registry entry by `playerobj.Player.Id` and sync the score back. Better approach: look up the registry `PlayerObj` directly before sending to `AnswerChan`, or have the processor update the registry entry.
  - **Simplest correct fix** (minimal scope): In `processor.go` `Play()`, after `playerobj.Player.Score++` (line 215), call a new unexported helper `updatePlayerScore(g.Code, playerobj.Player.Id, playerobj.Player.Score)` that writes the score back to the registry entry:
    ```go
    func updatePlayerScore(gameID string, playerID string, score int32) {
        GameRegistry.mu.Lock()
        defer GameRegistry.mu.Unlock()
        game, ok := GameRegistry.games[gameID]
        if !ok {
            return
        }
        if p, ok := game.players[playerID]; ok {
            p.Player.Score = score
        }
    }
    ```
  - Write unit test `TestScoreTrackedInRegistry`:
    - Add a player with score=0 → call `updatePlayerScore(code, id, 3)` → `GetPlayer(code, id).Player.Score == 3`
  - Write unit test `TestScorePreservedThroughDisconnect` (will be exercised more fully in Task 3, but lay the foundation here):
    - Set score → verify it's on the registry object

  **Must NOT do**:
  - Do not change the `AnswerChan` message type or `PlayerObj` struct
  - Do not add DB calls — in-memory only
  - Do not refactor how answers flow through the channel

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (Wave 1, parallel with Task 1)
  - **Blocks**: Task 5 (score must be correct before rejoin is meaningful)
  - **Blocked By**: None

  **References**:
  - `pkg/play/server.go:185-201` — answer handler (sends throwaway PlayerObj to AnswerChan)
  - `pkg/gameengine/quiz/processor.go:197-215` — `Play()` AnswerChan case (where Score++ happens on wrong object)
  - `pkg/gameengine/quiz/processor.go:57-72` — `GetPlayer()` helper (pattern for registry lookup)
  - `pkg/gameengine/quiz/processor_test.go` — existing test patterns

  **Acceptance Criteria**:
  - [ ] `go test ./pkg/gameengine/quiz/ -v -run TestScoreTrackedInRegistry` PASS
  - [ ] `go build ./...` exits 0

  **QA Scenarios**:
  ```
  Scenario: Score is updated on registry PlayerObj after correct answer
    Tool: Bash (go test)
    Steps:
      1. go test ./pkg/gameengine/quiz/ -v -run TestScoreTrackedInRegistry
    Expected Result: PASS
    Evidence: .sisyphus/evidence/task-2-score-tracking.txt
  ```

  **Commit**: YES
  - Message: `fix(game): sync player score back to registry on correct answer`
  - Files: `pkg/gameengine/quiz/processor.go`, `pkg/gameengine/quiz/processor_test.go`
  - Pre-commit: `go build ./... && go test ./pkg/...`

---

- [x] 3. Add per-player context to PlayerObj, DisconnectPlayer() helper, and filter DISCONNECTED players from broadcasts

  **What to do**:

  **3a — PlayerObj struct** (`pkg/gameengine/quiz/processor.go`):
  Add `cancelCtx context.CancelFunc` to `PlayerObj`:
  ```go
  type PlayerObj struct {
      Player            *pb.Player
      QuestionForPlayer chan *data.QuizData
      Result            chan *pb.GameSummary
      AnswerFromPlayer  *data.QuizData
      cancelCtx         context.CancelFunc  // cancels this player's stream goroutine
  }
  ```

  **3b — DisconnectPlayer() registry helper** (`pkg/gameengine/quiz/processor.go`):
  Add new exported function:
  ```go
  // DisconnectPlayer marks a player as DISCONNECTED (keeps them in registry for rejoin).
  // Calls their cancel func if set, so the stream goroutine exits promptly.
  func DisconnectPlayer(gameID string, playerID string) {
      GameRegistry.mu.Lock()
      defer GameRegistry.mu.Unlock()
      game, ok := GameRegistry.games[gameID]
      if !ok {
          return
      }
      p, ok := game.players[playerID]
      if !ok {
          return
      }
      p.Player.Status = pb.PlayerStatus_DISCONNECTED
      if p.cancelCtx != nil {
          p.cancelCtx()
      }
  }
  ```

  **3c — Filter in broadCastQuestion** (`pkg/gameengine/quiz/processor.go`):
  In `broadCastQuestion` (lines 236-252), before the fan-out goroutine, skip DISCONNECTED players:
  ```go
  for _, player := range players {
      if player.Player.Status == pb.PlayerStatus_DISCONNECTED {
          continue
      }
      // existing fan-out goroutine...
  }
  ```
  Apply the same filter in `broadCastResult` (lines 255-263).

  **3d — Update stream-close path** (`pkg/play/server.go`):
  In the `stream.Context().Done()` case (lines 144-158): replace `quiz.RemovePlayerFromRegistry(code, playObj.Player.Id)` with `quiz.DisconnectPlayer(code, playObj.Player.Id)`. Keep the "if all players gone, remove game" check BUT change the count: count only players where `Status != DISCONNECTED`. If all remaining PLAYING players are gone, still remove the game.

  **3e — Pass cancel func on player creation** (`pkg/play/server.go`):
  When creating `playObj` (lines 62-66), the cancel func isn't known yet. After calling `quiz.AddPlayerToRegistry`, update the registry entry's `cancelCtx` to a fresh cancel function. Or: create the context at the point `playObj` is created:
  ```go
  _, cancel := context.WithCancel(context.Background())
  playObj := &quiz.PlayerObj{
      QuestionForPlayer: qForPlayChan,
      Player:            &pb.Player{},
      Result:            make(chan *pb.GameSummary),
      cancelCtx:         cancel,
  }
  ```
  The spawned goroutine exits via `stream.Context().Done()` already, so the per-player context is mainly used by `DisconnectPlayer` to signal the goroutine to exit on duplicate rejoin (Task 5).

  **Write unit tests** in `processor_test.go`:
  - `TestDisconnectPlayer_SetsStatus`: Add player → `DisconnectPlayer` → assert `Status == DISCONNECTED`
  - `TestDisconnectPlayer_CancelsContext`: Add player with cancel func → `DisconnectPlayer` → assert cancel was called
  - `TestBroadcastSkipsDisconnected`: 2 PLAYING + 1 DISCONNECTED player (buffered channels size 1) → broadCastQuestion → assert DISCONNECTED player's channel empty after broadcast
  - `TestDisconnectPlayer_MissingGame`: `DisconnectPlayer("none","p1")` — no panic
  - `TestStreamCloseDisconnectsPlayer`: simulate the `stream.Context().Done()` path → player status DISCONNECTED, still in registry

  **Must NOT do**:
  - Do not close the player's `QuestionForPlayer` or `Result` channels on disconnect — they will be reused on rejoin
  - Do not change END action behavior — it still calls `RemovePlayerFromRegistry` (full delete)
  - Do not add timeout or eviction logic

  **Recommended Agent Profile**:
  - **Category**: `unspecified-low`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (Wave 2, parallel with Task 4)
  - **Blocks**: Task 5
  - **Blocked By**: Task 1 (nil-guard prerequisite for safety)

  **References**:
  - `pkg/gameengine/quiz/processor.go:118-127` — PlayerObj struct
  - `pkg/gameengine/quiz/processor.go:236-252` — broadCastQuestion fan-out loop
  - `pkg/gameengine/quiz/processor.go:255-263` — broadCastResult fan-out loop
  - `pkg/gameengine/quiz/processor.go:86-91` — RemovePlayerFromRegistry (the function stream-close currently calls — will be replaced by DisconnectPlayer call from server.go)
  - `pkg/play/server.go:144-158` — stream.Context().Done() case (where to call DisconnectPlayer instead of RemovePlayerFromRegistry)
  - `pkg/play/server.go:62-66` — playObj creation (where to set cancelCtx)
  - `api/quizz-us.proto:60-64` — PlayerStatus enum (PLAYING=0, WAITING=1, DISCONNECTED=2)
  - `gen/go/api/quizz-us.pb.go` — generated `pb.PlayerStatus_DISCONNECTED` constant
  - `pkg/gameengine/quiz/processor_test.go` — existing test patterns

  **Acceptance Criteria**:
  - [ ] `go test ./pkg/gameengine/quiz/ -v -run TestDisconnectPlayer` PASS
  - [ ] `go test ./pkg/gameengine/quiz/ -v -run TestBroadcastSkipsDisconnected` PASS
  - [ ] `go build ./...` exits 0
  - [ ] `go vet ./...` exits 0

  **QA Scenarios**:
  ```
  Scenario: DisconnectPlayer sets DISCONNECTED status
    Tool: Bash (go test)
    Steps:
      1. go test ./pkg/gameengine/quiz/ -v -run TestDisconnectPlayer_SetsStatus
    Expected Result: PASS
    Evidence: .sisyphus/evidence/task-3-disconnect-status.txt

  Scenario: broadCastQuestion skips DISCONNECTED players (no goroutine leak)
    Tool: Bash (go test)
    Steps:
      1. go test ./pkg/gameengine/quiz/ -v -run TestBroadcastSkipsDisconnected
    Expected Result: PASS — only PLAYING players receive question
    Evidence: .sisyphus/evidence/task-3-broadcast-filter.txt
  ```

  **Commit**: YES
  - Message: `feat(game): add per-player cancel context and DisconnectPlayer helper; filter DISCONNECTED from broadcasts`
  - Files: `pkg/gameengine/quiz/processor.go`, `pkg/play/server.go`, `pkg/gameengine/quiz/processor_test.go`
  - Pre-commit: `go build ./... && go test ./pkg/...`

---

- [x] 4. Add lastQuestion tracking to Game struct

  **What to do**:

  **4a — Game struct** (`pkg/gameengine/quiz/processor.go`):
  Add `lastQuestion` and a protecting mutex to `Game`:
  ```go
  type Game struct {
      GamePro      gameProcessor
      players      PlayersMap
      Code         string
      cancelFn     context.CancelFunc
      lastQuestion *data.QuizData   // last question broadcast to players
      lastQMu      sync.RWMutex     // protects lastQuestion
  }
  ```

  **4b — Store in broadCastQuestion** (`pkg/gameengine/quiz/processor.go`):
  After `data := <-quizdata` (line 242), store it on the game:
  ```go
  if g, ok := GameRegistry.games[code]; ok {   // reuse existing GetGame pattern
      g.lastQMu.Lock()
      g.lastQuestion = data
      g.lastQMu.Unlock()
  }
  ```
  Note: `broadCastQuestion` doesn't have a reference to `*Game` — it receives `code string`. Use `GetGame(code)` to look it up (already exists at processor.go:37-43).

  **4c — GetLastQuestion() helper**:
  ```go
  func GetLastQuestion(gameID string) *data.QuizData {
      game, ok := GetGame(gameID)
      if !ok {
          return nil
      }
      game.lastQMu.RLock()
      defer game.lastQMu.RUnlock()
      return game.lastQuestion
  }
  ```

  **Write unit tests** in `processor_test.go`:
  - `TestLastQuestionTracking`: Call broadCastQuestion (or manually set lastQuestion) → `GetLastQuestion(code)` returns the question
  - `TestGetLastQuestion_NoGame`: `GetLastQuestion("nonexistent")` returns nil (no panic)

  **Must NOT do**:
  - Do not store question history — only the single last question
  - Do not change the QuizEngine or question production logic
  - Do not add question timestamps for this task (out of scope)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-low`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (Wave 2, parallel with Task 3)
  - **Blocks**: Task 5
  - **Blocked By**: Task 1

  **References**:
  - `pkg/gameengine/quiz/processor.go:148-157` — Game struct (where to add lastQuestion + mutex)
  - `pkg/gameengine/quiz/processor.go:236-252` — broadCastQuestion (where to store lastQuestion)
  - `pkg/gameengine/quiz/processor.go:37-43` — GetGame() helper (pattern to reuse for looking up game by code)
  - `pkg/data/importer.go` — QuizData struct (the type being stored)
  - `pkg/gameengine/quiz/processor_test.go` — existing test patterns

  **Acceptance Criteria**:
  - [ ] `go test ./pkg/gameengine/quiz/ -v -run TestLastQuestion` PASS
  - [ ] `go test ./pkg/gameengine/quiz/ -v -run TestGetLastQuestion_NoGame` PASS
  - [ ] `go build ./...` exits 0

  **QA Scenarios**:
  ```
  Scenario: Last question is stored after broadcast
    Tool: Bash (go test)
    Steps:
      1. go test ./pkg/gameengine/quiz/ -v -run TestLastQuestionTracking
    Expected Result: PASS — GetLastQuestion returns the question that was broadcast
    Evidence: .sisyphus/evidence/task-4-last-question.txt

  Scenario: GetLastQuestion returns nil for missing game (no panic)
    Tool: Bash (go test)
    Steps:
      1. go test ./pkg/gameengine/quiz/ -v -run TestGetLastQuestion_NoGame
    Expected Result: PASS
    Evidence: .sisyphus/evidence/task-4-last-question-nil.txt
  ```

  **Commit**: YES
  - Message: `feat(game): track last broadcast question on Game for catch-up on rejoin`
  - Files: `pkg/gameengine/quiz/processor.go`, `pkg/gameengine/quiz/processor_test.go`
  - Pre-commit: `go build ./... && go test ./pkg/...`

---

- [x] 5. JOIN handler rejoin detection + RejoinPlayer() registry helper

  **What to do**:

  **5a — RejoinPlayer() registry helper** (`pkg/gameengine/quiz/processor.go`):
  ```go
  // RejoinPlayer restores a DISCONNECTED (or PLAYING) player to active state.
  // Swaps in fresh channels, cancels any old goroutine via the existing cancelCtx,
  // sets status to PLAYING, and returns the current last question for catch-up.
  // Returns (lastQuestion, true) on success, (nil, false) if player not found.
  func RejoinPlayer(gameID string, playerID string, newQ chan *data.QuizData, newResult chan *pb.GameSummary, newCancel context.CancelFunc) (*data.QuizData, bool) {
      GameRegistry.mu.Lock()
      game, ok := GameRegistry.games[gameID]
      if !ok {
          GameRegistry.mu.Unlock()
          return nil, false
      }
      p, ok := game.players[playerID]
      if !ok {
          GameRegistry.mu.Unlock()
          return nil, false
      }
      // Cancel the old goroutine (if any — e.g. duplicate PLAYING player)
      if p.cancelCtx != nil {
          p.cancelCtx()
      }
      // Swap in fresh channels and cancel func
      p.QuestionForPlayer = newQ
      p.Result = newResult
      p.cancelCtx = newCancel
      p.Player.Status = pb.PlayerStatus_PLAYING
      GameRegistry.mu.Unlock()

      // Fetch last question (uses its own RLock)
      lastQ := GetLastQuestion(gameID)
      return lastQ, true
  }
  ```

  **5b — JOIN handler in server.go** (`pkg/play/server.go`):
  In the JOIN action block (lines 98-162), add a check BEFORE `initGame` and `AddPlayerToRegistry`:
  ```go
  if in.GetAction() == pb.GamePlayAction_JOIN {
      log.Info("Player joined", "player", in.GetId(), "name", in.GetName())

      // Check if this player already exists in the game (rejoin scenario)
      existingPlayer, exists := quiz.GetPlayer(code, in.GetId())
      isRejoin := exists && (existingPlayer.Player.Status == pb.PlayerStatus_DISCONNECTED ||
                             existingPlayer.Player.Status == pb.PlayerStatus_PLAYING)

      if !isRejoin {
          // New player: normal path
          initGame(stream.Context(), in.GetCode())
          playObj.Player.Id = in.GetId()
          playObj.Player.Name = in.GetName()
          quiz.AddPlayerToRegistry(code, playObj)
      }
      // If isRejoin: skip initGame + AddPlayerToRegistry, call RejoinPlayer below

      // Spawn the stream goroutine (same for new and rejoin)
      go func() {
          var lastQ *data.QuizData
          if isRejoin {
              // Rejoin: swap channels + get last question atomically
              newQ := make(chan *data.QuizData)
              newResult := make(chan *pb.GameSummary)
              _, newCancel := context.WithCancel(context.Background())
              var ok bool
              lastQ, ok = quiz.RejoinPlayer(code, in.GetId(), newQ, newResult, newCancel)
              if !ok {
                  log.Warn("rejoin failed — player not found after check", "player", in.GetId())
                  return
              }
              // Update local playObj references to point to the new channels
              playObj.QuestionForPlayer = newQ
              playObj.Result = newResult
          }

          // If rejoining and there is a current question, push it immediately
          if isRejoin && lastQ != nil {
              out := &pb.GamePlay{
                  Id:   in.GetId(),
                  Code: code,
                  Cmd: &pb.GamePlay_Command{Command: &pb.GamePlayCommand{
                      Id:           lastQ.Id,
                      Question:     lastQ.Question,
                      QuestionTime: timestamppb.Now(),
                      // CorrectAnswer intentionally omitted
                  }},
              }
              if err := stream.Send(out); err != nil {
                  log.Error("error sending catch-up question to rejoining player", "err", err)
                  return
              }
          }

          // Existing select loop (unchanged)
          for {
              select {
              case quizQuestion := <-playObj.QuestionForPlayer:
                  // ... existing code
              case result := <-playObj.Result:
                  // ... existing code
              case <-ticker.C:
                  // ... existing code
              case <-stream.Context().Done():
                  // ... existing code (DisconnectPlayer instead of RemovePlayer)
              }
          }
      }()
  }
  ```

  **Write unit tests** in `processor_test.go`:
  - `TestRejoinPlayer_DisconnectedPlayer`: Add DISCONNECTED player with score=5 → `RejoinPlayer` → assert status PLAYING, score=5, channels replaced, lastQ returned
  - `TestRejoinPlayer_PlayingPlayer_KicksOld`: Add PLAYING player with cancelCtx → `RejoinPlayer` → assert old cancel was called, channels replaced
  - `TestRejoinPlayer_NotFound`: `RejoinPlayer("none","p1",...)` returns (nil, false), no panic
  - `TestRejoinPlayer_ScorePreserved`: Score=7 before disconnect → rejoin → score still 7

  **Must NOT do**:
  - Do not change the ticker keepalive logic
  - Do not change the answer submission path
  - Do not add a new `GamePlayAction` value — reuse JOIN
  - Do not modify the END action handler
  - Do not close old channels — old goroutine exits via its context being cancelled

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO (Wave 3 — depends on Tasks 2, 3, 4)
  - **Blocks**: Final verification
  - **Blocked By**: Tasks 2, 3, 4

  **References**:
  - `pkg/play/server.go:95-162` — Full JOIN handler block (where rejoin detection + goroutine launch goes)
  - `pkg/play/server.go:111-124` — Question send logic (pattern to copy for catch-up send)
  - `pkg/gameengine/quiz/processor.go:57-72` — GetPlayer() (use to detect existing player on JOIN)
  - `pkg/gameengine/quiz/processor.go:88-103` — AddPlayerToRegistry (skip on rejoin)
  - `pkg/gameengine/quiz/processor.go:37-43` — GetGame() pattern
  - `pkg/gameengine/quiz/processor.go:148-157` — Game struct (lastQuestion field added in Task 4)
  - `pkg/play/server.go:26-47` — initGame() (skip on rejoin)
  - `pkg/play/server.go:141-158` — stream.Context().Done() case (uses DisconnectPlayer from Task 3)
  - `api/quizz-us.proto:75-81` — GamePlayAction enum (JOIN=1 — reuse, no new value)
  - `gen/go/api/quizz-us.pb.go` — pb.PlayerStatus_DISCONNECTED, pb.PlayerStatus_PLAYING constants

  **Acceptance Criteria**:
  - [ ] `go test ./pkg/gameengine/quiz/ -v -run TestRejoinPlayer` PASS (all subtests)
  - [ ] `go test ./pkg/... -v` all PASS (no regressions)
  - [ ] `go build ./...` exits 0
  - [ ] `go vet ./...` exits 0

  **QA Scenarios**:
  ```
  Scenario: Disconnected player rejoins and gets current question
    Tool: Bash (go test)
    Steps:
      1. go test ./pkg/gameengine/quiz/ -v -run TestRejoinPlayer_DisconnectedPlayer
    Expected Result: PASS — status PLAYING, lastQ returned, channels replaced, score preserved
    Evidence: .sisyphus/evidence/task-5-rejoin-disconnected.txt

  Scenario: Duplicate PLAYING player — old goroutine cancelled on rejoin
    Tool: Bash (go test)
    Steps:
      1. go test ./pkg/gameengine/quiz/ -v -run TestRejoinPlayer_PlayingPlayer_KicksOld
    Expected Result: PASS — old cancel func was called, new channels in place
    Evidence: .sisyphus/evidence/task-5-rejoin-kick-old.txt

  Scenario: Score preserved across disconnect/rejoin
    Tool: Bash (go test)
    Steps:
      1. go test ./pkg/gameengine/quiz/ -v -run TestRejoinPlayer_ScorePreserved
    Expected Result: PASS — score unchanged after rejoin
    Evidence: .sisyphus/evidence/task-5-score-preserved.txt

  Scenario: Full test suite — no regressions
    Tool: Bash (go test)
    Steps:
      1. go test ./pkg/... -v
    Expected Result: All tests PASS
    Evidence: .sisyphus/evidence/task-5-full-suite.txt
  ```

  **Commit**: YES
  - Message: `feat(game): player rejoin — restore DISCONNECTED player with score and current question`
  - Files: `pkg/gameengine/quiz/processor.go`, `pkg/play/server.go`, `pkg/gameengine/quiz/processor_test.go`
  - Pre-commit: `go build ./... && go test ./pkg/...`

---

## Final Verification Wave

> 2 review agents run in PARALLEL. Both must APPROVE before work is considered done.

- [x] F1. **Full suite + build verification** — `unspecified-high`
  Run `go test ./pkg/... -v` and capture full output. Run `go build ./...`. Run `go vet ./...`.
  Verify: all new tests present and PASS (TestResultChannelAllocated, TestRemovePlayerFromRegistry_MissingGame, TestScoreTrackedInRegistry, TestDisconnectPlayer_SetsStatus, TestBroadcastSkipsDisconnected, TestLastQuestionTracking, TestRejoinPlayer_*).
  Output: `Build [PASS/FAIL] | Vet [PASS/FAIL] | Tests [N pass / N fail] | VERDICT`

- [x] F2. **Scope fidelity check** — `deep`
  For each task, read "What to do" and compare against actual git diff. Verify:
  - No proto files were modified
  - No frontend files were modified
  - END action still calls `RemovePlayerFromRegistry` (not DisconnectPlayer)
  - `initGame` is skipped on rejoin path
  - No new `GamePlayAction` enum value added
  - No timeout/eviction logic added
  Output: `Tasks [N/N compliant] | Proto untouched [YES/NO] | Frontend untouched [YES/NO] | VERDICT`

---

## Commit Strategy

| # | Message | Files |
|---|---------|-------|
| 1 | `fix(game): allocate Result channel and guard RemovePlayerFromRegistry against missing game` | server.go, processor.go, processor_test.go |
| 2 | `fix(game): sync player score back to registry on correct answer` | processor.go, processor_test.go |
| 3 | `feat(game): add per-player cancel context and DisconnectPlayer helper; filter DISCONNECTED from broadcasts` | processor.go, server.go, processor_test.go |
| 4 | `feat(game): track last broadcast question on Game for catch-up on rejoin` | processor.go, processor_test.go |
| 5 | `feat(game): player rejoin — restore DISCONNECTED player with score and current question` | processor.go, server.go, processor_test.go |

---

## Success Criteria

### Verification Commands
```bash
go build ./...                              # Expected: exit 0
go vet ./...                                # Expected: exit 0
go test ./pkg/... -v                        # Expected: all PASS
go test ./pkg/gameengine/quiz/ -v -run TestRejoinPlayer   # Expected: all subtests PASS
go test ./pkg/gameengine/quiz/ -v -run TestBroadcastSkipsDisconnected  # Expected: PASS
go test ./pkg/gameengine/quiz/ -v -run TestDisconnectPlayer             # Expected: PASS
go test ./pkg/gameengine/quiz/ -v -run TestScoreTrackedInRegistry       # Expected: PASS
```

### Final Checklist
- [ ] All "Must Have" present: DISCONNECTED on disconnect, score preserved, last question on rejoin, old goroutine cancelled, broadcast filter, Result channel allocated, nil-guard
- [ ] All "Must NOT Have" absent: no proto changes, no timeout, no END behavior change, no new action enum, no DB calls, no frontend changes
- [ ] All 5 commits land in order; each independently buildable
