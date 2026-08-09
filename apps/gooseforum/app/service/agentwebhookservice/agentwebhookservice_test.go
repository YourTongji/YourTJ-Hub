package agentwebhookservice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/agentInbox"
	"github.com/leancodebox/GooseForum/app/models/forum/agents"
	"github.com/leancodebox/GooseForum/app/models/forum/pageConfig"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/taskQueue"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/forum/userStatistics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/service/agentservice"
)

func setupWebhookTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&userStatistics.Entity{},
		&agents.Entity{},
		&agentInbox.Entity{},
		&taskQueue.Entity{},
		&topics.Entity{},
		&posts.Entity{},
		&pageConfig.Entity{},
	); err != nil {
		t.Fatalf("migrate webhook tables: %v", err)
	}
	conn.Where("1 = 1").Delete(&agentInbox.Entity{})
	conn.Where("1 = 1").Delete(&taskQueue.Entity{})
	conn.Where("1 = 1").Delete(&agents.Entity{})
	conn.Where("1 = 1").Delete(&userStatistics.Entity{})
	conn.Where("1 = 1").Delete(&users.EntityComplete{})
}

type fakeSender struct {
	payload WebhookPayload
	err     error
}

func (f *fakeSender) Send(_ context.Context, _ string, payload WebhookPayload) error {
	f.payload = payload
	return f.err
}

func withSender(sender DeliverySender, fn func()) {
	original := senderFactory
	senderFactory = func() DeliverySender { return sender }
	defer func() { senderFactory = original }()
	fn()
}

func createWebhookAgent(t *testing.T, username, endpoint string) uint64 {
	t.Helper()
	result, err := agentservice.Create(agentservice.CreateParams{Username: username, WebhookEndpoint: endpoint})
	if err != nil {
		t.Fatalf("create agent %s: %v", username, err)
	}
	return result.Agent.UserId
}

func createInboxRow(t *testing.T, agentID, topicID, postID uint64, eventType string) uint64 {
	t.Helper()
	row := agentInbox.Entity{
		AgentId:        agentID,
		TopicId:        topicID,
		PostId:         postID,
		EventType:      eventType,
		ActorId:        1,
		ContentPreview: "preview",
		Status:         agentInbox.StatusUnread,
		DeliveryStatus: agentInbox.DeliveryPending,
	}
	if err := db.Connect().Create(&row).Error; err != nil {
		t.Fatalf("create inbox: %v", err)
	}
	return row.Id
}

func createWebhookTask(t *testing.T, inboxID uint64) uint64 {
	t.Helper()
	payload, _ := json.Marshal(TaskPayload{InboxId: inboxID})
	task := taskQueue.Entity{Type: TaskTypeAgentWebhook, Status: taskQueue.StatusPending, TaskJson: string(payload)}
	if err := db.Connect().Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	return task.Id
}

func mustGetInbox(t *testing.T, id uint64) agentInbox.Entity {
	t.Helper()
	row := agentInbox.GetByID(id)
	if row == nil {
		t.Fatalf("inbox %d missing", id)
	}
	return *row
}

func TestRunTaskDeliversAndMarksDelivered(t *testing.T) {
	setupWebhookTestDB(t)
	agentID := createWebhookAgent(t, "hook-agent-1", "https://example.com/hook")
	inboxID := createInboxRow(t, agentID, 201, 2001, agentInbox.EventTypeTopicPublished)
	taskID := createWebhookTask(t, inboxID)

	fake := &fakeSender{}
	withSender(fake, func() {
		task := mustGetTask(t, taskID)
		if err := RunTask(context.Background(), task); err != nil {
			t.Fatalf("RunTask: %v", err)
		}
	})

	row := mustGetInbox(t, inboxID)
	if row.DeliveryStatus != agentInbox.DeliveryDelivered {
		t.Fatalf("delivery_status = %d, want delivered", row.DeliveryStatus)
	}
	if fake.payload.EventId != inboxID {
		t.Fatalf("eventId = %d, want inbox id %d", fake.payload.EventId, inboxID)
	}
	if fake.payload.EventType != "agent.mention.topic_published" {
		t.Fatalf("eventType = %q", fake.payload.EventType)
	}
	if fake.payload.Topic == nil || fake.payload.Topic.Id != 201 {
		t.Fatalf("topic fact = %#v", fake.payload.Topic)
	}
	if fake.payload.Preview != "preview" {
		t.Fatalf("preview = %q", fake.payload.Preview)
	}
	if strings.Contains(fake.payload.Preview, "token") || strings.Contains(fake.payload.Preview, "hash") {
		t.Fatal("payload must not carry token or hash material")
	}
}

