# Implementation Summary: Room Lifetime Feature

## Changes Made

### 1. Command-Line Interface (`cmd/olcrtc/main.go`)

- Added `-lifetime` flag to accept room lifetime in seconds
- Updated `config` struct to include `lifetime` field
- Updated `toSessionConfig()` to pass lifetime to session config

### 2. Session Configuration (`internal/app/session/session.go`)

- Added `Lifetime` field to `Config` struct
- Updated `server.Run()` call to pass lifetime parameter
- Updated `client.Run()` call to pass lifetime parameter

### 3. Server Implementation (`internal/server/server.go`)

- Added fields to `Server` struct to store configuration needed for room rotation:
  - `lifetime`: Room lifetime in seconds
  - `roomURL`, `linkName`, `transportName`, `carrierName`: Connection parameters
  - Video and VP8 configuration fields
- Added `ServiceMessage` struct for room rotation messages
- Updated `Run()` function signature to accept `lifetime` parameter
- Modified `Run()` to start `lifetimeManager` goroutine when lifetime > 0

### 4. Server Lifetime Management (`internal/server/lifetime.go`)

New file containing:
- `lifetimeManager()`: Periodically triggers room rotation every N seconds
- `rotateRoom()`: Handles the room rotation process:
  - Generates new room ID
  - Broadcasts service message
  - Closes current connection
  - Reconnects to new room
- `broadcastServiceMessage()`: Sends room rotation message via dedicated smux stream
- `createRotationRoom()`: Creates carrier-specific room IDs; WBStream rooms are created through the WBStream API before reconnecting
- `buildNewRoomURL()`: Constructs new room URL based on carrier type

### 5. Client Implementation (`internal/client/client.go`)

- Added fields to `Client` struct to store configuration for reconnection:
  - `lifetime`: Room lifetime in seconds
  - Connection parameters and video configuration
- Added `ServiceMessage` struct for receiving room rotation messages
- Updated `Run()` function signature to accept `lifetime` parameter
- Updated `RunWithReady()` function signature to accept `lifetime` parameter
- Modified `RunWithReady()` to initialize client with all necessary fields
- Added `serviceMessageListener()` goroutine startup

### 6. Client Lifetime Management (`internal/client/lifetime.go`)

New file containing:
- `serviceMessageListener()`: Continuously accepts streams for service messages
- `readServiceMessage()`: Reads and processes service messages from streams
- `handleServiceMessage()`: Parses JSON service messages
- `handleRoomRotation()`: Handles reconnection to new room:
  - Closes current connection
  - Builds new room URL
  - Reconnects to new room
- `buildNewRoomURL()`: Constructs new room URL based on carrier type

## Architecture

### Server-Side Flow

```
Run() with lifetime > 0
  ↓
bringUpLink()
  ↓
Start lifetimeManager goroutine
  ↓
Every N seconds:
  rotateRoom()
    ├─ Generate new room ID
    ├─ broadcastServiceMessage()
    ├─ Close current connection
    └─ bringUpLink() to new room
```

### Client-Side Flow

```
RunWithReady()
  ↓
Start serviceMessageListener goroutine
  ↓
Listen for service messages
  ↓
Receive room_rotate message
  ↓
handleRoomRotation()
  ├─ Close current connection
  ├─ Build new room URL
  └─ bringUpLink() to new room
```

## Key Design Decisions

1. **Service Messages via Dedicated Stream**: Service messages are sent via a separate smux stream to avoid interfering with regular tunnel traffic.

2. **Automatic Client Handling**: Clients automatically handle room rotation without requiring any special configuration or flags.

3. **Configuration Storage**: Both server and client store all necessary configuration parameters to enable seamless reconnection to new rooms.

4. **Reconnection Retry**: The client uses 30-second timeouts and up to 3 reconnect attempts with backoff.

5. **Carrier-Agnostic**: The implementation works with all supported carriers (telemost, jazz, wbstream) by building appropriate room URLs.

## Testing Recommendations

1. **Basic Functionality**: Start server with `-lifetime 10` and verify room rotation every 10 seconds
2. **Client Reconnection**: Verify clients automatically reconnect to new rooms
3. **Connection Behavior**: Verify new SOCKS5 connections work after rotation; active TCP streams can be interrupted and should be retried by callers
4. **Multiple Clients**: Test with multiple clients connecting to the same server
5. **Error Handling**: Test behavior when reconnection fails
6. **Different Carriers**: Test with different carrier types (telemost, jazz, wbstream)

## Files Modified

- `cmd/olcrtc/main.go`: Added lifetime flag
- `internal/app/session/session.go`: Added lifetime to config
- `internal/server/server.go`: Added lifetime support and service message struct
- `internal/server/lifetime.go`: NEW - Server lifetime management
- `internal/client/client.go`: Added lifetime support and service message struct
- `internal/client/lifetime.go`: NEW - Client lifetime management
- `internal/names/names.go`: Added carrier-aware room ID generation
- `internal/provider/wbstream/api.go`: Added exported WBStream room creation helper
- `docs/LIFETIME_FEATURE.md`: NEW - Feature documentation

## Backward Compatibility

- The feature is fully backward compatible
- If `-lifetime` is not specified or is 0, room rotation is disabled
- Existing code continues to work without modifications
- Clients automatically handle service messages if they receive them
