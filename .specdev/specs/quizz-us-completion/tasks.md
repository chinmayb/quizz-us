# Implementation Plan - QuizUS Application Completion

This implementation plan provides a comprehensive, ordered list of tasks to complete the QuizUS multiplayer quiz platform. Each task is designed to be actionable by a coding agent and references specific requirements from the requirements document.

## Backend Tasks

### Phase 1: Core Communication Layer

- [ ] 1. Fix WebSocket-to-gRPC message serialization
  - Create JSON struct types that map to protobuf GamePlay messages (GamePlayAction, GamePlayCommand, GameSummary)
  - Implement JSON unmarshaling in ws.go to parse incoming WebSocket messages
  - Implement protobuf-to-JSON marshaling for outgoing messages
  - Replace FIXME placeholders with proper message handling logic
  - Add error handling for malformed JSON messages
  - Write unit tests for JSON/Protobuf conversion functions
  - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [ ] 2. Implement game code generation utility
  - Create `GenerateGameCode()` function using crypto/rand for secure random generation
  - Implement 6-character alphanumeric code format (A-Z, 0-9)
  - Add uniqueness check against GameRegistry
  - Add collision handling with retry logic
  - Write unit tests for code generation (format, uniqueness, randomness)
  - _Requirements: 2.2, 2.5_

### Phase 2: Game Creation and Hosting

- [ ] 3. Add game creation endpoint
  - Define CreateGameRequest and CreateGameResponse protobuf messages
  - Add CreateGame RPC method to Games service in quizz-us.proto
  - Regenerate protobuf code using make/buf
  - Implement CreateGame handler in play/server.go
  - Initialize game with host player, settings, and categories
  - Return game code and metadata to client
  - _Requirements: 2.1, 2.2, 2.3_

- [ ] 4. Extend Game struct with new fields
  - Add `status` field (GameStatus enum: NOT_STARTED, IN_PROGRESS, FINISHED)
  - Add `hostID` field to track game host
  - Add `settings` field (GameSettings struct) for categories, duration, target score
  - Add `startTime` field for tracking game duration
  - Update game initialization in processor.go to set these fields
  - _Requirements: 2.3, 3.3_

- [ ] 5. Implement host-only authorization
  - Create `isHost()` validation function in processor.go
  - Add host check before allowing BEGIN action
  - Return error if non-host tries to start game
  - Write tests for host authorization logic
  - _Requirements: 2.4, 3.3_

### Phase 3: Pre-Game Lobby

- [ ] 6. Implement lobby player list broadcasting
  - Create function to build lobby update message using GameSummary with NOT_STARTED status
  - Broadcast updated player list when a player joins
  - Broadcast updated player list when a player disconnects
  - Include player names and status (WAITING) in the broadcast
  - Test lobby updates with multiple players joining/leaving
  - _Requirements: 3.1, 3.2, 3.4_

- [ ] 7. Enhance heartbeat mechanism for lobby
  - Track last heartbeat time per player
  - Create goroutine to periodically check for missed heartbeats
  - Send lobby status updates as heartbeat responses
  - Mark players as DISCONNECTED after 3 missed heartbeats (90 seconds)
  - Write tests for heartbeat timeout detection
  - _Requirements: 3.5, 10.1, 10.2_

### Phase 4: Question Lifecycle Management

- [ ] 8. Implement QuestionState tracking
  - Create QuestionState struct with StartTime, Duration, AnswerCount, TotalPlayers
  - Add CompletedChan to signal when all players have answered
  - Track per-question answer count in processor.go
  - Signal completion when answer count reaches total players
  - _Requirements: 4.6, 4.7_

- [ ] 9. Implement dynamic question timer
  - Replace fixed 30-second ticker with configurable duration from Game.settings
  - Create questionLifecycle function that manages timer and completion
  - Listen for timer expiration OR all players answering (whichever comes first)
  - Stop timer early if all players answer
  - Broadcast results at end of question lifecycle
  - Write tests for timer expiration and early completion
  - _Requirements: 4.3, 4.4, 4.6_

