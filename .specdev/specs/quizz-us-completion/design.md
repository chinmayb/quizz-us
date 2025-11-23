# Design Document - QuizUS Application Completion

## Overview

This design document details the architecture and implementation approach for completing the QuizUS multiplayer quiz platform. The application is built with:
- **Backend**: Go with gRPC for core game logic and WebSocket for real-time client communication
- **Frontend**: Vue.js 3 with component-based architecture
- **Data Storage**: YAML-based quiz data with future database integration
- **Communication**: Bidirectional streaming via gRPC, bridged to WebSocket for browser clients

The design focuses on completing the missing functionality while maintaining compatibility with existing code structures.

## Architecture

### High-Level Architecture

```
┌─────────────────┐         WebSocket          ┌──────────────────┐
│                 │◄──────────────────────────►│                  │
│  Vue.js Frontend│         JSON Messages       │  WebSocket       │
│                 │                             │  Handler (ws.go) │
└─────────────────┘                             └──────────────────┘
                                                         │
                                                         │ Protobuf
                                                         ▼
                                                ┌──────────────────┐
                                                │  gRPC Server     │
                                                │  (play/server.go)│
                                                └──────────────────┘
                                                         │
                                                         ▼
                                                ┌──────────────────┐
                                                │  Game Processor  │
                                                │  (processor.go)  │
                                                └──────────────────┘
                                                         │
                                                         ▼
                                                ┌──────────────────┐
                                                │  Quiz Engine     │
                                                │  (engine.go)     │
                                                └──────────────────┘
```

### Component Interaction Flow

1. **Game Creation**: Frontend → WebSocket → gRPC → GameProcessor (initialize game state)
2. **Player Join**: Frontend → WebSocket → gRPC → GameProcessor (add to registry)
3. **Game Start**: Host → WebSocket → gRPC → GameProcessor (trigger question flow)
4. **Question Flow**: GameProcessor → Quiz Engine → All Players via channels
5. **Answer Submission**: Player → WebSocket → gRPC → GameProcessor → Quiz Engine (validate)
6. **Result Broadcast**: GameProcessor → All Players via channels → WebSocket → Frontend

## Components and Interfaces

### 1. WebSocket Handler (cmd/ws.go)

**Current Issues:**
- FIXMEs at lines 60 and 85 indicate incomplete JSON/Protobuf serialization
- Currently passing `nil` to gRPC stream instead of parsed messages

**Design Solution:**

```go
type WebSocketMessage struct {
    ID      string                 `json:"id"`
    Code    string                 `json:"code"`
    Action  string                 `json:"action,omitempty"`
    Command *GamePlayCommandJSON   `json:"command,omitempty"`
    Summary *GameSummaryJSON       `json:"summary,omitempty"`
}

type GamePlayCommandJSON struct {
    ID            string `json:"id"`
    PlayerAnswer  string `json:"player_answer"`
    Question      string `json:"question,omitempty"`
    CorrectAnswer string `json:"correct_answer,omitempty"`
}
```

**Implementation Approach:**
- Unmarshal WebSocket JSON messages into Go structs
- Convert JSON structs to protobuf GamePlay messages
- Marshal protobuf responses to JSON before sending to WebSocket
- Add proper error handling for malformed messages

### 2. Game Creation and Host Flow

**Current Gap:** No implementation for game creation endpoint or host-specific logic

**Design Solution:**

Add new protobuf message:
```protobuf
message CreateGameRequest {
  string host_player_id = 1;
  string host_player_name = 2;
  repeated string categories = 3;
  Game.Spec spec = 4;
}

message CreateGameResponse {
  string game_code = 1;
  string game_id = 2;
  Game game = 3;
}
```

Add REST endpoint:
```go
// POST /games
func CreateGame(w http.ResponseWriter, r *http.Request) {
    // Generate 6-character alphanumeric code
    // Initialize game in registry
    // Return game code and metadata
}
```

