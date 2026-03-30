# Low-Complexity Architectural Fixes

## TL;DR

> **Quick Summary**: Three targeted surgical fixes addressing a security leak, orphaned goroutine lifecycle, and a misplaced transport package — no behavior changes, no new features.
>
> **Deliverables**:
> - `CorrectAnswer` stripped from question broadcasts in `pkg/play/server.go`
> - Per-game `context.CancelFunc` stored in `Game` struct, wired into `RemoveGame`
> - `cmd/ws.go` + `cmd/ws_test.go` moved to `pkg/transport/ws/`
>
> **Estimated Effort**: Quick
> **Parallel Execution**: NO — sequential (each fix is an independent atomic commit)
> **Critical Path**: Task 1 → Task 2 → Task 3

---

## Context

### Original Request
"Let's fix the low complexity tasks" — referring to the three low-complexity issues surfaced in the architecture review.

### Issues Being Fixed

| # | Issue | Severity | File(s) |
|---|---|---|---|
| 1 | Correct answer sent to every client in question broadcast | 🔴 Critical | `pkg/play/server.go:125` |
| 2 | Game goroutine bound to first player's stream context | 🔴 Critical | `pkg/play/server.go:26-44`, `pkg/gameengine/quiz/processor.go:139-154` |
| 3 | WebSocket handler (transport logic) lives in CLI package `cmd/` | 🟡 Medium | `cmd/ws.go`, `cmd/ws_test.go`, `cmd/serve.go` |

### Research Findings (Metis-validated)
- **Fix 1**: `CorrectAnswer` is populated at exactly one place (`server.go:125`). No tests assert on it. One-line removal + comment.
- **Fix 2**: `RemoveGame` has exactly 1 call site (`server.go:161`). `Game` struct has no cancel func today. Context must be `context.Background()` (NOT stream context) to decouple game from any one player's connection.
- **Fix 3**: All symbols in `cmd/ws.go` are self-contained — no references to unexported `cmd/` symbols. Tests move with the code (same package), only package declaration changes.

### Metis Review
**Gaps addressed**:
- Added nil-check before calling `game.cancelFn()` in `RemoveGame`
- `cancelFn()` must be called BEFORE `delete()` in `RemoveGame`
- `go mod tidy` required after Fix 3
- Comment explaining WHY `CorrectAnswer` is omitted (prevents re-introduction)
- Broadcast goroutines (`broadCastQuestion`, `broadCastScores`, etc.) are explicitly OUT OF SCOPE

---

## Work Objectives

### Core Objective
Apply three surgical fixes without changing any game logic, scoring, or player management behavior.

### Concrete Deliverables
- `pkg/play/server.go` — `CorrectAnswer` field removed from outbound question message
- `pkg/gameengine/quiz/processor.go` — `Game.cancelFn` field added; `RemoveGame` calls it
- `pkg/play/server.go` — `initGame` creates its own `context.WithCancel(context.Background())`
- `pkg/transport/ws/handler.go` — moved from `cmd/ws.go`
- `pkg/transport/ws/handler_test.go` — moved from `cmd/ws_test.go`
- `cmd/serve.go` — updated import for new ws package path

### Definition of Done
- [ ] `go test -v ./...` passes
- [ ] `go build ./...` compiles clean
- [ ] `go vet ./...` produces no errors
- [ ] `cmd/ws.go` and `cmd/ws_test.go` no longer exist
- [ ] `pkg/transport/ws/handler.go` exists with `package ws`

### Must Have
- All existing tests pass after every fix
- Each fix is one atomic commit
- `CorrectAnswer` removed from question-phase broadcast
- Per-game cancel func called in `RemoveGame`
- WS logic fully moved to `pkg/transport/ws/`

### Must NOT Have (Guardrails)
- Do NOT change the proto definition — `GamePlayCommand.correct_answer` field stays
- Do NOT rename any exported symbols (especially `WSHandler`)
- Do NOT make unexported functions exported during the WS move
- Do NOT use `stream.Context()` as parent for game context (use `context.Background()`)
- Do NOT propagate context to broadcast goroutines (`broadCastQuestion`, `broadCastScores`, `BroadCastLobby`, `broadCastResult`, `ProduceQuestions`) — out of scope
- Do NOT change CORS origins in the `upgrader` during the move
- Do NOT change any game logic, scoring, or player management behavior
- Do NOT add any new features or abstractions beyond what's listed

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed.