- [ ] 10. Implement question broadcast with metadata
  - Include question ID, text, and start timestamp in GamePlayCommand
  - Remove correct answer from question broadcast (only show after answering)
  - Add question duration to command so frontend can show timer
  - Test question broadcast to multiple players
  - _Requirements: 4.1, 4.2_

### Phase 5: Enhanced Answer Validation

- [ ] 11. Create advanced answer validation module
  - Create AnswerValidator struct with configurable fuzzy matching threshold
  - Implement normalizeAnswer function (lowercase, trim, collapse spaces)
  - Implement exact match check
  - Implement partial match check (substring matching)
  - Implement Levenshtein distance calculation for fuzzy matching
  - Return validation result with confidence score
  - Write comprehensive unit tests for all matching scenarios
  - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5_

- [ ] 12. Integrate answer validator into game processor
  - Replace simple ValidateAnswer with new AnswerValidator
  - Update answer processing logic to use validator
  - Log validation results with confidence scores
  - Handle edge cases (empty answers, very long answers)
  - Test integration with real quiz data
  - _Requirements: 9.1, 9.2, 9.3, 9.4_

### Phase 6: Scoring System

- [ ] 13. Implement time-based score calculation
  - Create ScoreCalculator struct with BasePoints and TimeBonusMax
  - Implement Calculate method that factors in answer time
  - Use linear time bonus (faster answer = more bonus points)
  - Update score immediately when correct answer is validated
  - Store answer timestamp when player submits answer
  - Write unit tests for score calculation at various times
  - _Requirements: 5.5_

- [ ] 14. Track and broadcast score updates
  - Update player score in PlayerObj when answer is correct
  - Include updated scores in result broadcast
  - Sort players by score in GameSummary
  - Broadcast intermediate score updates after each question
  - Test score consistency across concurrent answers
  - _Requirements: 5.2, 5.3_

### Phase 7: Category-Based Question Selection

- [ ] 15. Implement category selection in game creation
  - Add categories field to Game and Game.Spec
  - Pass categories from CreateGameRequest to Game initialization
  - Validate that selected categories exist in QuizDataByTag
  - Store categories in Game.settings
  - _Requirements: 6.1, 6.4_

- [ ] 16. Create question selector with category filtering
  - Create QuestionSelector struct with categories and usedIDs fields
  - Implement SelectNext method that filters by categories
  - Track used question IDs to avoid repeats
  - Fall back to all categories if selected ones are exhausted
  - Return error if no questions available
  - Write tests for category filtering and duplicate prevention
  - _Requirements: 6.2, 6.3, 6.5_

- [ ] 17. Integrate question selector into game processor
  - Replace random selection in ProduceQuestions with QuestionSelector
  - Initialize selector with game categories
  - Use selector for each question in game loop
  - Handle "no more questions" error gracefully (end game or wrap around)
  - Test with single category, multiple categories, and no category selection
  - _Requirements: 6.2, 6.3_

### Phase 8: Game End Detection

- [ ] 18. Implement game end condition checking
  - Create checkGameEnd function that runs after each question
  - Check if any player has reached target score
  - Check if game duration has exceeded target time
  - Return winner information when game should end
  - Test both end conditions separately and together
  - _Requirements: 8.1, 8.2_

- [ ] 19. Implement winner declaration and final broadcast
  - Create buildGameSummary function with final scores
  - Sort players by score in descending order
  - Set GamePlayStatus to GAME_OVER
  - Include winner in GameSummary
  - Broadcast final summary to all players
  - Test winner declaration with tie scenarios (same score)
  - _Requirements: 8.3, 8.4_

- [ ] 20. Clean up game state after completion
  - Remove game from GameRegistry after final broadcast
  - Close all player channels
  - Log game completion with final statistics
  - Implement timeout-based cleanup for abandoned games
  - Test cleanup doesn't affect active games
  - _Requirements: 8.5, 10.4_