**Game Code Generation:**
```go
func GenerateGameCode() string {
    const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    const codeLength = 6
    // Use crypto/rand for secure random generation
    // Ensure uniqueness by checking GameRegistry
}
```

### 3. Pre-Game Lobby

**Design Solution:**

Add to GameProcessor:
```go
type Game struct {
    GamePro  gameProcessor
    players  PlayersMap
    status   GameStatus  // NEW: NOT_STARTED, IN_PROGRESS, FINISHED
    hostID   string      // NEW: Host player ID
    settings GameSettings // NEW: Category, duration, target score
}

type GameStatus int
const (
    GameNotStarted GameStatus = iota
    GameInProgress
    GameFinished
)
```

**Lobby Update Mechanism:**
- When player joins: broadcast updated player list via Result channel
- Use GameSummary with status NOT_STARTED to send lobby updates
- Include player list with status (WAITING, READY, DISCONNECTED)

### 4. Question Timer and Lifecycle

**Current Implementation:** Fixed 30-second ticker in processor.go

**Design Enhancement:**

```go
type QuestionState struct {
    QuizData      *data.QuizData
    StartTime     time.Time
    Duration      time.Duration
    AnswerCount   int
    TotalPlayers  int
    CompletedChan chan bool
}

func (g *Game) questionLifecycle(ctx context.Context, question *data.QuizData) {
    state := &QuestionState{
        QuizData:      question,
        StartTime:     time.Now(),
        Duration:      g.settings.QuestionDuration,
        TotalPlayers:  len(g.players),
        CompletedChan: make(chan bool),
    }
    
    timer := time.NewTimer(state.Duration)
    
    select {
    case <-timer.C:
        // Timer expired - show results
        g.broadcastResults(ctx, state)
    case <-state.CompletedChan:
        // All players answered - show results early
        timer.Stop()
        g.broadcastResults(ctx, state)
    }
}
```

### 5. Answer Validation Enhancement

**Current Implementation:** Simple case-insensitive string comparison

**Design Enhancement:**

```go
type AnswerValidator struct {
    FuzzyThreshold float64 // Levenshtein distance threshold
}

func (v *AnswerValidator) Validate(userAnswer, correctAnswer string) (bool, float64) {
    // 1. Normalize: lowercase, trim, remove extra spaces
    normalized := normalizeAnswer(userAnswer)
    expected := normalizeAnswer(correctAnswer)
    
    // 2. Exact match check
    if normalized == expected {
        return true, 1.0
    }
    
    // 3. Partial match (e.g., "Gavaskar" in "Sunil Gavaskar")
    if strings.Contains(expected, normalized) || strings.Contains(normalized, expected) {
        return true, 0.9
    }
    
    // 4. Fuzzy match using Levenshtein distance
    distance := levenshteinDistance(normalized, expected)
    similarity := 1.0 - float64(distance)/float64(max(len(normalized), len(expected)))
    
    return similarity >= v.FuzzyThreshold, similarity
}

func normalizeAnswer(answer string) string {
    // lowercase, trim, collapse multiple spaces
    a := strings.TrimSpace(strings.ToLower(answer))
    return regexp.MustCompile(`\s+`).ReplaceAllString(a, " ")
}
```

### 6. Score Calculation with Time Bonus

**Design Solution:**

```go
type ScoreCalculator struct {
    BasePoints      int
    TimeBonusMax    int
    QuestionDuration time.Duration
}

func (sc *ScoreCalculator) Calculate(answerTime time.Time, questionStart time.Time, isCorrect bool) int {
    if !isCorrect {
        return 0
    }
    
    elapsed := answerTime.Sub(questionStart)
    
    // Time bonus: more points for faster answers
    // Linear decrease from max bonus to 0
    remainingRatio := 1.0 - (float64(elapsed) / float64(sc.QuestionDuration))
    if remainingRatio < 0 {
        remainingRatio = 0
    }
    
    timeBonus := int(float64(sc.TimeBonusMax) * remainingRatio)
    
    return sc.BasePoints + timeBonus
}
```

