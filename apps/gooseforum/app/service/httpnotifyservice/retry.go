package httpnotifyservice

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

// TaskTypeDeliveryRetry is the prefix consumed by the HTTP notification retry worker.
const TaskTypeDeliveryRetry = "http-notify.delivery-retry."

type deliveryRetryTask struct {
	EndpointKey string `json:"endpointKey"`
	ChannelType string `json:"channelType"`
	Event       string `json:"event"`
	Timestamp   int64  `json:"timestamp"`
	Body        []byte `json:"body"`
	DedupeKey   string `json:"dedupeKey,omitempty"`
	ExpiresAt   int64  `json:"expiresAt"`
}

func enqueueDeliveryRetry(endpoint pageConfig.HttpNotifyEndpoint, event string, timestamp int64, body []byte, dedupeKey string) {
	payload, err := json.Marshal(deliveryRetryTask{
		EndpointKey: endpointKey(endpoint),
		ChannelType: pageConfig.NormalizeHttpNotifyChannel(endpoint.ChannelType),
		Event:       event,
		Timestamp:   timestamp,
		Body:        body,
		DedupeKey:   dedupeKey,
		ExpiresAt:   timestamp + int64(dedupeTTL.Seconds()),
	})
	if err != nil {
		slog.Error("httpnotify: encode retry task failed", "event", event, "err", err)
		return
	}
	if err := taskQueue.Create(&taskQueue.Entity{
		Type:     TaskTypeDeliveryRetry + strconv.FormatInt(time.Now().UnixNano(), 10),
		Status:   taskQueue.StatusPending,
		TaskJson: string(payload),
	}); err != nil {
		slog.Error("httpnotify: enqueue retry task failed", "event", event, "err", err)
	}
}

// RunDeliveryRetryTask retries one failed endpoint delivery through taskQueue's bounded retry policy.
func RunDeliveryRetryTask(_ context.Context, task *taskQueue.Entity) error {
	var payload deliveryRetryTask
	if err := json.Unmarshal([]byte(task.TaskJson), &payload); err != nil {
		slog.Warn("httpnotify: malformed retry task", "taskId", task.Id, "err", err)
		return nil
	}
	if payload.EndpointKey == "" || payload.ChannelType == "" || payload.Event == "" || payload.Timestamp <= 0 || payload.ExpiresAt <= 0 || len(payload.Body) == 0 {
		slog.Warn("httpnotify: incomplete retry task", "taskId", task.Id)
		return nil
	}
	if payload.ExpiresAt > 0 && time.Now().Unix() > payload.ExpiresAt {
		return nil
	}
	config := hotdataserve.GetHttpNotifyConfigCache()
	if !config.Enabled {
		return nil
	}
	var endpoint pageConfig.HttpNotifyEndpoint
	found := false
	for _, candidate := range config.Endpoints {
		if endpointKey(candidate) == payload.EndpointKey {
			endpoint = candidate
			found = true
			break
		}
	}
	if !found || !endpointAccepts(endpoint, payload.Event) ||
		pageConfig.NormalizeHttpNotifyChannel(endpoint.ChannelType) != payload.ChannelType {
		return nil
	}

	claimAt := time.Unix(payload.Timestamp, 0)
	dedupeKey := ""
	if payload.DedupeKey != "" {
		dedupeKey = payload.EndpointKey + "\x00" + payload.DedupeKey
		if !recentDeliveries.claim(dedupeKey, claimAt) {
			return nil
		}
	}
	if !deliver(endpoint, channelFor(endpoint), payload.Event, time.Now().Unix(), payload.Body) {
		if dedupeKey != "" {
			recentDeliveries.release(dedupeKey, claimAt)
		}
		return errors.New("HTTP notification delivery failed")
	}
	return nil
}

// CleanupTerminalTasks deletes retry task history after the retained inspection window.
func CleanupTerminalTasks(before time.Time, limit int) (int64, error) {
	return taskQueue.DeleteTerminalByTypePrefix(TaskTypeDeliveryRetry,
		[]int{int(taskQueue.StatusSuccess), int(taskQueue.StatusFailed)}, before, limit)
}