### Phase 9: Connection Management

- [ ] 21. Implement disconnection detection
  - Track last heartbeat timestamp per player
  - Create goroutine that checks for stale connections
  - Mark player as DISCONNECTED after 3 missed heartbeats
  - Broadcast player status update to other players
  - Test disconnection detection with simulated missed heartbeats
  - _Requirements: 10.1, 10.2_

- [ ] 22. Implement reconnection logic
  - Allow reconnecting player to rejoin with same player ID
  - Send current game state to reconnecting player
  - Send current question if game is in progress
  - Update player status from DISCONNECTED to PLAYING
  - Broadcast reconnection to other players
  - Test reconnection at various game stages
  - _Requirements: 10.3_

- [ ] 23. Handle host disconnection
  - Detect when host disconnects
  - Implement host transfer to another player OR end game based on configuration
  - Notify all players if game ends due to host disconnect
  - Test host disconnection in lobby and during game
  - _Requirements: 10.5_

### Phase 10: Admin Quiz Management

- [ ] 24. Create admin API endpoint for quiz upload
  - Add UploadQuizData RPC method to a new Admin service
  - Accept YAML file content in request
  - Validate YAML structure and required fields
  - Return validation errors with specific line numbers if invalid
  - _Requirements: 7.1, 7.2, 7.3_

- [ ] 25. Implement quiz data validation
  - Validate required fields: question, answer, tags
  - Validate optional fields: imageSrc (valid URL), hints (array)
  - Check for duplicate question IDs
  - Check for empty or malformed tags
  - Write unit tests for validation logic
  - _Requirements: 7.2, 7.3_

- [ ] 26. Implement quiz data reload mechanism
  - Create ReloadQuizData function that updates QuizDataRefined and QuizDataByTag
  - Acquire write lock during reload
  - Swap in new data atomically
  - Log successful reload with question count
  - Test that in-progress games continue with old data
  - _Requirements: 7.4_

- [ ] 27. Create quiz listing endpoint
  - Add ListQuizzes RPC method to Admin service
  - Support filtering by category
  - Return question count per category
  - Support pagination for large datasets
  - Test with various category filters
  - _Requirements: 7.5_

## Frontend Tasks

### Phase 11: Host Game Flow

- [ ] 28. Create HostGameForm component
  - Create components/HostGameForm.js with Vue component
  - Add form fields: player name, category selection (multi-select), target score, question duration
  - Implement category list display with question counts
  - Add form validation for required fields
  - Emit 'create-game' event with form data
  - Write component with proper styling
  - _Requirements: 2.1, 6.1, 6.4_

- [ ] 29. Implement create game WebSocket message
  - Create sendCreateGame method in main app
  - Send CREATE action with host name and settings
  - Handle CreateGameResponse with game code
  - Transition to lobby view with game code displayed
  - Show shareable game link and copy button
  - Test game creation flow end-to-end
  - _Requirements: 2.2, 2.3, 2.4_

### Phase 12: Lobby Interface

- [ ] 30. Create LobbyView component
  - Create components/LobbyView.js with Vue component
  - Display game code prominently with copy button
  - Show list of joined players with avatars and names
  - Show "Waiting for players..." message
  - Display "Start Game" button for host only
  - Add player count indicator (e.g., "3 players joined")
  - _Requirements: 3.1, 3.2, 3.3_

- [ ] 31. Implement lobby player list updates
  - Handle GameSummary messages with NOT_STARTED status
  - Update player list reactively when new players join
  - Show visual indicator when players disconnect
  - Animate player additions/removals
  - Test with multiple players joining/leaving
  - _Requirements: 3.1, 3.2, 3.4_

- [ ] 32. Implement game start functionality
  - Show "Start Game" button only if user is host
  - Send BEGIN action when start button clicked
  - Disable start button while waiting for response
  - Transition to game view when first question received
  - Test host-only start restriction
  - _Requirements: 3.3, 4.1_