### 7. Category Selection and Question Filtering

**Current Implementation:** Random selection from all questions

**Design Enhancement:**

```go
type QuestionSelector struct {
    categories []string
    usedIDs    map[string]bool
}

func (qs *QuestionSelector) SelectNext(ctx context.Context) (*data.QuizData, error) {
    var eligibleQuestions []string
    
    // Filter by selected categories
    if len(qs.categories) > 0 {
        for _, cat := range qs.categories {
            if questionIDs, ok := data.QuizDataByTag[cat]; ok {
                eligibleQuestions = append(eligibleQuestions, questionIDs...)
            }
        }
    } else {
        // Use all questions if no categories selected
        for id := range data.QuizDataRefined {
            eligibleQuestions = append(eligibleQuestions, id)
        }
    }
    
    // Filter out already used questions
    var available []string
    for _, id := range eligibleQuestions {
        if !qs.usedIDs[id] {
            available = append(available, id)
        }
    }
    
    if len(available) == 0 {
        return nil, fmt.Errorf("no more questions available")
    }
    
    // Select random from available
    selectedID := available[rand.Intn(len(available))]
    qs.usedIDs[selectedID] = true
    
    question := data.QuizDataRefined[selectedID]
    return &question, nil
}
```

### 8. Game End Detection and Winner Declaration

**Design Solution:**

```go
func (g *Game) checkGameEnd(ctx context.Context) (*pb.GameSummary, bool) {
    var topScore int32
    var winner *pb.Player
    
    for _, player := range g.players {
        if player.Player.Score > topScore {
            topScore = player.Player.Score
            winner = player.Player
        }
    }
    
    // Check target score condition
    if topScore >= g.settings.TargetScore {
        return g.buildGameSummary(winner, pb.GamePlayStatus_GAME_OVER), true
    }
    
    // Check target time condition
    elapsed := time.Since(g.startTime)
    if elapsed >= g.settings.TargetTime {
        return g.buildGameSummary(winner, pb.GamePlayStatus_GAME_OVER), true
    }
    
    return nil, false
}

func (g *Game) buildGameSummary(winner *pb.Player, status pb.GamePlayStatus) *pb.GameSummary {
    var players []*pb.Player
    for _, p := range g.players {
        players = append(players, p.Player)
    }
    
    // Sort by score descending
    sort.Slice(players, func(i, j int) bool {
        return players[i].Score > players[j].Score
    })
    
    return &pb.GameSummary{
        Players: players,
        Winner:  winner,
        Status:  status,
    }
}
```

### 9. Frontend Component Integration

**Current Components:** HomePage, JoinGameForm, QuizQuestion, ScoreDisplay, SidebarMenu, UserAvatars

**Missing Components:**
1. **HostGameForm** - Create game with category selection
2. **LobbyView** - Show waiting players before game starts
3. **QuestionTimer** - Visual countdown timer
4. **ResultsScreen** - Show answer results and updated scores
5. **GameOverScreen** - Final winner and stats display

**Frontend State Management:**

```javascript
const gameState = {
    // Current view
    view: 'home', // home, lobby, playing, results, gameover
    
    // Connection
    websocket: null,
    isConnected: false,
    
    // Game info
    gameCode: null,
    playerId: null,
    playerName: null,
    isHost: false,
    
    // Game data
    currentQuestion: null,
    questionStartTime: null,
    questionDuration: 30,
    players: [],
    myScore: 0,
    
    // UI state
    userAnswer: '',
    answerSubmitted: false,
    showResults: false,
    lastResult: null
}
```

**WebSocket Message Handlers:**

