# Usage Examples: Room Lifetime Feature

## Example 1: Basic Room Rotation (10 seconds)

### Terminal 1 - Start Server

```bash
# Generate a 32-byte hex key
KEY=$(openssl rand -hex 32)
echo "Key: $KEY"

# Start server with 10-second room lifetime
./olcrtc -mode srv \
  -link direct \
  -transport datachannel \
  -carrier telemost \
  -id test-room \
  -key $KEY \
  -dns 1.1.1.1:53 \
  -lifetime 10 \
  -debug
```

Expected output:
```
Connecting link via direct/datachannel/telemost...
Link connected
Rotating room (lifetime 10 seconds expired)
New room ID: <generated_id>
Broadcasted service message: {"type":"room_rotate","room_id":"<generated_id>"}
Reconnecting to new room: https://telemost.yandex.ru/j/<generated_id>
Link connected
```

### Terminal 2 - Start Client

```bash
./olcrtc -mode cnc \
  -link direct \
  -transport datachannel \
  -carrier telemost \
  -id test-room \
  -key $KEY \
  -dns 1.1.1.1:53 \
  -socks-host 127.0.0.1 \
  -socks-port 1080 \
  -debug
```

Expected output:
```
SOCKS5 server listening on 127.0.0.1:1080
Received room rotation message, new room ID: <generated_id>
Handling room rotation to new room: <generated_id>
New room URL: https://telemost.yandex.ru/j/<generated_id>
Successfully reconnected to new room
```

### Terminal 3 - Test Connection

```bash
# Use curl with SOCKS5 proxy to test the tunnel
curl -x socks5://127.0.0.1:1080 https://example.com
```

New SOCKS5 connections should continue working after each rotation. A request that is active during the room switch can fail and should be retried.

---

## Example 2: Long-Lived Connection with 5-Minute Rotation

### Server

```bash
KEY=$(openssl rand -hex 32)

./olcrtc -mode srv \
  -link direct \
  -transport datachannel \
  -carrier telemost \
  -id long-lived-room \
  -key $KEY \
  -dns 1.1.1.1:53 \
  -lifetime 300
```

The server will rotate to a new room every 5 minutes.

### Client

```bash
./olcrtc -mode cnc \
  -link direct \
  -transport datachannel \
  -carrier telemost \
  -id long-lived-room \
  -key $KEY \
  -dns 1.1.1.1:53 \
  -socks-host 127.0.0.1 \
  -socks-port 1080
```

The client will automatically reconnect to new rooms as they're created.

---

## Example 3: Multiple Clients with Room Rotation

### Terminal 1 - Start Server

```bash
KEY=$(openssl rand -hex 32)

./olcrtc -mode srv \
  -link direct \
  -transport datachannel \
  -carrier telemost \
  -id multi-client-room \
  -key $KEY \
  -dns 1.1.1.1:53 \
  -lifetime 30
```

### Terminal 2 - Start Client 1

```bash
./olcrtc -mode cnc \
  -link direct \
  -transport datachannel \
  -carrier telemost \
  -id multi-client-room \
  -key $KEY \
  -dns 1.1.1.1:53 \
  -socks-host 127.0.0.1 \
  -socks-port 1080
```

### Terminal 3 - Start Client 2

```bash
./olcrtc -mode cnc \
  -link direct \
  -transport datachannel \
  -carrier telemost \
  -id multi-client-room \
  -key $KEY \
  -dns 1.1.1.1:53 \
  -socks-host 127.0.0.1 \
  -socks-port 1081
```

### Terminal 4 - Test Connections

```bash
# Test client 1
curl -x socks5://127.0.0.1:1080 https://example.com

# Test client 2
curl -x socks5://127.0.0.1:1081 https://example.com
```

Both clients will automatically reconnect to new rooms as the server rotates them.

---

## Example 4: Using with Different Carriers

### Jazz Carrier