### Phase 13: Question Display and Timer

- [ ] 33. Create QuestionTimer component
  - Create components/QuestionTimer.js with Vue component
  - Display countdown in seconds
  - Use computed property for time remaining
  - Add visual urgency (color change) when < 5 seconds remain
  - Emit 'timer-expired' event when time runs out
  - Animate timer progress bar
  - _Requirements: 4.3, 11.3, 11.5_

- [ ] 34. Enhance QuizQuestion component with timer integration
  - Integrate QuestionTimer into QuizQuestion component
  - Start timer when question is received
  - Auto-submit answer when timer expires
  - Disable input when timer expires
  - Show question image with proper loading state
  - Display hints after configurable time threshold
  - _Requirements: 11.1, 11.2, 11.3, 11.4_

- [ ] 35. Implement question display flow
  - Handle incoming GamePlayCommand messages
  - Extract question text, ID, and duration
  - Update currentQuestion state
  - Clear previous answer
  - Reset submission state
  - Transition to 'playing' view
  - Test question transitions
  - _Requirements: 4.1, 4.2, 11.1_

### Phase 14: Answer Submission

- [ ] 36. Implement answer submission logic
  - Capture answer input from QuizQuestion component
  - Send GamePlayCommand with player answer on submit
  - Include question ID and timestamp
  - Disable submit button after submission
  - Show "Submitted" state while waiting for result
  - Handle Enter key for submission
  - _Requirements: 4.5_

- [ ] 37. Implement answer result feedback
  - Handle result in GameSummary or separate result message
  - Show correct/incorrect indicator
  - Display correct answer if user was wrong
  - Show points earned for correct answer
  - Animate result appearance
  - Test with correct and incorrect answers
  - _Requirements: 5.1, 5.2_

### Phase 15: Score Display and Rankings

- [ ] 38. Enhance ScoreDisplay component
  - Update ScoreDisplay to show all players with scores
  - Sort players by score in descending order
  - Highlight current user
  - Show rank position (1st, 2nd, 3rd, etc.)
  - Add visual distinction for top 3 players
  - Update scores reactively when new results arrive
  - _Requirements: 5.2, 5.3, 5.4_

- [ ] 39. Create ResultsScreen component
  - Create components/ResultsScreen.js with Vue component
  - Show correct answer prominently
  - Display updated scoreboard
  - Show who answered correctly with checkmarks
  - Show points earned by each player for that question
  - Display "Next question in X seconds..." countdown
  - _Requirements: 5.1, 5.2, 5.3, 5.4_

- [ ] 40. Implement results screen timing
  - Display results for fixed duration (e.g., 5 seconds)
  - Show countdown to next question
  - Auto-transition to next question when timer expires
  - Allow manual "Continue" button for testing
  - Test result screen timing and transitions
  - _Requirements: 5.4_

### Phase 16: Game Over Screen

- [ ] 41. Create GameOverScreen component
  - Create components/GameOverScreen.js with Vue component
  - Display winner with celebration animation
  - Show final scoreboard with all players
  - Display game statistics (total questions, time played, etc.)
  - Add "Play Again" button that returns to home
  - Add "View Details" option for detailed stats
  - _Requirements: 8.3, 8.4, 8.5_

- [ ] 42. Implement game over state handling
  - Handle GameSummary with GAME_OVER status
  - Extract winner information
  - Transition to 'gameover' view
  - Display game over screen with final data
  - Handle navigation back to home page
  - Test game over flow from last question
  - _Requirements: 8.3, 8.4, 8.5_

### Phase 17: Connection Management UI

- [ ] 43. Enhance connection status indicators
  - Show persistent connection indicator in header/footer
  - Display reconnection attempts with progress
  - Show "Reconnecting..." message during reconnect
  - Display error message after max retries exceeded
  - Add manual "Retry Connection" button
  - Test with simulated connection loss
  - _Requirements: 10.2, 10.3_