func TestRunTaskDuplicateNonPendingNoOp(t *testing.T) {
	setupWebhookTestDB(t)
	agentID := createWebhookAgent(t, "hook-agent-2", "https://example.com/hook")
	inboxID := createInboxRow(t, agentID, 202, 2002, agentInbox.EventTypePostCreated)
	taskID := createWebhookTask(t, inboxID)
	if err := agentInbox.MarkDelivered(inboxID); err != nil {
		t.Fatalf("mark delivered: %v", err)
	}

	fake := &fakeSender{err: errors.New("must not be called")}
	withSender(fake, func() {
		task := mustGetTask(t, taskID)
		if err := RunTask(context.Background(), task); err != nil {
			t.Fatalf("RunTask: %v", err)
		}
	})
	if fake.payload.EventId != 0 {
		t.Fatal("sender must not run for non-pending inbox")
	}
}

func TestRunTaskMissingInboxNoOp(t *testing.T) {
	setupWebhookTestDB(t)
	taskID := createWebhookTask(t, 999999)
	fake := &fakeSender{err: errors.New("must not be called")}
	withSender(fake, func() {
		task := mustGetTask(t, taskID)
		if err := RunTask(context.Background(), task); err != nil {
			t.Fatalf("RunTask: %v", err)
		}
	})
	if fake.payload.EventId != 0 {
		t.Fatal("sender must not run for missing inbox")
	}
}

func TestRunTaskSkippedWhenAgentUnavailable(t *testing.T) {
	setupWebhookTestDB(t)
	agentID := createWebhookAgent(t, "hook-agent-3", "https://example.com/hook")
	inboxID := createInboxRow(t, agentID, 203, 2003, agentInbox.EventTypePostCreated)
	taskID := createWebhookTask(t, inboxID)

	t.Run("disabled agent", func(t *testing.T) {
		if err := agentservice.Disable(agentID); err != nil {
			t.Fatalf("disable agent: %v", err)
		}
		fake := &fakeSender{err: errors.New("must not be called")}
		withSender(fake, func() {
			task := mustGetTask(t, taskID)
			if err := RunTask(context.Background(), task); err != nil {
				t.Fatalf("RunTask: %v", err)
			}
		})
		row := mustGetInbox(t, inboxID)
		if row.DeliveryStatus != agentInbox.DeliverySkipped {
			t.Fatalf("delivery_status = %d, want skipped", row.DeliveryStatus)
		}
		if fake.payload.EventId != 0 {
			t.Fatal("sender must not run for disabled agent")
		}
	})

	t.Run("no endpoint", func(t *testing.T) {
		noEndpointID := createWebhookAgent(t, "hook-agent-3b", "")
		inboxID2 := createInboxRow(t, noEndpointID, 204, 2004, agentInbox.EventTypePostCreated)
		taskID2 := createWebhookTask(t, inboxID2)
		fake := &fakeSender{err: errors.New("must not be called")}
		withSender(fake, func() {
			task := mustGetTask(t, taskID2)
			if err := RunTask(context.Background(), task); err != nil {
				t.Fatalf("RunTask: %v", err)
			}
		})
		row := mustGetInbox(t, inboxID2)
		if row.DeliveryStatus != agentInbox.DeliverySkipped {
			t.Fatalf("delivery_status = %d, want skipped", row.DeliveryStatus)
		}
	})
}

