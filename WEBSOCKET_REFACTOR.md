# WebSocket Connection Refactoring

## Summary
Refactored the application to move WebSocket connection initialization from the homepage to the game join/host actions. This prevents unnecessary WebSocket connections on every page load/refresh.

## Changes Made

### 1. **index-modular.html**
- **Removed**: `connectWebSocket()` call from `mounted()` lifecycle hook
- **Updated**: `handleJoinGame()` to establish WebSocket connection before joining
- **Updated**: `handleHostGame()` to establish WebSocket connection before hosting  
- **Updated**: `leaveGame()` to disconnect WebSocket when leaving
- **Removed**: `isConnected` and `connectionStatus` props from `<home-page>` component
- **Removed**: `isConnected` props from `<join-game-form>` and `<host-game-form>` components

### 2. **components/HomePage.js**
- **Removed**: `isConnected` and `connectionStatus` props
- **Removed**: Connection status display UI and computed property
- **Removed**: `:disabled="!isConnected"` from Join and Host buttons

### 3. **components/JoinGameForm.js**
- **Removed**: `isConnected` prop
- **Removed**: Connection warning message in template
- **Updated**: `canJoin` computed property to not check connection status

### 4. **components/HostGameForm.js**
- **Removed**: `isConnected` prop
- **Removed**: Connection warning message in template
- **Updated**: `canHost` computed property to not check connection status
- **Updated**: `validateForm()` to remove connection check

## Architecture Before vs After

### Before ❌
```
Page Load → WebSocket Connect → Homepage Display → User Action → Game
                ↑
         Reconnects on every refresh
```

### After ✅
```
Page Load → Homepage Display → User Action → WebSocket Connect → Game
                                                     ↑
                              Only connects when needed
```

## Benefits

1. **No unnecessary connections**: Homepage doesn't establish WebSocket until user joins/hosts
2. **Cleaner logs**: No more "going away" errors on homepage refresh
3. **Better resource management**: Server doesn't maintain idle connections
4. **Improved UX**: Faster homepage load, connection happens when needed
5. **Proper lifecycle**: WebSocket opens on game join, closes on game leave

## Connection Flow

### Joining a Game
1. User clicks "Join" button on homepage
2. User enters name and game code in modal
3. User clicks "Join Game" button
4. App establishes WebSocket connection (shows "Connecting to server...")
5. Once connected, sends JOIN action to server
6. Transitions to game view in waiting state

### Hosting a Game  
1. User clicks "Host" button on homepage
2. User enters name and generates/enters game code
3. User clicks "Start Game" button
4. App establishes WebSocket connection (shows "Connecting to server...")
5. Once connected, sends JOIN action with HOST role
6. Transitions to game view in waiting state
7. Host can click "Start Game" to send BEGIN action

### Leaving a Game
1. User clicks "Leave Game" button
2. App sends END action to server
3. App disconnects WebSocket
4. Returns to homepage with no active connection

## Error Handling

- Connection timeout: 10 seconds
- If connection fails during join/host, user sees error toast
- isJoining/isHosting flags are reset on failure
- User can retry without page refresh

## Testing Checklist

- [ ] Homepage loads without WebSocket connection
- [ ] No "going away" errors on homepage refresh
- [ ] Join game establishes connection successfully
- [ ] Host game establishes connection successfully
- [ ] Leave game disconnects WebSocket
- [ ] Connection timeout shows error message
- [ ] Forms work correctly without connection checks
- [ ] Game functionality remains unchanged

## Notes

- Homepage is now purely presentational
- WebSocket connection state is managed per game session
- Connection status UI removed from homepage (not needed)
- Forms validate input but don't check connection (connection happens on submit)