- [ ] 44. Implement disconnection notifications
  - Show toast notification when player disconnects
  - Display in-lobby notification when players leave
  - Show in-game indicator when players disconnect
  - Notify when host disconnects
  - Test notifications at various game stages
  - _Requirements: 10.2, 10.5_

### Phase 18: User Experience Enhancements

- [ ] 45. Add loading states to all async operations
  - Show loading spinner during game creation
  - Show spinner while joining game
  - Show spinner while waiting for question
  - Implement skeleton screens for data loading
  - Test loading states with slow connections
  - _Requirements: UI/UX best practices_

- [ ] 46. Implement error handling and user feedback
  - Display user-friendly error messages for all error cases
  - Add inline validation errors in forms
  - Show toast notifications for temporary errors
  - Provide retry options for failed operations
  - Test all error scenarios (invalid code, connection loss, etc.)
  - _Requirements: 1.3, UI/UX best practices_

- [ ] 47. Add accessibility improvements
  - Ensure all interactive elements are keyboard accessible
  - Add ARIA labels to components
  - Implement focus management for modals
  - Add screen reader announcements for important events
  - Test with keyboard-only navigation
  - _Requirements: Accessibility best practices_

- [ ] 48. Implement responsive design
  - Ensure all components work on mobile devices
  - Adjust layouts for tablet and phone screens
  - Test touch interactions
  - Optimize font sizes and button sizes for mobile
  - Test on various device sizes
  - _Requirements: Responsive design best practices_

## Integration and Testing Tasks

### Phase 19: Integration Testing

- [ ] 49. Write integration tests for complete game flow
  - Create test that simulates host creating game
  - Add test for multiple players joining
  - Test game start by host
  - Test question-answer-result cycle
  - Test game end and winner declaration
  - Use mock WebSocket connections
  - _Requirements: All requirements_

- [ ] 50. Test concurrent player scenarios
  - Test multiple players answering simultaneously
  - Test race conditions in score updates
  - Verify message broadcast ordering
  - Test with high player count (20+ players)
  - Profile for performance issues
  - _Requirements: Performance requirements_

- [ ] 51. Test disconnection and reconnection scenarios
  - Test player disconnect during lobby
  - Test player disconnect during question
  - Test successful reconnection
  - Test host disconnect handling
  - Test all players disconnecting
  - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5_

### Phase 20: End-to-End Testing

- [ ] 52. Create automated E2E tests
  - Set up E2E testing framework (Playwright or Cypress)
  - Create test for full game flow from host perspective
  - Create test for player join and play flow
  - Test error scenarios (invalid codes, etc.)
  - Test across different browsers
  - _Requirements: All requirements_

- [ ] 53. Perform manual testing scenarios
  - Test on multiple devices simultaneously
  - Verify real-time synchronization
  - Test network interruptions (disconnect Wi-Fi, etc.)
  - Verify all UI states and transitions
  - Check for console errors and warnings
  - _Requirements: All requirements_

### Phase 21: Documentation and Deployment

- [ ] 54. Update API documentation
  - Document all WebSocket message formats
  - Document gRPC service methods
  - Add example messages for each operation
  - Document error codes and meanings
  - Create API reference guide
  - _Requirements: Documentation_

- [ ] 55. Create deployment configuration
  - Create Dockerfile for backend
  - Create docker-compose for local development
  - Add environment variable configuration
  - Create deployment guide
  - Add health check endpoints
  - _Requirements: Deployment_

- [ ] 56. Update README and user guides
  - Update README with complete setup instructions
  - Add user guide for hosting games
  - Add user guide for playing games
  - Document quiz data format and upload process
  - Add troubleshooting section
  - _Requirements: Documentation_

## Notes

- Tasks are ordered to minimize dependencies and allow incremental progress
- Each task includes specific requirements it addresses
- Backend tasks should be completed before dependent frontend tasks
- Testing tasks can run in parallel with implementation
- All code should include appropriate error handling and logging