### Test Decision
- **Infrastructure exists**: YES (`go test`)
- **Automated tests**: Tests-after (existing tests must pass; no new tests required for these fixes)
- **Framework**: Go standard `testing` package

### QA Policy
Every task includes agent-executed verification. Evidence saved to `.sisyphus/evidence/`.

---

## Execution Strategy

### Sequential Execution (No Parallelism — by design)

Each fix is independently verifiable and atomic. Run them in order:

```
Task 1: Strip CorrectAnswer from question broadcast
    ↓ (go test ./pkg/... passes)
Task 2: Add per-game cancel context
    ↓ (go test ./pkg/... passes)
Task 3: Move WS handler to pkg/transport/ws/
    ↓ (go test ./... passes, go build ./... compiles)
```

### Agent Dispatch
- Task 1 → `quick`
- Task 2 → `quick`
- Task 3 → `quick`

---

## TODOs

- [x] 1. Strip `CorrectAnswer` from question-phase broadcast

  **What to do**:
  - Open `pkg/play/server.go`
  - Find the question send block around line 120-126 inside the `case quizQuestion := <-playObj.QuestionForPlayer:` select arm
  - Remove the line `CorrectAnswer: quizQuestion.Answer,`
  - Replace the removed TODO comment (line 124) with an explanatory comment: `// CorrectAnswer intentionally omitted — only exposed in results/summary phase`
  - The resulting `GamePlayCommand` struct literal should only have: `Id`, `Question`, `QuestionTime`

  **Must NOT do**:
  - Do NOT remove `correct_answer` from the proto file
  - Do NOT touch `broadCastResult` or the `GameSummary` path — those are fine
  - Do NOT change any other field in that struct literal
  - Do NOT modify scoring or game logic

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Single-line removal in one file, crystal-clear scope
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Sequential — must be first
  - **Blocks**: Task 2, Task 3
  - **Blocked By**: None (can start immediately)

  **References**:

  **Pattern References**:
  - `pkg/play/server.go:112-131` — The goroutine spawned on JOIN; find the `GamePlayCommand` struct literal inside `case quizQuestion`
  - `pkg/play/server.go:124` — The TODO comment to replace

  **Why Each Reference Matters**:
  - Line 120-131 is the exact struct literal to modify — remove `CorrectAnswer` field, add explanatory comment

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: CorrectAnswer is absent from the question-send struct literal
    Tool: Bash (ast_grep_search or grep)
    Preconditions: Fix applied
    Steps:
      1. Run: grep -n "CorrectAnswer" pkg/play/server.go
      2. Assert: no line contains `CorrectAnswer: quizQuestion.Answer`
      3. Assert: the explanatory comment IS present (grep for "intentionally omitted")
    Expected Result: grep returns 0 matches for CorrectAnswer assignment; comment present
    Evidence: .sisyphus/evidence/task-1-correct-answer-removed.txt

  Scenario: Existing tests still pass
    Tool: Bash (go test)
    Preconditions: Fix applied
    Steps:
      1. Run: go test -v ./pkg/... 2>&1
      2. Assert: exit code 0
      3. Assert: output contains "PASS" and zero "FAIL"
    Expected Result: All pkg tests pass
    Evidence: .sisyphus/evidence/task-1-tests-pass.txt

  Scenario: go vet clean
    Tool: Bash
    Preconditions: Fix applied
    Steps:
      1. Run: go vet ./pkg/play/... 2>&1
      2. Assert: empty output, exit code 0
    Expected Result: No vet errors
    Evidence: .sisyphus/evidence/task-1-vet-clean.txt
  ```

  **Evidence to Capture**:
  - [ ] `task-1-correct-answer-removed.txt` — grep output showing absence
  - [ ] `task-1-tests-pass.txt` — go test output
  - [ ] `task-1-vet-clean.txt` — go vet output

  **Commit**: YES
  - Message: `fix(security): strip correct answer from question broadcast`
  - Files: `pkg/play/server.go`
  - Pre-commit: `go test ./pkg/...`

---

- [ ] 2. Add per-game cancel context to prevent orphaned goroutines

  **What to do**:
  - **In `pkg/gameengine/quiz/processor.go`**:
    - Add `cancelFn context.CancelFunc` field to the `Game` struct (after `answeredCorrectly`)
    - In `RemoveGame(gameID string)`: before `delete(GameRegistry.games, gameID)`, retrieve the game and call `game.cancelFn()` with a nil-check:
      ```go
      if game, ok := GameRegistry.games[gameID]; ok {
          if game.cancelFn != nil {
              game.cancelFn()
          }
      }
      delete(GameRegistry.games, gameID)
      ```

  - **In `pkg/play/server.go`**:
    - In `initGame(ctx context.Context, code string)`: replace the use of the passed-in `ctx` with a new independent context:
      ```go
      gameCtx, cancel := context.WithCancel(context.Background())
      ```
    - Pass `gameCtx` to `p.Process(gameCtx)` instead of `ctx`
    - After `quiz.AddGame(code, p)` succeeds, set `p.cancelFn = cancel` directly on the processor before the goroutine starts
    - The `ctx context.Context` parameter to `initGame` can be kept or removed — keep it to avoid changing the signature if it's exported or tested; just stop using it for the goroutine

  **Must NOT do**:
  - Do NOT use `stream.Context()` or the `ctx` parameter passed into `initGame` as the parent context — use `context.Background()`
  - Do NOT add context to `broadCastQuestion`, `broadCastScores`, `BroadCastLobby`, `broadCastResult`, or `ProduceQuestions`
  - Do NOT change the signature of `Process(ctx context.Context) error`
  - Do NOT change any game logic or scoring

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Struct field addition + two targeted wiring changes across two files
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Sequential — must follow Task 1
  - **Blocks**: Task 3
  - **Blocked By**: Task 1

  **References**:

  **Pattern References**:
  - `pkg/gameengine/quiz/processor.go:139-154` — `Game` struct definition; add `cancelFn` field here
  - `pkg/gameengine/quiz/processor.go:74-79` — `RemoveGame` function; add cancelFn call before delete
  - `pkg/play/server.go:26-44` — `initGame` function; replace ctx with `context.WithCancel(context.Background())`
  - `pkg/gameengine/quiz/processor.go:126-137` — `NewGameProcessor`; `cancelFn` will be set directly after creation (field assignment, not constructor param)

  **Why Each Reference Matters**:
  - `Game` struct needs the new field so it can be stored and retrieved in `RemoveGame`
  - `RemoveGame` is the single teardown point — this is where cancel must fire
  - `initGame` is where the goroutine is spawned — this is where the new context is created

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: cancelFn field present in Game struct
    Tool: Bash (grep)
    Preconditions: Fix applied
    Steps:
      1. Run: grep -n "cancelFn" pkg/gameengine/quiz/processor.go
      2. Assert: at least 2 matches — one in struct definition, one in RemoveGame
    Expected Result: 2+ matches confirming field declaration and usage
    Evidence: .sisyphus/evidence/task-2-cancelfn-wired.txt

  Scenario: initGame uses context.Background() not stream context
    Tool: Bash (grep)
    Preconditions: Fix applied
    Steps:
      1. Run: grep -n "context.WithCancel" pkg/play/server.go
      2. Assert: exactly 1 match inside initGame using context.Background()
      3. Run: grep -n "context.Background" pkg/play/server.go
      4. Assert: the match is in initGame
    Expected Result: context.WithCancel(context.Background()) present in initGame
    Evidence: .sisyphus/evidence/task-2-context-background.txt

  Scenario: All tests pass
    Tool: Bash (go test)
    Preconditions: Fix applied
    Steps:
      1. Run: go test -v ./pkg/... 2>&1
      2. Assert: exit code 0, zero FAIL lines
    Expected Result: Full pkg test suite passes
    Evidence: .sisyphus/evidence/task-2-tests-pass.txt

  Scenario: go vet clean
    Tool: Bash
    Preconditions: Fix applied
    Steps:
      1. Run: go vet ./pkg/... 2>&1
      2. Assert: empty output, exit code 0
    Expected Result: No vet errors
    Evidence: .sisyphus/evidence/task-2-vet-clean.txt
  ```

  **Evidence to Capture**:
  - [ ] `task-2-cancelfn-wired.txt` — grep confirming field + RemoveGame usage
  - [ ] `task-2-context-background.txt` — grep confirming context.Background() in initGame
  - [ ] `task-2-tests-pass.txt` — go test output
  - [ ] `task-2-vet-clean.txt` — go vet output

  **Commit**: YES
  - Message: `fix(lifecycle): add per-game cancel context to prevent orphaned goroutines`
  - Files: `pkg/gameengine/quiz/processor.go`, `pkg/play/server.go`
  - Pre-commit: `go test ./pkg/...`

