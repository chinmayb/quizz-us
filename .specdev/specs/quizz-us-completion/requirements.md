# Requirements Document - QuizUS Application Completion

## Introduction

This document outlines the requirements for completing the QuizUS multiplayer quiz platform. The application allows users to host quiz games, invite players via game codes, and play in real-time with question timers and score tracking. The system consists of a Go backend with gRPC/WebSocket communication and a Vue.js frontend.

Based on the existing codebase analysis, several core features are partially implemented while others need to be completed. This document focuses on the remaining work required to achieve a fully functional multiplayer quiz experience.

## Requirements

### Requirement 1: Complete WebSocket Message Serialization

**User Story:** As a developer, I want proper JSON serialization/deserialization between WebSocket and gRPC, so that the frontend and backend can communicate effectively.

#### Acceptance Criteria

1. WHEN a player sends a message via WebSocket THEN the system SHALL properly deserialize the JSON message into a GamePlay protobuf message
2. WHEN the backend sends a GamePlay response THEN the system SHALL serialize the protobuf message to JSON before sending via WebSocket
3. WHEN invalid JSON is received THEN the system SHALL log the error and send an appropriate error response to the client
4. WHEN a player joins a game THEN the system SHALL correctly parse the player ID, game code, and action fields

### Requirement 2: Complete Question Flow Implementation

**User Story:** As a player, I want to receive questions with timers and submit answers, so that I can participate in the quiz game.

#### Acceptance Criteria

1. WHEN the host starts the game THEN all players SHALL receive the first question simultaneously
2. WHEN a question is sent THEN it SHALL include the question text, question ID, and timestamp
3. WHEN a question is displayed THEN the frontend SHALL show a countdown timer based on the question duration
4. WHEN the timer expires THEN the system SHALL automatically move to the next question for all players
5. WHEN a player submits an answer THEN the system SHALL validate it and update their score if correct
6. WHEN all players answer THEN the system SHALL immediately show results and move to the next question
7. IF a player doesn't answer THEN the system SHALL treat it as an incorrect answer

### Requirement 3: Implement Game Hosting Flow

**User Story:** As a host, I want to create a new game and receive a unique game code, so that I can invite players to join my quiz game.

#### Acceptance Criteria

1. WHEN a user clicks the "host" button THEN the system SHALL generate a unique 6-character alphanumeric game code
2. WHEN a game is created THEN the system SHALL initialize a new game with configurable settings (question duration, target score)
3. WHEN a game code is generated THEN the system SHALL display it to the host with options to copy/share
4. WHEN a host creates a game THEN the system SHALL automatically join them as the first player
5. IF a game code already exists THEN the system SHALL generate a new unique code

### Requirement 4: Implement Pre-Game Lobby

**User Story:** As a player, I want to see who has joined the game before it starts, so that I know when all my friends have joined.

#### Acceptance Criteria

1. WHEN a player joins a game THEN all players in the lobby SHALL receive an updated player list
2. WHEN players are in the lobby THEN the system SHALL display each player's name and avatar
3. WHEN the host is in the lobby THEN the system SHALL display a "Start Game" button visible only to the host
4. WHEN a player disconnects from the lobby THEN all other players SHALL see the updated player list
5. WHEN players are waiting THEN the system SHALL send periodic heartbeat messages to maintain connections

### Requirement 5: Implement Real-Time Result Display

**User Story:** As a player, I want to see the correct answer and updated scores after each question, so that I can track my progress and compete with others.

#### Acceptance Criteria

1. WHEN all players answer or the timer expires THEN the system SHALL broadcast the correct answer to all players
2. WHEN answers are validated THEN the system SHALL update and broadcast the current scoreboard
3. WHEN a result is displayed THEN it SHALL show each player's score sorted by ranking
4. WHEN the result screen is shown THEN it SHALL display for a fixed duration before the next question
5. WHEN a player answers correctly THEN the system SHALL award points with time-based bonuses for faster answers

### Requirement 6: Implement Category Selection

**User Story:** As a host, I want to choose quiz categories when creating a game, so that I can customize the topic of questions.

#### Acceptance Criteria

1. WHEN creating a game THEN the host SHALL be able to select one or more quiz categories
2. WHEN categories are selected THEN the system SHALL only serve questions from those categories
3. WHEN no categories are selected THEN the system SHALL serve questions from all available categories
4. WHEN categories are displayed THEN they SHALL show the number of available questions in each
5. IF a selected category has fewer questions than needed THEN the system SHALL include questions from other selected categories

### Requirement 7: Implement Quiz Data Administration

