package mailservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/closer"
	paniclog "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/recovery"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/backgroundservice"
)

const (
	MaxRetries    = 3
	RetryInterval = time.Second * 5
	BatchSize     = 10
)

type EmailTask struct {
	To       string `json:"to"`
	Username string `json:"username"`
	Token    string `json:"token"`
	NewEmail string `json:"newEmail,omitempty"`
	Type     string `json:"type"`
	Locale   string `json:"locale,omitempty"`
}

var emailProcessor = struct {
	once   sync.Once
	stopCh chan struct{}
	wg     sync.WaitGroup
}{
	stopCh: make(chan struct{}),
}

// emailTaskTypePrefix is applied to taskQueue type so the email worker only
// consumes its own tasks. The EmailTask.Type payload keeps the legacy value
// ("activation" / "reset_password") for processing compatibility.
const emailTaskTypePrefix = "email."

// AddToQueue stores an email task for background processing.
func AddToQueue(task EmailTask) error {
	taskJson, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("序列化邮件任务失败: %w", err)
	}

	queueTask := &taskQueue.Entity{
		Type:     emailTaskTypePrefix + task.Type,
		Status:   taskQueue.StatusPending,
		TaskJson: string(taskJson),
	}

	if err = taskQueue.Create(queueTask); err != nil {
		slog.Debug("邮件任务写入队列失败", "type", task.Type, "to", task.To, "err", err)
		return err
	}
	slog.Debug("邮件任务写入队列成功", "id", queueTask.Id, "type", task.Type, "to", task.To)
	return nil
}

// StartEmailProcessor starts the background email queue worker.
func StartEmailProcessor() {
	emailProcessor.once.Do(func() {
		closer.RegisterPriorityContext(closer.PriorityProducer, func(context.Context) error {
			return StopEmailProcessor()
		})
		emailProcessor.wg.Go(func() {
			defer paniclog.Recover("mail_processor")
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			if !processPendingEmailTasks(emailProcessor.stopCh) {
				return
			}

			for {
				select {
				case <-ticker.C:
					if !processPendingEmailTasks(emailProcessor.stopCh) {
						return
					}
				case <-emailProcessor.stopCh:
					return
				}
			}
		})
	})
}

// StopEmailProcessor stops the background email queue worker.
func StopEmailProcessor() error {
	select {
	case <-emailProcessor.stopCh:
	default:
		close(emailProcessor.stopCh)
	}
	emailProcessor.wg.Wait()
	return nil
}

func processPendingEmailTasks(stopCh <-chan struct{}) bool {
	for {
		select {
		case <-stopCh:
			return false
		default:
		}

		// 周期回收租约过期的 Running 任务（issue #138）：崩溃 worker 的
		// 任务在 LeaseDuration 后回到 Pending 重新领取；运行中任务靠心跳
		// 续租，不会被误回收。与领取侧共用同一类型谓词，存量历史邮件行
		// 崩溃后同样能被回收。
		if err := taskQueue.RecoverStaleEmailTasks(taskQueue.LeaseDuration); err != nil {
			slog.Error("恢复过期邮件任务失败", "error", err)
		}

		tasks := taskQueue.GetPendingEmailTasks(BatchSize)
		if len(tasks) == 0 {
			return true
		}
		slog.Debug("邮件队列拉取任务", "count", len(tasks))

		for _, task := range tasks {
			slog.Debug("邮件队列开始处理任务", "id", task.Id, "type", task.Type, "status", task.Status, "retryCount", task.RetryCount)
			// 注意 stop 语义：true = stop（与 generic worker 的
			// drainTasks 相反，后者 false = stop）。调用方各自匹配。
			if stop := processClaimedEmailTask(stopCh, task); stop {
				return false
			}
		}
	}
}