---

- [ ] 3. Move WebSocket handler from `cmd/` to `pkg/transport/ws/`

  **What to do**:
  - Create directory `pkg/transport/ws/`
  - Create `pkg/transport/ws/handler.go`:
    - Copy full contents of `cmd/ws.go`
    - Change `package cmd` → `package ws`
    - All function signatures, logic, CORS origins, imports — unchanged
  - Create `pkg/transport/ws/handler_test.go`:
    - Copy full contents of `cmd/ws_test.go`
    - Change `package cmd` → `package ws`
  - Update `cmd/serve.go`:
    - Add import: `ws "github.com/chinmayb/quizz-us/pkg/transport/ws"`
    - Change call at line 103: `WSHandler(ctx, *logger, client)` → `ws.WSHandler(ctx, *logger, client)`
  - Delete `cmd/ws.go`
  - Delete `cmd/ws_test.go`
  - Run `go mod tidy` and include any changes

  **Must NOT do**:
  - Do NOT rename `WSHandler` or any other function
  - Do NOT make unexported functions exported
  - Do NOT change the CORS origin list in the `upgrader`
  - Do NOT change any function signatures or logic
  - Do NOT modify `cmd/client.go`, `cmd/root.go`, or `cmd/config.go`

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Pure file move + two-line import update; no logic changes
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Sequential — must follow Task 2
  - **Blocks**: Nothing
  - **Blocked By**: Task 2

  **References**:

  **Pattern References**:
  - `cmd/ws.go` — full source to move (275 lines); change only the package declaration
  - `cmd/ws_test.go` — full test file to move (120 lines); change only the package declaration
  - `cmd/serve.go:103` — the single call site of `WSHandler` to update

  **Why Each Reference Matters**:
  - `cmd/ws.go` is the entire file being moved — copy verbatim, change package
  - `cmd/serve.go:103` is the only consumer of `WSHandler` — needs import + call site update
  - Tests move with the code so they remain in-package (testing unexported functions)

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Old files are gone, new files exist with correct package
    Tool: Bash (ls + grep)
    Preconditions: Fix applied
    Steps:
      1. Run: ls cmd/ws.go 2>&1; echo "exit:$?"
      2. Assert: exit code non-zero (file does not exist)
      3. Run: ls cmd/ws_test.go 2>&1; echo "exit:$?"
      4. Assert: exit code non-zero (file does not exist)
      5. Run: head -1 pkg/transport/ws/handler.go
      6. Assert: output is "package ws"
      7. Run: head -1 pkg/transport/ws/handler_test.go
      8. Assert: output is "package ws"
    Expected Result: Old files gone, new files with correct package declaration
    Evidence: .sisyphus/evidence/task-3-files-moved.txt

  Scenario: Full build compiles clean
    Tool: Bash (go build)
    Preconditions: Fix applied
    Steps:
      1. Run: go build ./... 2>&1
      2. Assert: exit code 0, no error output
    Expected Result: Clean compilation
    Evidence: .sisyphus/evidence/task-3-build-clean.txt

  Scenario: All 5 WS tests pass in new location
    Tool: Bash (go test)
    Preconditions: Fix applied
    Steps:
      1. Run: go test -v ./pkg/transport/ws/... 2>&1
      2. Assert: exit code 0
      3. Assert: "TestParseWebSocketMessage_Action" — PASS
      4. Assert: "TestParseWebSocketMessage_Command" — PASS
      5. Assert: "TestParseWebSocketMessage_Invalid" — PASS
      6. Assert: "TestBuildWebSocketPayload_Action" — PASS
      7. Assert: "TestBuildWebSocketPayload_Command" — PASS
    Expected Result: All 5 tests pass
    Evidence: .sisyphus/evidence/task-3-ws-tests-pass.txt

  Scenario: Full test suite passes
    Tool: Bash (go test)
    Preconditions: Fix applied
    Steps:
      1. Run: go test -v ./... 2>&1
      2. Assert: exit code 0, zero FAIL lines
    Expected Result: All packages pass
    Evidence: .sisyphus/evidence/task-3-full-suite-pass.txt

  Scenario: go vet clean across entire repo
    Tool: Bash
    Preconditions: Fix applied
    Steps:
      1. Run: go vet ./... 2>&1
      2. Assert: empty output, exit code 0
    Expected Result: No vet errors
    Evidence: .sisyphus/evidence/task-3-vet-clean.txt
  ```

  **Evidence to Capture**:
  - [ ] `task-3-files-moved.txt` — ls output confirming old gone, new exist
  - [ ] `task-3-build-clean.txt` — go build output
  - [ ] `task-3-ws-tests-pass.txt` — go test output for new package
  - [ ] `task-3-full-suite-pass.txt` — full go test ./... output
  - [ ] `task-3-vet-clean.txt` — go vet output

  **Commit**: YES
  - Message: `refactor(transport): move WebSocket handler from cmd/ to pkg/transport/ws/`
  - Files: `pkg/transport/ws/handler.go`, `pkg/transport/ws/handler_test.go`, `cmd/serve.go` (delete `cmd/ws.go`, `cmd/ws_test.go`)
  - Pre-commit: `go build ./... && go test ./...`

---

## Final Verification Wave

> Run after ALL three tasks. All must pass before marking work complete.

- [ ] F1. **Full suite + vet + build** — `quick`

  Run the following in sequence and confirm all pass:
  ```bash
  go build ./...        # Expected: exit 0, no output
  go vet ./...          # Expected: exit 0, no output
  go test -v ./...      # Expected: exit 0, all PASS
  ```
  Output: `Build [PASS/FAIL] | Vet [PASS/FAIL] | Tests [N pass / 0 fail] | VERDICT: APPROVE/REJECT`

- [ ] F2. **File existence audit** — `quick`

  Verify:
  - `cmd/ws.go` — DOES NOT exist
  - `cmd/ws_test.go` — DOES NOT exist
  - `pkg/transport/ws/handler.go` — EXISTS, package `ws`
  - `pkg/transport/ws/handler_test.go` — EXISTS, package `ws`
  - `pkg/play/server.go` — contains `context.WithCancel(context.Background())` in `initGame`
  - `pkg/play/server.go` — does NOT contain `CorrectAnswer: quizQuestion.Answer`
  - `pkg/gameengine/quiz/processor.go` — contains `cancelFn context.CancelFunc` in `Game` struct
  - `pkg/gameengine/quiz/processor.go` — `RemoveGame` calls `game.cancelFn()` before `delete`

  Output: `[N/8] checks pass | VERDICT: APPROVE/REJECT`

---

## Commit Strategy

| # | Message | Files | Gate |
|---|---|---|---|
| 1 | `fix(security): strip correct answer from question broadcast` | `pkg/play/server.go` | `go test ./pkg/...` |
| 2 | `fix(lifecycle): add per-game cancel context to prevent orphaned goroutines` | `pkg/gameengine/quiz/processor.go`, `pkg/play/server.go` | `go test ./pkg/...` |
| 3 | `refactor(transport): move WebSocket handler from cmd/ to pkg/transport/ws/` | `pkg/transport/ws/handler.go`, `pkg/transport/ws/handler_test.go`, `cmd/serve.go` (rm `cmd/ws.go`, `cmd/ws_test.go`) | `go build ./... && go test ./...` |

---

## Success Criteria

```bash
go build ./...   # Expected: no output, exit 0
go vet ./...     # Expected: no output, exit 0
go test ./...    # Expected: all PASS, exit 0
ls cmd/ws.go     # Expected: No such file
ls pkg/transport/ws/handler.go  # Expected: file exists
grep -n "CorrectAnswer: quizQuestion.Answer" pkg/play/server.go  # Expected: no match
grep -n "cancelFn" pkg/gameengine/quiz/processor.go  # Expected: 2+ matches
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] `go test ./...` passes
- [ ] `go build ./...` compiles
- [ ] `go vet ./...` clean
- [ ] Three atomic commits created