**User Story:** As an admin, I want to upload and manage quiz questions, so that the platform has fresh and diverse content.

#### Acceptance Criteria

1. WHEN an admin uploads a YAML file THEN the system SHALL parse and validate the quiz data structure
2. WHEN quiz data is uploaded THEN it SHALL include question text, answer, category tags, and optional hints/images
3. WHEN invalid quiz data is uploaded THEN the system SHALL return validation errors specifying the issues
4. WHEN quiz data is successfully uploaded THEN the system SHALL reload the in-memory quiz data cache
5. WHEN an admin wants to view quizzes THEN the system SHALL provide an endpoint to list all questions by category

### Requirement 8: Implement Game End and Winner Declaration

**User Story:** As a player, I want to see the final results and winner when the game ends, so that I know who won the quiz.

#### Acceptance Criteria

1. WHEN a player reaches the target score THEN the system SHALL end the game and declare them the winner
2. WHEN the target time is reached THEN the system SHALL end the game and declare the highest scorer as winner
3. WHEN the game ends THEN all players SHALL receive a GameSummary with final scores and winner information
4. WHEN a game ends THEN the system SHALL display a winner celebration screen on the frontend
5. WHEN a game ends THEN players SHALL have options to return to home, play again, or view detailed stats

### Requirement 9: Implement Improved Answer Validation

**User Story:** As a player, I want flexible answer matching, so that minor spelling variations don't count as wrong answers.

#### Acceptance Criteria

1. WHEN validating answers THEN the system SHALL use case-insensitive comparison
2. WHEN validating answers THEN the system SHALL trim leading/trailing whitespace
3. WHEN validating answers THEN the system SHALL accept partial matches (e.g., "Gavaskar" for "Sunil Gavaskar")
4. WHEN validating answers THEN the system SHALL use fuzzy matching with configurable tolerance for typos
5. WHEN an answer is ambiguous THEN the system SHALL apply consistent validation rules

### Requirement 10: Implement Player Disconnection Handling

**User Story:** As a player, I want the game to continue if someone disconnects, so that one player's connection issues don't ruin the game for everyone.

#### Acceptance Criteria

1. WHEN a player misses N consecutive heartbeats THEN the system SHALL mark them as disconnected
2. WHEN a player is disconnected THEN other players SHALL be notified via updated player status
3. WHEN a disconnected player reconnects THEN they SHALL rejoin the game at the current question
4. WHEN all players disconnect THEN the system SHALL clean up the game after a timeout period
5. WHEN the host disconnects THEN the system SHALL either transfer host status or end the game based on configuration

### Requirement 11: Implement Frontend Question Display

**User Story:** As a player, I want to see questions clearly with images, hints, and a countdown timer, so that I have all the information needed to answer.

#### Acceptance Criteria

1. WHEN a question is received THEN the frontend SHALL display the question text prominently
2. WHEN a question has an image THEN it SHALL be displayed with proper loading states
3. WHEN a question is displayed THEN the countdown timer SHALL show remaining time in seconds
4. WHEN hints are available THEN they SHALL be displayed after a certain time threshold
5. WHEN the timer reaches critical time (e.g., 5 seconds) THEN the UI SHALL provide visual urgency indicators

### Requirement 12: Implement Game State Persistence

**User Story:** As a host, I want games to be saved to a database, so that we can view game history and stats later.

#### Acceptance Criteria

1. WHEN a game is created THEN its metadata SHALL be persisted to the database
2. WHEN players join THEN their participation SHALL be recorded in the database
3. WHEN answers are submitted THEN they SHALL be logged with timestamps for analysis
4. WHEN a game ends THEN the final results SHALL be permanently stored
5. WHEN retrieving game history THEN the system SHALL support filtering by player, date, and category

## Non-Functional Requirements

### Performance
1. The system SHALL support at least 10 concurrent games with 20 players each
2. Question delivery SHALL have latency under 100ms from server to all players
3. The frontend SHALL load and become interactive within 3 seconds on standard connections

### Reliability
1. WebSocket connections SHALL automatically reconnect with exponential backoff
2. The system SHALL handle concurrent player actions without race conditions
3. Game state SHALL remain consistent across all player connections

### Security
1. Game codes SHALL be cryptographically random to prevent guessing
2. Player IDs SHALL be unique and validated on each action
3. The system SHALL prevent unauthorized users from becoming game hosts

### Usability
1. The UI SHALL provide clear feedback for all user actions
2. Error messages SHALL be user-friendly and actionable
3. The game flow SHALL be intuitive without requiring instructions
