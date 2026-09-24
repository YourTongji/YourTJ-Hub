package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/authsessionservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/realtimeservice"
	"github.com/gin-gonic/gin"
)

const (
	eventHeartbeat  = 15 * time.Second
	eventWriteLimit = 5 * time.Second
	eventMaxAge     = time.Hour
)

// StreamEvents sends owner-scoped invalidation hints. The first hello frame
// always requires REST reconciliation, including after a dropped connection.
func StreamEvents(c *gin.Context) {
	streamEvents(c, realtimeservice.DefaultHub, authsessionservice.CheckStreamToken, eventHeartbeat)
}

func streamEvents(c *gin.Context, hub *realtimeservice.Hub, check func(context.Context, string) (bool, error), heartbeat time.Duration) {
	userID := c.GetUint64("userId")
	if userID == 0 {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	token := c.Writer.Header().Get("New-Token")
	if token == "" {
		token = jwtopt.GetGinAccessToken(c)
	}
	valid, err := check(c.Request.Context(), token)
	if err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	if !valid {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	sub, err := hub.Subscribe(userID)
	if errors.Is(err, realtimeservice.ErrConnectionLimit) {
		c.Header("Retry-After", "5")
		c.AbortWithStatus(http.StatusTooManyRequests)
		return
	}
	if err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	defer sub.Close()

	controller := http.NewResponseController(c.Writer)
	// The normal server WriteTimeout is 10s. Disable it for the idle lifetime
	// and arm a short, bounded deadline around each actual network write.
	if err := controller.SetWriteDeadline(time.Time{}); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("X-Accel-Buffering", "no")
	c.Header("X-Content-Type-Options", "nosniff")
	if err := writeStreamFrame(c, controller, "hello", map[string]any{
		"version": 1, "heartbeatSeconds": int(heartbeat / time.Second), "resync": true,
		"capabilities": map[string]bool{"visibleRead": true},
	}); err != nil {
		return
	}
	beats := time.NewTicker(heartbeat)
	defer beats.Stop()
	maxAge := time.NewTimer(eventMaxAge)
	defer maxAge.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-sub.Done():
			return
		case <-maxAge.C:
			// A fresh handshake renews a near-expiry session token in headers.
			return
		case event := <-sub.Events():
			select {
			case <-sub.Done():
				return
			default:
			}
			if err := writeStreamFrame(c, controller, event.Type, streamEventData(event)); err != nil {
				return
			}
		case <-beats.C:
			valid, err := check(c.Request.Context(), token)
			if err != nil {
				slog.Warn("realtime session recheck failed", "err", err)
				return
			}
			if !valid {
				_ = writeStreamFrame(c, controller, "session.invalidated", map[string]any{})
				return
			}
			if err := writeStreamComment(c, controller, "ping"); err != nil {
				return
			}
		}
	}
}

func streamEventData(event realtimeservice.Event) map[string]any {
	data := make(map[string]any, 2)
	if event.ConvID != 0 {
		data["convId"] = event.ConvID
	}
	if event.Change != "" {
		data["change"] = event.Change
	}
	return data
}

func writeStreamFrame(c *gin.Context, controller *http.ResponseController, name string, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return writeStreamBytes(c, controller, []byte("event: "+name+"\ndata: "+string(encoded)+"\n\n"))
}

func writeStreamComment(c *gin.Context, controller *http.ResponseController, comment string) error {
	return writeStreamBytes(c, controller, []byte(": "+strings.ReplaceAll(comment, "\n", "")+"\n\n"))
}

func writeStreamBytes(c *gin.Context, controller *http.ResponseController, frame []byte) error {
	if err := controller.SetWriteDeadline(time.Now().Add(eventWriteLimit)); err != nil {
		return err
	}
	if _, err := c.Writer.Write(frame); err != nil {
		return err
	}
	if err := controller.Flush(); err != nil {
		return err
	}
	return controller.SetWriteDeadline(time.Time{})
}