```bash
KEY=$(openssl rand -hex 32)

# Server
./olcrtc -mode srv \
  -link direct \
  -transport datachannel \
  -carrier jazz \
  -id any \
  -key $KEY \
  -dns 1.1.1.1:53 \
  -lifetime 60

# Client
./olcrtc -mode cnc \
  -link direct \
  -transport datachannel \
  -carrier jazz \
  -id <room-id-from-server-log> \
  -key $KEY \
  -dns 1.1.1.1:53 \
  -socks-host 127.0.0.1 \
  -socks-port 1080
```

### WBStream Carrier

```bash
KEY=$(openssl rand -hex 32)
ROOM_ID="any"

# Server
./olcrtc -mode srv \
  -link direct \
  -transport datachannel \
  -carrier wbstream \
  -id $ROOM_ID \
  -key $KEY \
  -dns 1.1.1.1:53 \
  -lifetime 60

# Client
./olcrtc -mode cnc \
  -link direct \
  -transport datachannel \
  -carrier wbstream \
  -id <room-id-from-server-log> \
  -key $KEY \
  -dns 1.1.1.1:53 \
  -socks-host 127.0.0.1 \
  -socks-port 1080
```

---

## Example 5: Monitoring Room Rotations

### Server with Debug Logging

```bash
KEY=$(openssl rand -hex 32)

./olcrtc -mode srv \
  -link direct \
  -transport datachannel \
  -carrier telemost \
  -id monitored-room \
  -key $KEY \
  -dns 1.1.1.1:53 \
  -lifetime 20 \
  -debug 2>&1 | grep -E "(Rotating|New room|Broadcasted|Reconnecting)"
```

This will show only the room rotation-related messages.

### Client with Debug Logging

```bash
./olcrtc -mode cnc \
  -link direct \
  -transport datachannel \
  -carrier telemost \
  -id monitored-room \
  -key $KEY \
  -dns 1.1.1.1:53 \
  -socks-host 127.0.0.1 \
  -socks-port 1080 \
  -debug 2>&1 | grep -E "(Received|Handling|Successfully)"
```

---

## Troubleshooting

### Issue: Client doesn't reconnect to new room

**Solution**:
- Check that both server and client are using the same carrier and key
- Verify network connectivity between server and client
- Check logs for errors during reconnection

### Issue: SOCKS5 connections drop during room rotation

**Solution**:
- This can happen during rotation because the underlying smux session and link are replaced
- Applications should implement retry logic for dropped connections
- Consider increasing the room lifetime if rotations are too frequent

### Issue: "Failed to open stream for service message"

**Solution**:
- This may indicate network instability
- The server will retry on the next rotation
- Check network connectivity and firewall rules

---

## Performance Considerations

1. **Room Rotation Frequency**:
   - Too frequent (< 10 seconds): May cause excessive reconnections
   - Recommended: 60-300 seconds for stable operation

2. **Client Timeout**:
   - Clients have a 30-second timeout per reconnect attempt
   - The client retries up to 3 times
   - Ensure network latency is < 10 seconds for reliable reconnection

3. **Multiple Clients**:
   - Each client reconnects independently
   - Server broadcasts to all connected clients simultaneously

---

## Integration with Existing Applications

### Python Example

```python
import socket
import socks

# Configure SOCKS5 proxy
socks.set_default_proxy(socks.SOCKS5, "127.0.0.1", 1080)
socket.socket = socks.socksocket

# Your existing code continues to work
import requests
response = requests.get("https://example.com")
print(response.text)
```

The connection will automatically handle room rotations transparently.

### Node.js Example

```javascript
const SocksClient = require('socks').SocksClient;

const options = {
  proxy: {
    host: '127.0.0.1',
    port: 1080,
    type: 5
  },
  command: 'connect',
  destination: {
    host: 'example.com',
    port: 443
  }
};

SocksClient.createConnection(options, (error, socket) => {
  if (error) console.error(error);
  else {
    // Your code here
    socket.write('GET / HTTP/1.1\r\n...');
  }
});
```

Room rotations are handled transparently by the SOCKS5 layer.
