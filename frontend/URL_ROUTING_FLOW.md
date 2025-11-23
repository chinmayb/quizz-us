# URL Routing Flow Diagram

## User Journey with URL Changes

```
┌─────────────────────────────────────────────────────────────────┐
│                         HOME PAGE                                │
│                  https://yoursite.com/                           │
│                                                                   │
│  [Join Game]              [Host Game]                           │
└─────────────────────────────────────────────────────────────────┘
           │                        │
           │ User enters            │ User enters
           │ Game Code: ABC123      │ Game Code: XYZ789
           │ Player Name: John      │ Player Name: Alice
           ↓                        ↓
┌──────────────────────┐  ┌──────────────────────┐
│   JOIN GAME          │  │   HOST GAME          │
│                      │  │                      │
│ handleJoinGame()     │  │ handleHostGame()     │
│   ↓                  │  │   ↓                  │
│ updateUrlWithGameCode│  │ updateUrlWithGameCode│
└──────────────────────┘  └──────────────────────┘
           │                        │
           │                        │
           ↓                        ↓
┌─────────────────────────────────────────────────────────────────┐
│                      GAME PAGE                                   │
│              https://yoursite.com/ABC123                         │
│              https://yoursite.com/XYZ789                         │
│                                                                   │
│  Game Code: ABC123          [Leave Game]                        │
│  Player: John                                                    │
│  Status: Waiting...                                              │
└─────────────────────────────────────────────────────────────────┘
           │
           │ User clicks "Leave Game"
           │      or
           │ User clicks Browser Back Button
           ↓
┌──────────────────────┐
│   LEAVE GAME         │
│                      │
│ leaveGame()          │
│   ↓                  │
│ updateUrlToHome()    │
└──────────────────────┘
           │
           ↓
┌─────────────────────────────────────────────────────────────────┐
│                         HOME PAGE                                │
│                  https://yoursite.com/                           │
└─────────────────────────────────────────────────────────────────┘
```

## URL State Transitions

```
Initial Load
     │
     ↓
┌──────────────────────────┐
│ Check URL Path           │
│ window.location.pathname │
└──────────────────────────┘
     │
     ├─── "/" (empty/home) ──→ Show Home Page
     │                          history.replaceState({ view: 'home' })
     │
     └─── "/ABC123" (game) ──→ handleDirectUrlAccess()
                                │
                                ├─ Valid Game Code → Show Join Prompt
                                └─ Invalid → Redirect to Home
```

## Browser Navigation (Back/Forward)

```
User History Stack:

1. https://yoursite.com/              ← Initial page
2. https://yoursite.com/ABC123        ← Joined game
3. https://yoursite.com/              ← Left game

User clicks Back Button from step 3:
     ↓
┌──────────────────────┐
│ popstate event fired │
│ state: { view: 'game', gameCode: 'ABC123' }
└──────────────────────┘
     ↓
handleBrowserNavigation()
     ↓
Check if currently in game:
     ├─ Yes → Maintain state
     └─ No  → Show message or redirect home
```

## Method Call Flow

```
Join Game Flow:
┌─────────────────────────────────┐
│ User clicks "Join Game" button  │
└─────────────────────────────────┘
              ↓
┌─────────────────────────────────┐
│ handleJoinGame(gameData)        │
│ - Sets gameCode, playerName     │
│ - Sends WebSocket JOIN action   │
│ - Sets currentView = 'game'     │
└─────────────────────────────────┘
              ↓
┌─────────────────────────────────┐
│ updateUrlWithGameCode(gameCode) │
│ - history.pushState()           │
│ - Updates document.title        │
└─────────────────────────────────┘
              ↓
      URL: /ABC123
      Title: "Game ABC123 - QuizUS"


Leave Game Flow:
┌─────────────────────────────────┐
│ User clicks "Leave Game" button │
│ or Browser Back Button          │
└─────────────────────────────────┘
              ↓
┌─────────────────────────────────┐
│ leaveGame()                     │
│ - Sends WebSocket LEAVE action  │
│ - Clears game state             │
│ - Sets currentView = 'home'     │
└─────────────────────────────────┘
              ↓
┌─────────────────────────────────┐
│ updateUrlToHome()               │
│ - history.pushState()           │
│ - Resets document.title         │
└─────────────────────────────────┘
              ↓
      URL: /
      Title: "QuizUS - Multiplayer Quiz Platform"
```

## History State Structure

```javascript
// Home Page State
{
  view: 'home'
}

// Game Page State
{
  view: 'game',
  gameCode: 'ABC123'
}

// Accessing state
window.history.state.gameCode  // 'ABC123'
window.history.state.view      // 'game'
```

## URL Validation Logic

```
URL Path: /ABC123
     ↓
Split by '/' → ['', 'ABC123']
     ↓
Filter empty → ['ABC123']
     ↓
Get first part → 'ABC123'
     ↓
Validate:
├─ Length >= 4           ✓
├─ Uppercase letters     ✓
├─ Numbers allowed       ✓
└─ Special chars?        ✗

Valid Game Code? YES → Process
                 NO  → Redirect Home
```

## Event Listeners

```
Application Lifecycle:
┌────────────────────────┐
│ mounted() hook         │
│ - connectWebSocket()   │
│ - addEventListener()   │  ← Register popstate listener
│ - handleDirectURL()    │
└────────────────────────┘
         │
         │ App is running
         │ User navigates
         ↓
┌────────────────────────┐
│ popstate event         │
│ ↓                      │
│ handleBrowserNav()     │  ← Handle back/forward
└────────────────────────┘
         │
         │ App is closing
         ↓
┌────────────────────────┐
│ beforeUnmount() hook   │
│ - removeEventListener()│  ← Clean up
│ - disconnectWebSocket()│
└────────────────────────┘
```

## Shareable Link Flow

```
Player A (Host):
1. Creates game → URL: /XYZ789
2. Copies URL
3. Shares with friends

Player B (Joining):
1. Opens URL: /XYZ789
     ↓
2. handleDirectUrlAccess()
     ↓
3. Detects game code "XYZ789"
     ↓
4. Shows toast: "Enter your name to join game XYZ789"
     ↓
5. User enters name
     ↓
6. handleJoinGame({ gameCode: 'XYZ789', playerName: 'Bob' })
     ↓
7. URL stays: /XYZ789
8. Player B joins the game
```
