package client

import (
	"context"
	"encoding/json"
	"time"

	"github.com/openlibrecommunity/olcrtc/internal/logger"
	"github.com/xtaci/smux"
)

// serviceMessageListener listens for service messages on a dedicated stream
func (c *Client) serviceMessageListener(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		c.sessMu.RLock()
		sess := c.session
		c.sessMu.RUnlock()

		if sess == nil || sess.IsClosed() {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Try to accept a stream for service messages
		stream, err := sess.AcceptStream()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				logger.Debugf("AcceptStream failed: %v", err)
				time.Sleep(100 * time.Millisecond)
				continue
			}
		}

		// Read service message
		c.readServiceMessage(ctx, stream)
	}
}

// readServiceMessage reads a service message from a stream
func (c *Client) readServiceMessage(ctx context.Context, stream *smux.Stream) {
	defer func() { _ = stream.Close() }()

	// Set read deadline
	_ = stream.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Read the message
	buf := make([]byte, 4096)
	n, err := stream.Read(buf)
	if err != nil {
		logger.Debugf("Failed to read service message: %v", err)
		return
	}

	if n > 0 {
		_ = c.handleServiceMessage(buf[:n])
	}
}

// handleServiceMessage processes service messages from the server
func (c *Client) handleServiceMessage(data []byte) error {
	var msg ServiceMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		logger.Debugf("Failed to unmarshal service message: %v", err)
		return nil // Not a service message, ignore
	}

	switch msg.Type {
	case "room_rotate":
		logger.Infof("Received room rotation message, new room ID: %s", msg.RoomID)
		return c.handleRoomRotation(msg.RoomID)
	default:
		logger.Debugf("Unknown service message type: %s", msg.Type)
		return nil
	}
}

// handleRoomRotation handles reconnection to a new room
func (c *Client) handleRoomRotation(newRoomID string) error {
	logger.Infof("Handling room rotation to new room: %s", newRoomID)

	// Build new room URL
	newRoomURL := c.buildNewRoomURL(newRoomID)
	logger.Infof("New room URL: %s", newRoomURL)

	// Close current session
	c.sessMu.Lock()
	if c.session != nil {
		_ = c.session.Close()
		c.session = nil
	}
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	c.sessMu.Unlock()

	if c.ln != nil {
		_ = c.ln.Close()
	}

	// Wait a bit for the old room to fully close on the carrier side
	time.Sleep(1 * time.Second)

	// Reconnect to new room with retry logic
	const maxRetries = 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		if err := c.bringUpLink(
			ctx, c.linkName, c.transportName, c.carrierName, newRoomURL, cancel,
			c.dnsServer, "", 0,
			c.videoWidth, c.videoHeight, c.videoFPS, c.videoBitrate, c.videoHW,
			c.videoQRSize, c.videoQRRecovery, c.videoCodec, c.videoTileModule, c.videoTileRS,
			c.vp8FPS, c.vp8BatchSize,
			c.seiFPS, c.seiBatchSize, c.seiFragmentSize, c.seiAckTimeoutMS,
		); err != nil {
			cancel()
			lastErr = err
			logger.Warnf("Reconnection attempt %d/%d failed: %v", attempt, maxRetries, err)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second)
			}
			continue
		}

		cancel()
		logger.Infof("Successfully reconnected to new room on attempt %d", attempt)
		return nil
	}

	logger.Errorf("Failed to reconnect to new room after %d attempts: %v", maxRetries, lastErr)
	return lastErr
}

// buildNewRoomURL builds a new room URL with the new room ID.
func (c *Client) buildNewRoomURL(newRoomID string) string {
	switch c.carrierName {
	case "telemost":
		return "https://telemost.yandex.ru/j/" + newRoomID
	case "jazz":
		return newRoomID
	case "wbstream":
		return newRoomID
	default:
		return newRoomID
	}
}
