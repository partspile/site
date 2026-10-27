package handler

import (
	"bufio"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/local"
	"github.com/partspile/site/internal/logger"
	"github.com/sasha-s/go-deadlock"
	"github.com/valyala/fasthttp"
)

// SSEEvent represents an SSE event to be sent to a user
type SSEEvent struct {
	Event string // Event name (e.g., "message-count")
	Data  string // Event data (HTML or text)
}

var (
	// userChannels maps userID to a slice of their event channels (multiple tabs/browsers)
	userChannels = make(map[int][]chan SSEEvent)
	// channelMutex protects userChannels map
	channelMutex deadlock.RWMutex
)

// closeSSE closes all SSE connections for a specific user by sending a close event
func closeSSE(userID int) {
	if !local.IsLoggedIn(userID) {
		return
	}
	SendSSEEvent(userID, SSEEvent{
		Event: "close",
		Data:  "",
	})
}

// SendSSEEvent sends an SSE event to all connections for a specific user
// If the user is not connected, the event is dropped
func SendSSEEvent(userID int, event SSEEvent) {
	if !local.IsLoggedIn(userID) {
		return
	}

	channelMutex.RLock()
	defer channelMutex.RUnlock()

	channels, exists := userChannels[userID]
	if !exists || len(channels) == 0 {
		// User not connected, drop event
		return
	}

	// Send event to all user's connections (multiple tabs/browsers)
	for _, ch := range channels {
		// Non-blocking send - if channel is full, drop event
		select {
		case ch <- event:
			// Event sent successfully
		default:
			logger.Warn("SSE channel full, dropping event",
				"userID", userID,
				"event", event.Event)
		}
	}
}

// SSEHandler handles SSE stream for a specific user
func SSEHandler(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	if !local.IsLoggedIn(userID) {
		return fiber.NewError(fiber.StatusUnauthorized, "Authentication required")
	}

	c.Set(fiber.HeaderContentType, "text/event-stream")
	c.Set(fiber.HeaderCacheControl, "no-cache")
	c.Set(fiber.HeaderConnection, "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	// Create new event channel for this connection
	ch := make(chan SSEEvent, config.SSEChannelBufferSize)

	// Register channel for this user
	channelMutex.Lock()
	userChannels[userID] = append(userChannels[userID], ch)
	channelMutex.Unlock()

	logger.Info("SSE connection established", "userID", userID)

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		// Clean up channel when stream writer exits
		defer func() {
			channelMutex.Lock()
			// Remove this channel from user's channels
			channels := userChannels[userID]
			for i, c := range channels {
				if c == ch {
					userChannels[userID] = append(channels[:i], channels[i+1:]...)
					break
				}
			}
			// Remove user entry if no more channels
			if len(userChannels[userID]) == 0 {
				delete(userChannels, userID)
			}
			close(ch)
			channelMutex.Unlock()
			logger.Info("SSE connection closed", "userID", userID)
		}()

		for event := range ch {
			// Send SSE event in proper format
			if event.Event != "" {
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Event, event.Data)
			} else {
				// Default event (no event name specified)
				fmt.Fprintf(w, "data: %s\n\n", event.Data)
			}

			// Flush detects client disconnect
			if err := w.Flush(); err != nil {
				return
			}
		}
	}))

	return nil
}