// processClaimedEmailTask 处理单个邮件任务：原子领取（CAS）、处理期间心跳
// 续租、执行、收敛租约值并以 fencing 写入终态。返回 stop 表示 stopCh 已
// 关闭，worker 应退出。与 generic worker 的 processTask 结构对齐。
func processClaimedEmailTask(stopCh <-chan struct{}, task *taskQueue.Entity) (stop bool) {
	// 原子领取（issue #138）：pending/retrying → running 的 CAS 更新，
	// 多 worker 并发时只有一个成功，其余跳过，避免重复发送邮件。
	running, claimed, err := taskQueue.ClaimTask(task.Id)
	if err != nil {
		slog.Error("领取任务失败", "id", task.Id, "error", err)
		return false
	}
	if !claimed {
		slog.Debug("任务已被其他 worker 领取", "id", task.Id)
		return false
	}

	// 处理期间心跳续租；租约丢失（任务被回收重领）时取消 ctx 终止处理。
	guard := backgroundservice.NewLeaseGuard(running.LeaseToken)
	ctx, cancel := context.WithCancel(context.Background())
	// 即使 executeClaimedEmail panic（review P1），defer 也会在栈展开时取消
	// ctx：心跳 goroutine 退出，不再续租，任务租约正常过期后可被回收，
	// 避免心跳泄漏导致任务永久卡在 Running。
	defer cancel()
	heartbeatDone := backgroundservice.StartLeaseHeartbeat(ctx, cancel, running.Id, guard)

	// 与 generic worker 一致：传给执行函数的必须是 ClaimTask 重读的已领取
	// 实体（&running），保证读到的 payload/状态与当前租约一致（review C1）。
	emailTask, outcome, sendErr := executeClaimedEmail(&running)

	// 停止心跳并等待退出，再做一次最终续租拿到权威租约值；后续状态
	// 写入以它为 CAS 前置条件（fencing）。若心跳退出瞬间与最后一次
	// 续租交叠，guard 中的租约值可能滞后于 DB，最终续租负责收敛；
	// 续租失败说明租约已被回收，跳过终态写入，避免覆盖新持有者导致
	// 重复发送。
	cancel()
	<-heartbeatDone
	lease := guard.Get()
	if ok, _, token, renewErr := taskQueue.RenewLease(task.Id, lease); renewErr != nil {
		slog.Error("邮件任务最终续租失败", "id", task.Id, "error", renewErr)
	} else if ok {
		lease = token
	} else {
		slog.Warn("邮件任务租约已丢失，跳过终态写入", "id", task.Id)
		return false
	}

	if retrying := writeEmailTaskOutcome(task, &running, emailTask, outcome, sendErr, lease); retrying {
		select {
		case <-time.After(RetryInterval):
		case <-stopCh:
			return true
		}
	}
	return false
}

// emailTaskOutcome 描述一次邮件任务执行的业务结果，供终态写入决策。
type emailTaskOutcome int

const (
	emailOutcomeInvalid emailTaskOutcome = iota // 任务数据无法解析，直接 Failed
	emailOutcomeNoop                            // noop dummy 任务，静默删除
	emailOutcomeSent                            // 邮件发送成功
	emailOutcomeFailed                          // 发送失败，按重试策略处理
)

// executeClaimedEmail 执行一个已被当前 worker 原子领取的邮件任务：解析载荷、
// 识别 noop dummy 任务、发送邮件。终态写入由调用方以最终租约值为 CAS
// 前置条件执行（fencing），租约已丢失时跳过，避免覆盖新持有者导致重复发送。
func executeClaimedEmail(task *taskQueue.Entity) (emailTask EmailTask, outcome emailTaskOutcome, sendErr error) {
	if err := json.Unmarshal([]byte(task.TaskJson), &emailTask); err != nil {
		slog.Error("解析任务数据失败", "error", err)
		return emailTask, emailOutcomeInvalid, err
	}

	// noop 是 forgot-password 等时化 dummy 任务（#124）：静默消费并删除行，
	// 不保留 Success 状态、不发送邮件、不打"邮件发送成功"日志，避免未认证
	// 请求通过未知邮箱路径让 task_queue 无界增长。
	if emailTask.Type == "noop" {
		return emailTask, emailOutcomeNoop, nil
	}

	if err := processEmailTask(emailTask); err != nil {
		return emailTask, emailOutcomeFailed, err
	}
	return emailTask, emailOutcomeSent, nil
}