func TestRunTaskMalformedPayloadNoRetry(t *testing.T) {
	setupWebhookTestDB(t)
	task := taskQueue.Entity{Type: TaskTypeAgentWebhook, Status: taskQueue.StatusPending, TaskJson: `{not-json`}
	if err := db.Connect().Create(&task).Error; err != nil {
		t.Fatalf("create malformed task: %v", err)
	}
	if err := RunTask(context.Background(), &task); err != nil {
		t.Fatalf("RunTask must swallow malformed payload, got %v", err)
	}
}

func TestRunTaskRetryScheduleAndTerminalFailure(t *testing.T) {
	setupWebhookTestDB(t)
	agentID := createWebhookAgent(t, "hook-agent-4", "https://example.com/hook")
	inboxID := createInboxRow(t, agentID, 205, 2005, agentInbox.EventTypePostCreated)
	taskID := createWebhookTask(t, inboxID)

	failing := &fakeSender{err: errors.New("boom raw error with https://secret.example/x?token=abc")}
	withSender(failing, func() {
		task := mustGetTask(t, taskID)

		// Attempt 1: attempts=1, run_at ~ +1m, worker retrying
		if err := RunTask(context.Background(), task); err == nil {
			t.Fatal("first failure must return error for worker retry")
		}
		row := mustGetInbox(t, inboxID)
		if row.Attempts != 1 || row.DeliveryStatus != agentInbox.DeliveryPending {
			t.Fatalf("after attempt 1 = %#v", row)
		}
		if strings.Contains(row.LastError, "secret.example") || strings.Contains(row.LastError, "token=abc") {
			t.Fatalf("last_error leaks URL: %q", row.LastError)
		}
		taskRow := mustGetTask(t, taskID)
		if taskRow.RunAt == nil || taskRow.RunAt.Sub(time.Now()) < 50*time.Second || taskRow.RunAt.Sub(time.Now()) > 70*time.Second {
			t.Fatalf("run_at after attempt 1 = %v, want ~1m", taskRow.RunAt)
		}

		// Attempt 2: attempts=2, run_at ~ +5m
		if err := RunTask(context.Background(), task); err == nil {
			t.Fatal("second failure must return error for worker retry")
		}
		row = mustGetInbox(t, inboxID)
		if row.Attempts != 2 {
			t.Fatalf("attempts after 2 = %d", row.Attempts)
		}
		taskRow = mustGetTask(t, taskID)
		if taskRow.RunAt == nil || taskRow.RunAt.Sub(time.Now()) < 4*time.Minute+50*time.Second || taskRow.RunAt.Sub(time.Now()) > 5*time.Minute+10*time.Second {
			t.Fatalf("run_at after attempt 2 = %v, want ~5m", taskRow.RunAt)
		}

		// Attempt 3: terminal failed, task completes
		if err := RunTask(context.Background(), task); err != nil {
			t.Fatalf("third failure must return nil, got %v", err)
		}
		row = mustGetInbox(t, inboxID)
		if row.Attempts != 3 || row.DeliveryStatus != agentInbox.DeliveryFailed {
			t.Fatalf("after attempt 3 = %#v, want failed terminal", row)
		}
	})
}

func TestRunTaskSuccessAfterRetry(t *testing.T) {
	setupWebhookTestDB(t)
	agentID := createWebhookAgent(t, "hook-agent-5", "https://example.com/hook")
	inboxID := createInboxRow(t, agentID, 206, 2006, agentInbox.EventTypeTopicUpdated)
	taskID := createWebhookTask(t, inboxID)

	attempts := 0
	flaky := &flakySender{failFirst: 1, attempts: &attempts}
	withSender(flaky, func() {
		task := mustGetTask(t, taskID)
		if err := RunTask(context.Background(), task); err == nil {
			t.Fatal("first attempt must fail")
		}
		if err := RunTask(context.Background(), task); err != nil {
			t.Fatalf("second attempt must succeed: %v", err)
		}
	})
	row := mustGetInbox(t, inboxID)
	if row.DeliveryStatus != agentInbox.DeliveryDelivered || row.Attempts != 1 {
		t.Fatalf("final state = %#v", row)
	}
}

