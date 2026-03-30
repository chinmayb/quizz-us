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
