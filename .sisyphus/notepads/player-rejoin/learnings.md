- Allocating PlayerObj.Result at creation prevents nil-send leaks when broadCastResult fires.
- Guarding RemovePlayerFromRegistry against a missing game avoids panics from stale or absent game IDs.

- DisconnectPlayer must call cancelCtx OUTSIDE the lock to avoid deadlock — copy the cancel func, unlock, then call it.
- cancelCtx field is unexported; exposed via SetCancelFunc setter for cross-package access from server.go.
- broadCastQuestion and broadCastResult both need DISCONNECTED filter — skip players with Status == DISCONNECTED in the fan-out loop.
- "all players gone" check at stream-close must count active (non-DISCONNECTED) players, not use len(players), since DISCONNECTED players remain in the map.
- context.WithCancel at JOIN time creates a per-player cancel context derived from stream.Context(); the playerCtx is not used to replace stream.Context().

- lastQuestion is stored on Game struct with its own sync.RWMutex (lastQMu) — separate from the registry lock to avoid lock nesting.
- GetLastQuestion uses a two-phase lock release: acquire GameRegistry.mu.RLock() via GetGame() (which releases before returning), then acquire game.lastQMu.RLock() separately — no nested locking.
- broadCastQuestion stores the question after receiving from the channel and before fanning out, using game.lastQMu.Lock()/Unlock() directly (no defer, short critical section).
- Working tree may contain uncommitted test changes from a parallel task (Task 3 tests were present before Task 4 started); always check git status before assuming the file matches HEAD.
- When working-tree test file references symbols not yet in processor.go, the build fails; implementing the missing symbols (Task 3 impl was already present in working tree) resolves it.

- RejoinPlayer must release GameRegistry.mu.Lock() BEFORE calling GetLastQuestion(), which internally acquires GameRegistry.mu.RLock() via GetGame() — avoids RLock-inside-Lock deadlock.
- The old cancelCtx is called INSIDE the lock in RejoinPlayer (context.CancelFunc is non-blocking) — safe and prevents a race where old goroutine could read stale channels.
- GetPlayer returns (*PlayerObj, error) not (*PlayerObj, bool) — rejoin detection uses `err == nil` to detect existing player.
- Fresh channels (newQ, newResult) are created INSIDE the goroutine in server.go, before calling RejoinPlayer — ensures the goroutine's select loop reads from the same channels that get swapped into the registry.
- playObj.QuestionForPlayer and playObj.Result are updated AFTER RejoinPlayer returns so the local select loop uses the new channels.
- isRejoin logic: `playerExists == nil` (no error) means player was found — covers both DISCONNECTED and PLAYING states for kick-and-rejoin.