type flakySender struct {
	failFirst int
	attempts  *int
}

func (f *flakySender) Send(_ context.Context, _ string, _ WebhookPayload) error {
	*f.attempts++
	if *f.attempts <= f.failFirst {
		return errors.New("webhook request timed out")
	}
	return nil
}

func TestValidateEndpointMatrix(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		wantErr  bool
	}{
		{"empty", "", true},
		{"missing scheme", "example.com/hook", true},
		{"ftp scheme", "ftp://example.com/hook", true},
		{"userinfo", "https://user:pass@example.com/hook", true},
		{"fragment", "https://example.com/hook#frag", true},
		{"localhost", "https://localhost/hook", true},
		{"subdomain localhost", "https://foo.localhost/hook", true},
		{"local tld", "https://foo.local/hook", true},
		{"internal tld", "https://foo.internal/hook", true},
		{"loopback literal", "https://127.0.0.1/hook", true},
		{"private literal", "http://192.168.1.1/hook", true},
		{"link-local literal", "http://169.254.1.1/hook", true},
		{"ULA literal", "http://[fd00::1]/hook", true},
		{"multicast literal", "http://224.0.0.1/hook", true},
		{"unspecified literal", "http://0.0.0.0/hook", true},
		{"CGNAT literal", "http://100.64.0.1/hook", true},
		{"documentation literal", "http://192.0.2.1/hook", true},
		{"ipv6 loopback literal", "http://[::1]/hook", true},
		{"public hostname", "https://example.com/hook", false},
		{"public literal", "https://93.184.216.34/hook", false},
		{"query allowed", "https://example.com/hook?x=1", false},
		{"path allowed", "https://example.com/a/b", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validateEndpoint(tc.endpoint)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateEndpoint(%q) err = %v, wantErr %v", tc.endpoint, err, tc.wantErr)
			}
			if err != nil && tc.endpoint != "" && strings.Contains(err.Error(), tc.endpoint) {
				t.Fatalf("error leaks endpoint: %q", err.Error())
			}
		})
	}
}

type staticResolver struct {
	addrs []net.IPAddr
	err   error
}

func (r staticResolver) LookupIPAddr(_ context.Context, _ string) ([]net.IPAddr, error) {
	return r.addrs, r.err
}

type blockingResolver struct{}

