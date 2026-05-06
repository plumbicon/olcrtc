# Room Lifetime Feature

## Overview

The `-lifetime` flag enables automatic room rotation on the server. When enabled, the server will:

1. Create a new room every N seconds
2. Send a service message to all connected clients with the new room ID
3. Disconnect from the current room
4. Connect to the new room

Clients automatically receive the service message and reconnect to the new room without manual intervention. Existing TCP streams through SOCKS can be interrupted during rotation; applications should retry failed requests.

## Usage

### Server

To enable room rotation with a lifetime of 60 seconds:

```bash
olcrtc -mode srv \
  -link direct \
  -transport datachannel \
  -carrier telemost \
  -id <room_id> \
  -key <hex_key> \
  -dns 1.1.1.1:53 \
  -lifetime 60
```

### Client

The client automatically handles room rotation when the server sends a service message:

```bash
olcrtc -mode cnc \
  -link direct \
  -transport datachannel \
  -carrier telemost \
  -id <room_id> \
  -key <hex_key> \
  -dns 1.1.1.1:53 \
  -socks-host 127.0.0.1 \
  -socks-port 1080
```

## How It Works

### Server-Side

1. The server starts a `lifetimeManager` goroutine when `-lifetime` is specified
2. Every N seconds, the timer fires and triggers `rotateRoom()`
3. `rotateRoom()` performs the following steps:
   - Generates a new room ID using `names.GenerateForCarrier()` (carrier-specific format)
   - Broadcasts a service message to all connected clients via a dedicated smux stream
   - Waits 2 seconds for clients to receive the message
   - Closes the current smux session and link
   - Waits 1 second for the carrier to close the old room
   - Reconnects to the new room using `bringUpLink()`

### Client-Side

1. The client starts a `serviceMessageListener` goroutine
2. The listener continuously accepts streams from the server
3. When a service message is received:
   - The message is parsed as JSON
   - If it's a `room_rotate` message, `handleRoomRotation()` is called
4. `handleRoomRotation()` performs the following steps:
   - Closes the current smux session and link
   - Builds the new room URL from the new room ID
   - Reconnects to the new room using `bringUpLink()`
   - Resumes normal operation

## Message Format

Service messages are sent as JSON:

```json
{
  "type": "room_rotate",
  "room_id": "<new_room_id>"
}
```

## Configuration

- `-lifetime <n>`: Room lifetime in seconds (server only, 0 = infinite, default = 0)

## Room ID Format by Carrier

The server automatically generates room IDs in the correct format for each carrier:

### Telemost
- **Format**: Human-readable names
- **Example**: "Иван Петров" (supports Cyrillic)
- **Used in URL**: `https://telemost.yandex.ru/j/Иван%20Петров`
- **Notes**: Telemost supports any UTF-8 characters in room IDs

### Jazz
- **Format**: Human-readable names
- **Example**: "Alice Johnson"
- **Used as**: Direct room ID
- **Notes**: Jazz supports human-readable names for room identification

### WBStream
- **Format**: Room ID returned by the WBStream API
- **Example**: "550e8400-e29b-41d4-a716-446655440000"
- **Used as**: Direct room ID
- **Reason**: The server creates the next WBStream room through the API before reconnecting and sending the ID to the client

## Behavior

- **Server**: If `-lifetime` is 0 or not specified, room rotation is disabled
- **Client**: The client automatically handles room rotation messages regardless of whether `-lifetime` is specified
- **Reconnection**: Clients reconnect automatically; active SOCKS5 streams can be interrupted during rotation
- **Timeout**: Clients have a 30-second timeout per reconnect attempt and retry up to 3 times
- **Carrier-Specific IDs**: Room IDs are automatically generated in the correct format for each carrier

## Example Scenarios

### Scenario 1: 5-Minute Room Rotation

Server:
```bash
olcrtc -mode srv -link direct -transport datachannel -carrier telemost \
  -id myroom -key <key> -dns 1.1.1.1:53 -lifetime 300
```

The server will rotate to a new room every 5 minutes.

### Scenario 2: Continuous Connection with Room Rotation

Client:
```bash
olcrtc -mode cnc -link direct -transport datachannel -carrier telemost \
  -id myroom -key <key> -dns 1.1.1.1:53 \
  -socks-host 127.0.0.1 -socks-port 1080
```

The client will automatically reconnect to new rooms as the server rotates them.

## Logging

When debug logging is enabled (`-debug`), you'll see:

- Server: `Rotating room (lifetime N seconds expired)` and `New room ID: <id>`
- Server: `Broadcasted service message: ...`
- Client: `Received room rotation message, new room ID: <id>`
- Client: `Successfully reconnected to new room`

## Limitations

- Service messages are sent via a dedicated smux stream, which may not reach clients if the connection is unstable
- Clients have a 30-second timeout per reconnect attempt and retry up to 3 times
- The feature is designed for continuous long-lived connections, not for one-off requests
- WBStream requires UUID format for room IDs (no non-ASCII characters in HTTP paths)

## Future Enhancements

- Configurable reconnection timeout
- Support for graceful connection draining before room rotation
- Metrics/monitoring for room rotation events
