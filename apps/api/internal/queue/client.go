package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/config"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/jobs"
)

const pendingExpiryDelay = 30 * time.Minute

type Client struct {
	client *asynq.Client
}

func NewClient(cfg config.Config) *Client {
	return &Client{
		client: asynq.NewClient(asynq.RedisClientOpt{
			Addr:     cfg.Redis.Address(),
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		}),
	}
}

func (c *Client) Close() error {
	if c == nil || c.client == nil {
		return nil
	}

	return c.client.Close()
}

func (c *Client) EnqueueProcessQRISCallback(ctx context.Context, payload jobs.QRISCallbackPayload) error {
	return c.enqueueJSON(ctx, jobs.TaskProcessQRISCallback, payload,
		asynq.Queue("critical"),
		asynq.MaxRetry(3),
		asynq.Timeout(30*time.Second),
		asynq.TaskID("qris:"+payload.TrxID),
	)
}

func (c *Client) EnqueueProcessDisbursementCallback(ctx context.Context, payload jobs.DisbursementCallbackPayload) error {
	return c.enqueueJSON(ctx, jobs.TaskProcessDisbursementCallback, payload,
		asynq.Queue("critical"),
		asynq.MaxRetry(3),
		asynq.Timeout(30*time.Second),
		asynq.TaskID("disbursement:"+payload.PartnerRefNo),
	)
}

func (c *Client) EnqueueExpirePendingTransaction(ctx context.Context, transactionID int64) error {
	return c.enqueueJSON(ctx, jobs.TaskExpirePendingTransaction, jobs.ExpirePendingTransactionPayload{
		TransactionID: transactionID,
	},
		asynq.Queue("default"),
		asynq.MaxRetry(1),
		asynq.ProcessIn(pendingExpiryDelay),
		asynq.Timeout(15*time.Second),
		asynq.TaskID(fmt.Sprintf("expire:%d", transactionID)),
	)
}

func (c *Client) EnqueueRelayTokoCallback(ctx context.Context, payload jobs.TokoCallbackPayload) error {
	return c.enqueueJSON(ctx, jobs.TaskRelayTokoCallback, payload,
		asynq.Queue("default"),
		asynq.MaxRetry(4),
		asynq.Timeout(15*time.Second),
		asynq.TaskID(fmt.Sprintf("%s:%s", payload.EventType, payload.Reference)),
	)
}

func (c *Client) enqueueJSON(ctx context.Context, taskType string, payload any, opts ...asynq.Option) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("queue client is not configured")
	}

	taskPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal task payload %s: %w", taskType, err)
	}

	task := asynq.NewTask(taskType, taskPayload, opts...)
	if _, err := c.client.EnqueueContext(ctx, task); err != nil {
		if errors.Is(err, asynq.ErrTaskIDConflict) {
			return nil
		}
		return fmt.Errorf("enqueue task %s: %w", taskType, err)
	}

	return nil
}