func (blockingResolver) LookupIPAddr(ctx context.Context, _ string) ([]net.IPAddr, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestResolvePinnedRejectsAnyNonPublic(t *testing.T) {
	public := net.ParseIP("93.184.216.34")
	loopback := net.ParseIP("127.0.0.1")
	private := net.ParseIP("10.0.0.1")
	linkLocal := net.ParseIP("169.254.1.1")
	ula := net.ParseIP("fd00::1")
	nat64 := net.ParseIP("64:ff9b::a00:1")
	teredo := net.ParseIP("2001::1")
	orchidV1 := net.ParseIP("2001:10::1")
	orchidV2 := net.ParseIP("2001:20::1")
	sixToFour := net.ParseIP("2002:0a00:0001::")

	tests := []struct {
		name  string
		addrs []net.IPAddr
		ok    bool
	}{
		{"all public", []net.IPAddr{{IP: public}}, true},
		{"mixed public and private", []net.IPAddr{{IP: public}, {IP: private}}, false},
		{"private only", []net.IPAddr{{IP: private}}, false},
		{"loopback only", []net.IPAddr{{IP: loopback}}, false},
		{"link local only", []net.IPAddr{{IP: linkLocal}}, false},
		{"ULA only", []net.IPAddr{{IP: ula}}, false},
		{"NAT64 translation only", []net.IPAddr{{IP: nat64}}, false},
		{"Teredo only", []net.IPAddr{{IP: teredo}}, false},
		{"ORCHIDv1 only", []net.IPAddr{{IP: orchidV1}}, false},
		{"ORCHIDv2 only", []net.IPAddr{{IP: orchidV2}}, false},
		{"6to4 only", []net.IPAddr{{IP: sixToFour}}, false},
		{"empty", nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolvePinned(context.Background(), staticResolver{addrs: tc.addrs}, "example.com")
			if tc.ok && (err != nil || got == "") {
				t.Fatalf("resolvePinned = %q, %v; want success", got, err)
			}
			if !tc.ok && err == nil {
				t.Fatalf("resolvePinned = %q, want rejection", got)
			}
		})
	}

	t.Run("dns error", func(t *testing.T) {
		_, err := resolvePinned(context.Background(), staticResolver{err: errors.New("dns down")}, "example.com")
		if err == nil || err.Error() != "webhook DNS resolution failed" {
			t.Fatalf("dns error = %v", err)
		}
	})
}

func TestWebhookSenderPinnedDialingAndHeaders(t *testing.T) {
	var receivedAddr string
	var receivedReq *http.Request
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReq = r
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := &webhookSender{
		resolver: staticResolver{addrs: []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}},
		dialContext: func(_ context.Context, network, addr string) (net.Conn, error) {
			receivedAddr = addr
			// Route the pinned dial to the local test server.
			conn, err := net.Dial("tcp", strings.TrimPrefix(server.URL, "http://"))
			return conn, err
		},
		timeout: 2 * time.Second,
	}
	payload := WebhookPayload{EventId: 42, EventType: "agent.mention.topic_published"}
	err := sender.Send(context.Background(), "http://example.com/hook", payload)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if receivedAddr != "93.184.216.34:80" {
		t.Fatalf("dialed addr = %q, want pinned public IP:80", receivedAddr)
	}
	if receivedReq == nil {
		t.Fatal("server did not receive a request")
	}
	if receivedReq.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("content-type = %q", receivedReq.Header.Get("Content-Type"))
	}
	if receivedReq.Header.Get("X-Yourtj-Event-Id") != "42" {
		t.Fatalf("event id header = %q", receivedReq.Header.Get("X-Yourtj-Event-Id"))
	}
	if receivedReq.Header.Get("X-Yourtj-Event-Type") != "agent.mention.topic_published" {
		t.Fatalf("event type header = %q", receivedReq.Header.Get("X-Yourtj-Event-Type"))
	}
	if receivedReq.Header.Get("User-Agent") != userAgent {
		t.Fatalf("user-agent = %q", receivedReq.Header.Get("User-Agent"))
	}
	if receivedReq.Host != "example.com" {
		t.Fatalf("host = %q, want original hostname preserved", receivedReq.Host)
	}
	if !strings.Contains(string(receivedBody), `"eventId":42`) {
		t.Fatalf("body = %s", receivedBody)
	}
}

func TestWebhookSenderNoRedirectFollowed(t *testing.T) {
	redirects := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirects++
		w.Header().Set("Location", "/elsewhere")
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()

	sender := &webhookSender{
		resolver: staticResolver{addrs: []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}},
		dialContext: func(_ context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("tcp", strings.TrimPrefix(server.URL, "http://"))
		},
		timeout: 2 * time.Second,
	}
	err := sender.Send(context.Background(), "http://example.com/hook", WebhookPayload{EventId: 1})
	if err == nil || !strings.HasPrefix(err.Error(), "webhook returned HTTP 302") {
		t.Fatalf("Send = %v, want 302 failure", err)
	}
	if redirects != 1 {
		t.Fatalf("server hits = %d, want 1 (redirect not followed)", redirects)
	}
}

