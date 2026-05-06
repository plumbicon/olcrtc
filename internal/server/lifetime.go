package server

import (
	"context"
	"encoding/json"
	"time"

	"github.com/openlibrecommunity/olcrtc/internal/logger"
	"github.com/openlibrecommunity/olcrtc/internal/names"
	"github.com/openlibrecommunity/olcrtc/internal/provider/wbstream"
)

// lifetimeManager periodically rotates the room every lifetime seconds
func (s *Server) lifetimeManager(ctx context.Context, cancel context.CancelFunc) {
	ticker := time.NewTicker(time.Duration(s.lifetime) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.rotateRoom(ctx, cancel)
		}
	}
}

// rotateRoom sends a service message to all clients and reconnects to a new room
func (s *Server) rotateRoom(ctx context.Context, cancel context.CancelFunc) {
	s.rotateMu.Lock()
	defer s.rotateMu.Unlock()

	logger.Infof("Rotating room (lifetime %d seconds expired)", s.lifetime)

	newRoomID, err := s.createRotationRoom(ctx)
	if err != nil {
		logger.Errorf("Failed to create rotation room: %v", err)
		cancel()
		return
	}
	logger.Infof("New room ID: %s", newRoomID)

	// Send service message to all connected clients
	s.broadcastServiceMessage(newRoomID)

	// Wait longer for clients to receive and process the message
	// This gives clients time to read from the old session before we close it
	time.Sleep(2 * time.Second)

	// Disconnect from current room
	s.sessMu.Lock()
	if s.session != nil {
		_ = s.session.Close()
		s.session = nil
	}
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}
	s.sessMu.Unlock()

	if s.ln != nil {
		_ = s.ln.Close()
	}

	// Wait a bit for the carrier to fully close the old room
	time.Sleep(1 * time.Second)

	// Reconnect to new room
	newRoomURL := s.buildNewRoomURL(newRoomID)
	logger.Infof("Reconnecting to new room: %s", newRoomURL)

	if err := s.bringUpLink(
		ctx, s.linkName, s.transportName, s.carrierName, newRoomURL, cancel,
		s.videoWidth, s.videoHeight, s.videoFPS, s.videoBitrate, s.videoHW,
		s.videoQRSize, s.videoQRRecovery, s.videoCodec, s.videoTileModule, s.videoTileRS,
		s.vp8FPS, s.vp8BatchSize,
		s.seiFPS, s.seiBatchSize, s.seiFragmentSize, s.seiAckTimeoutMS,
	); err != nil {
		logger.Errorf("Failed to reconnect to new room: %v", err)
		cancel()
	}
}

// broadcastServiceMessage sends a service message to all connected streams
func (s *Server) broadcastServiceMessage(newRoomID string) {
	s.sessMu.RLock()
	sess := s.session
	s.sessMu.RUnlock()

	if sess == nil {
		return
	}

	msg := ServiceMessage{
		Type:   "room_rotate",
		RoomID: newRoomID,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		logger.Warnf("Failed to marshal service message: %v", err)
		return
	}

	// Open a new stream to send the service message
	stream, err := sess.OpenStream()
	if err != nil {
		logger.Warnf("Failed to open stream for service message: %v", err)
		return
	}
	defer func() { _ = stream.Close() }()

	// Send the message
	if _, err := stream.Write(msgBytes); err != nil {
		logger.Warnf("Failed to send service message: %v", err)
		return
	}

	logger.Infof("Broadcasted service message: %s", string(msgBytes))
}

func (s *Server) createRotationRoom(ctx context.Context) (string, error) {
	if s.carrierName == "wbstream" {
		return wbstream.CreateRoom(ctx, names.Generate())
	}
	return names.GenerateForCarrier(s.carrierName), nil
}

// buildNewRoomURL builds a new room URL with the new room ID.
func (s *Server) buildNewRoomURL(newRoomID string) string {
	switch s.carrierName {
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
