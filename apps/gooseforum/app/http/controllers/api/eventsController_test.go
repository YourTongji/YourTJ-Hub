package api

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/realtimeservice"
	"github.com/gin-gonic/gin"
)

func TestEventStreamFlushesPastServerWriteTimeout(t *testing.T) {
	hub := realtimeservice.NewHub(2, 3, 4)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/events", func(c *gin.Context) {
		c.Set("userId", uint64(31))
		streamEvents(c, hub, func(context.Context, string) (bool, error) { return true, nil }, 25*time.Millisecond)
	})
	server := httptest.NewUnstartedServer(engine)
	server.Config.WriteTimeout = 50 * time.Millisecond
	server.Start()
	defer server.Close()
	request, _ := http.NewRequest(http.MethodGet, server.URL+"/events", nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK || !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("status=%d content-type=%q", response.StatusCode, response.Header.Get("Content-Type"))
	}
	reader := bufio.NewReader(response.Body)
	if got := readSSEFrame(t, reader); !strings.Contains(got, "event: hello") || !strings.Contains(got, `"resync":true`) {
		t.Fatalf("first frame=%q", got)
	}
	// A connection left under server.WriteTimeout would be dead before this hint.
	time.Sleep(90 * time.Millisecond)
	hub.Publish(31, realtimeservice.Event{Type: realtimeservice.EventChatChanged, ConvID: 8, Change: "created"})
	for i := 0; i < 10; i++ {
		frame := readSSEFrame(t, reader)
		if strings.Contains(frame, "event: chat.changed") {
			if !strings.Contains(frame, `"convId":8`) {
				t.Fatalf("chat frame=%q", frame)
			}
			return
		}
	}
	t.Fatal("chat hint was not flushed")
}

func TestEventStreamReportsDefinitiveSessionInvalidation(t *testing.T) {
	hub := realtimeservice.NewHub(1, 1, 2)
	var live atomic.Bool
	live.Store(true)
	engine := gin.New()
	engine.GET("/events", func(c *gin.Context) {
		c.Set("userId", uint64(31))
		streamEvents(c, hub, func(context.Context, string) (bool, error) { return live.Load(), nil }, 10*time.Millisecond)
	})
	server := httptest.NewServer(engine)
	defer server.Close()
	response, err := server.Client().Get(server.URL + "/events")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	reader := bufio.NewReader(response.Body)
	if frame := readSSEFrame(t, reader); !strings.Contains(frame, "event: hello") {
		t.Fatalf("frame=%q", frame)
	}
	live.Store(false)
	if frame := readSSEFrame(t, reader); !strings.Contains(frame, "event: session.invalidated") {
		t.Fatalf("frame=%q", frame)
	}
}

func TestEventHubCloseLetsHTTPShutdownFinish(t *testing.T) {
	hub := realtimeservice.NewHub(1, 1, 2)
	engine := gin.New()
	engine.GET("/events", func(c *gin.Context) {
		c.Set("userId", uint64(31))
		streamEvents(c, hub, func(context.Context, string) (bool, error) { return true, nil }, time.Hour)
	})
	server := httptest.NewUnstartedServer(engine)
	server.Start()
	defer server.Close()
	response, err := server.Client().Get(server.URL + "/events")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	readSSEFrame(t, bufio.NewReader(response.Body))
	hub.CloseAll()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := server.Config.Shutdown(ctx); err != nil {
		t.Fatalf("stream blocked shutdown: %v", err)
	}
}

func readSSEFrame(t *testing.T, reader *bufio.Reader) string {
	t.Helper()
	var frame strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read SSE frame: %v", err)
		}
		frame.WriteString(line)
		if line == "\n" {
			return frame.String()
		}
	}
}