```javascript
function handleWebSocketMessage(message) {
    const data = JSON.parse(message.data);
    
    if (data.command) {
        // New question received
        handleQuestion(data.command);
    } else if (data.summary) {
        // Game summary (lobby update, results, or game over)
        handleSummary(data.summary);
    }
}

function handleQuestion(command) {
    gameState.currentQuestion = {
        id: command.id,
        text: command.question,
        answer: command.correct_answer
    };
    gameState.questionStartTime = Date.now();
    gameState.view = 'playing';
    gameState.answerSubmitted = false;
    gameState.showResults = false;
    
    // Start timer
    startQuestionTimer();
}

function handleSummary(summary) {
    if (summary.status === 'NOT_STARTED') {
        // Lobby update
        gameState.players = summary.players;
        gameState.view = 'lobby';
    } else if (summary.status === 'ON_GOING') {
        // Results after question
        gameState.players = summary.players;
        gameState.showResults = true;
        updateScores(summary.players);
    } else if (summary.status === 'GAME_OVER') {
        // Game ended
        gameState.players = summary.players;
        gameState.winner = summary.winner;
        gameState.view = 'gameover';
    }
}
```

### 10. Database Integration (Future Enhancement)

**Design for Persistence:**

```go
type GameRepository interface {
    CreateGame(game *Game) error
    GetGame(code string) (*Game, error)
    UpdateGame(game *Game) error
    SaveAnswer(gameID, playerID, questionID, answer string, isCorrect bool, score int) error
    GetGameHistory(playerID string, limit int) ([]*GameSummary, error)
}

// Implementation would use PostgreSQL or MongoDB
type PostgresGameRepo struct {
    db *sql.DB
}
```

**Schema Design:**
```sql
CREATE TABLE games (
    id UUID PRIMARY KEY,
    code VARCHAR(6) UNIQUE NOT NULL,
    host_player_id VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL,
    target_score INT,
    question_duration INT,
    created_at TIMESTAMP,
    started_at TIMESTAMP,
    ended_at TIMESTAMP
);

CREATE TABLE game_players (
    game_id UUID REFERENCES games(id),
    player_id VARCHAR(255),
    player_name VARCHAR(255),
    final_score INT,
    joined_at TIMESTAMP,
    PRIMARY KEY (game_id, player_id)
);

CREATE TABLE game_answers (
    id UUID PRIMARY KEY,
    game_id UUID REFERENCES games(id),
    player_id VARCHAR(255),
    question_id VARCHAR(50),
    player_answer TEXT,
    is_correct BOOLEAN,
    points_earned INT,
    answer_time TIMESTAMP
);
```

## Data Models

### Existing Protobuf Messages (Enhanced)
```protobuf
message Game {
  message Spec {
    google.protobuf.Duration question_duration = 1;
    int32 target_score = 2;
    google.protobuf.Duration target_time = 3;
  }
  string id = 1;
  string game_kind_id = 2;
  string code = 3;
  google.protobuf.Timestamp created_at = 4;
  google.protobuf.Timestamp updated_at = 5;
  string result = 6;
  string status = 7;
  Spec spec = 8;
  string host_player_id = 9;  // NEW
  repeated string categories = 10;  // NEW
}

message Player {
  string id = 1;
  int32 score = 2;
  string name = 3;
  bool is_bot = 4;
  PlayerStatus status = 5;
  bool is_host = 6;  // NEW
}
```

## Error Handling

### Backend Error Handling

1. **Invalid Game Code**
   - Return error message: "Game not found"
   - Log attempt with player ID
   - Frontend displays user-friendly error

2. **Player Connection Loss**
   - Mark player as DISCONNECTED after 3 missed heartbeats
   - Broadcast status update to other players
   - Allow rejoin within 5-minute window

3. **Malformed Messages**
   - Log error with message content
   - Send error response via WebSocket
   - Don't crash server - continue processing

4. **Quiz Data Errors**
   - Validate quiz data on upload
   - Return specific validation errors
   - Fall back to default questions if corruption detected

### Frontend Error Handling

1. **WebSocket Connection Failure**
   - Display reconnection UI
   - Automatic retry with exponential backoff
   - Max 5 retry attempts before showing manual refresh option