// writeEmailTaskOutcome 以最终 fencing token 为 CAS 前置条件写入任务终态
// （fencing）。返回 retrying 表示任务已标记为 Retrying，调用方应等待
// RetryInterval 后再继续，保持重试节奏。
//
// task 是批次拉取时的实体（仅用于日志等展示），running 是 ClaimTask 重读的
// 已领取实体：重试上限判断必须用 running.RetryCount（review G4）——fetch→
// claim 窗口内另一 worker 抢先重试后，旧实体的 RetryCount 已陈旧，用旧值
// 会让已达上限的任务被再重试一次。
func writeEmailTaskOutcome(task, running *taskQueue.Entity, emailTask EmailTask, outcome emailTaskOutcome, sendErr error, token string) (retrying bool) {
	switch outcome {
	case emailOutcomeNoop:
		if delErr := taskQueue.DeleteOwned(task.Id, token); delErr != nil {
			slog.Error("删除 noop 任务失败", "id", task.Id, "error", delErr)
		}
		return false

	case emailOutcomeInvalid:
		if updateErr := taskQueue.UpdateStatusOwned(task.Id, taskQueue.StatusFailed, token, sendErr); updateErr != nil {
			slog.Error("更新任务状态失败", "id", task.Id, "error", updateErr)
		}
		return false

	case emailOutcomeSent:
		if err := taskQueue.UpdateStatusOwned(task.Id, taskQueue.StatusSuccess, token, nil); err != nil {
			slog.Error("更新任务状态失败", "id", task.Id, "error", err)
			return false
		}
		slog.Info("邮件发送成功",
			"id", task.Id,
			"type", emailTask.Type,
			"to", emailTask.To,
		)
		return false

	case emailOutcomeFailed:
		slog.Error("处理邮件任务失败",
			"id", task.Id,
			"type", emailTask.Type,
			"to", emailTask.To,
			"retryCount", running.RetryCount,
			"error", sendErr,
		)

		if running.RetryCount < MaxRetries {
			if updateErr := taskQueue.IncrementRetryCountOwned(task.Id, token); updateErr != nil {
				slog.Error("更新任务重试次数失败", "id", task.Id, "error", updateErr)
			}
			if updateErr := taskQueue.UpdateStatusOwned(task.Id, taskQueue.StatusRetrying, token, sendErr); updateErr != nil {
				slog.Error("更新任务状态失败", "id", task.Id, "error", updateErr)
			}
			return true
		}

		if updateErr := taskQueue.UpdateStatusOwned(task.Id, taskQueue.StatusFailed, token, sendErr); updateErr != nil {
			slog.Error("更新任务状态失败", "id", task.Id, "error", updateErr)
		}
		return false
	}
	return false
}

// processEmailTask dispatches an email task by type.
func processEmailTask(task EmailTask) error {
	switch task.Type {
	case "activation":
		return SendActivationEmail(task.To, task.Username, task.Token, task.Locale)
	case "reset_password":
		return SendPasswordResetEmail(task.To, task.Username, task.Token, task.Locale)
	case "email_changed":
		return SendEmailChangedEmail(task.To, task.Username, task.NewEmail, task.Locale)
	case "noop":
		// 等时化 dummy 任务：静默消费，不发送任何邮件（账号枚举防护 #124）。
		return nil
	default:
		return fmt.Errorf("未知的邮件类型: %s", task.Type)
	}
}

// RecoverStaleTasks 启动时恢复邮件 worker 名下崩溃遗留的 Running 任务
// （含存量历史邮件行，与领取侧谓词一致）。
func RecoverStaleTasks() error {
	return taskQueue.RecoverStaleEmailTasks(taskQueue.LeaseDuration)
}