func TestWebhookSenderNon2xxAndBodyTooLarge(t *testing.T) {
	t.Run("500 failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()
		sender := &webhookSender{
			resolver: staticResolver{addrs: []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}},
			dialContext: func(_ context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("tcp", strings.TrimPrefix(server.URL, "http://"))
			},
			timeout: 2 * time.Second,
		}
		err := sender.Send(context.Background(), "http://example.com/hook", WebhookPayload{EventId: 1})
		if err == nil || err.Error() != "webhook returned HTTP 500" {
			t.Fatalf("Send = %v", err)
		}
	})

	t.Run("body over 64KB", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(bytes.Repeat([]byte("x"), maxResponseBody+1024))
		}))
		defer server.Close()
		sender := &webhookSender{
			resolver: staticResolver{addrs: []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}},
			dialContext: func(_ context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("tcp", strings.TrimPrefix(server.URL, "http://"))
			},
			timeout: 2 * time.Second,
		}
		err := sender.Send(context.Background(), "http://example.com/hook", WebhookPayload{EventId: 1})
		if err == nil || err.Error() != "webhook response body too large" {
			t.Fatalf("Send = %v", err)
		}
	})
}

func TestWebhookSenderTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	sender := &webhookSender{
		resolver: staticResolver{addrs: []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}},
		dialContext: func(_ context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("tcp", strings.TrimPrefix(server.URL, "http://"))
		},
		timeout: 100 * time.Millisecond,
	}
	start := time.Now()
	err := sender.Send(context.Background(), "http://example.com/hook", WebhookPayload{EventId: 1})
	if err == nil || err.Error() != "webhook request timed out" {
		t.Fatalf("Send = %v, want timeout", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("timeout took %v, want bounded by client timeout", elapsed)
	}
}

func TestWebhookSenderTimeoutIncludesDNSResolution(t *testing.T) {
	sender := &webhookSender{
		resolver: blockingResolver{},
		timeout:  50 * time.Millisecond,
	}
	start := time.Now()
	err := sender.Send(context.Background(), "https://example.com/hook", WebhookPayload{EventId: 1})
	if err == nil || err.Error() != "webhook request timed out" {
		t.Fatalf("Send = %v, want DNS timeout", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("DNS timeout took %v, want bounded by sender timeout", elapsed)
	}
}

func TestWebhookSenderTLSVerificationFailure(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := &webhookSender{
		resolver: staticResolver{addrs: []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}},
		dialContext: func(_ context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("tcp", strings.TrimPrefix(server.URL, "https://"))
		},
		timeout: 2 * time.Second,
	}
	err := sender.Send(context.Background(), "https://example.com/hook", WebhookPayload{EventId: 1})
	if err == nil || err.Error() != "webhook TLS verification failed" {
		t.Fatalf("Send = %v, want TLS verification failure", err)
	}
}

func TestSanitizeErrorNeverLeaksURL(t *testing.T) {
	raw := []error{
		errors.New("boom https://secret.example/x?token=abc"),
		errors.New("Get \"https://secret.example/x\": dial tcp: connection refused"),
		errors.New("webhook returned HTTP 500"),
		errors.New("webhook request timed out"),
	}
	for _, err := range raw {
		sanitized := sanitizeError(err)
		if strings.Contains(sanitized, "secret.example") || strings.Contains(sanitized, "token=abc") {
			t.Fatalf("sanitizeError(%q) leaks URL: %q", err, sanitized)
		}
		if strings.Contains(sanitized, "http://") || strings.Contains(sanitized, "https://") {
			t.Fatalf("sanitizeError(%q) leaks URL: %q", err, sanitized)
		}
	}
}

func mustGetTask(t *testing.T, id uint64) *taskQueue.Entity {
	t.Helper()
	task, err := taskQueue.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID(%d) error = %v", id, err)
	}
	return &task
}