2. **Invalid Game Code**
   - Show inline error in JoinGameForm
   - Suggest checking code or asking host

3. **Timeout/No Response**
   - Show loading spinner with timeout (10s)
   - After timeout, show error and retry option

## Testing Strategy

### Unit Tests

1. **Answer Validation**
   - Test exact matches
   - Test partial matches
   - Test fuzzy matching with various distances
   - Test normalization (case, whitespace)

2. **Score Calculation**
   - Test base points
   - Test time bonus at various answer times
   - Test zero points for incorrect

3. **Game Code Generation**
   - Test uniqueness
   - Test format (6 chars, alphanumeric)
   - Test collision handling

4. **Question Selection**
   - Test category filtering
   - Test avoiding duplicates
   - Test handling exhausted questions

### Integration Tests

1. **WebSocket ↔ gRPC Bridge**
   - Test JSON serialization/deserialization
   - Test message routing
   - Test error propagation

2. **Game Lifecycle**
   - Test create → lobby → start → questions → end flow
   - Test player join/leave at each stage
   - Test host-specific actions

3. **Multi-Player Scenarios**
   - Test concurrent player answers
   - Test race conditions in scoring
   - Test broadcast to all players

### End-to-End Tests

1. **Complete Game Flow**
   - Host creates game
   - Multiple players join
   - Host starts game
   - Players answer questions
   - Game ends with winner declaration

2. **Disconnection Recovery**
   - Player disconnects mid-game
   - Player reconnects and continues
   - Game continues for other players

3. **Edge Cases**
   - All players disconnect
   - Host disconnects
   - Invalid answer submissions
   - Timer edge cases

### Frontend Component Tests

1. **Component Rendering**
   - Test all components render correctly
   - Test with various props
   - Test event emissions

2. **WebSocket Integration**
   - Mock WebSocket connection
   - Test message sending
   - Test message receiving and state updates

3. **User Interactions**
   - Test form validation
   - Test button clicks
   - Test keyboard shortcuts (Enter to submit)

## Performance Considerations

1. **Concurrent Game Handling**
   - Each game runs in its own goroutine
   - Game registry uses RWMutex for thread-safe access
   - Consider sharding registry if >100 concurrent games

2. **Question Broadcast**
   - Fan-out pattern with goroutines (already implemented)
   - Non-blocking channel sends with timeout
   - Buffer channels to prevent blocking

3. **WebSocket Connection Management**
   - Connection pooling for gRPC clients
   - Heartbeat mechanism to detect stale connections
   - Cleanup goroutines on disconnect

4. **Memory Management**
   - Remove finished games from registry after cleanup period
   - Limit quiz data cache size
   - Profile for memory leaks in long-running games

## Security Considerations

1. **Game Code Security**
   - Use crypto/rand for generation (not math/rand)
   - 36^6 = 2.1 billion combinations for 6-character code
   - Short-lived validity (e.g., expire after 24 hours)

2. **Player Validation**
   - Validate player ID on every action
   - Prevent impersonation
   - Rate limit actions per player

3. **Input Validation**
   - Sanitize player names (prevent XSS)
   - Validate game settings (prevent abuse)
   - Limit answer length

4. **CORS and WebSocket Origin**
   - Already implemented in ws.go
   - Whitelist allowed origins
   - Production: only allow specific domains

## Deployment Considerations

1. **Configuration**
   - Environment variables for ports, timeouts
   - Quiz data file path
   - Database connection strings (future)

2. **Logging**
   - Structured logging with slog (already used)
   - Log levels: DEBUG for dev, INFO for prod
   - Log aggregation for monitoring

3. **Health Checks**
   - Endpoint for k8s/docker health checks
   - Check WebSocket handler status
   - Check gRPC server status

4. **Graceful Shutdown**
   - Handle SIGTERM/SIGINT
   - Finish in-progress games or notify players
   - Close connections gracefully
