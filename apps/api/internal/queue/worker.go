package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/config"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/jobs"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/notifications"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/transactions"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/webhooks"
)

type QRISProcessor interface {
	ProcessQRISCallback(ctx context.Context, payload jobs.QRISCallbackPayload) error
}

type DisbursementProcessor interface {
	ProcessDisbursementCallback(ctx context.Context, payload jobs.DisbursementCallbackPayload) error
}

type PendingExpirer interface {
	ExpirePendingTransaction(ctx context.Context, transactionID int64) error
}

type TokoCallbackRelayer interface {
	RelayTokoCallback(ctx context.Context, payload jobs.TokoCallbackPayload) error
}

type Dependencies struct {
	Config config.Config
	Logger zerolog.Logger
	DB     *pgxpool.Pool
	Queue  *Client
}

type WorkerServices struct {
	QRISProcessor         QRISProcessor
	DisbursementProcessor DisbursementProcessor
	PendingExpirer        PendingExpirer
	TokoCallbackRelayer   TokoCallbackRelayer
}

type Handler struct {
	logger                zerolog.Logger
	qrisProcessor         QRISProcessor
	disbursementProcessor DisbursementProcessor
	pendingExpirer        PendingExpirer
	tokoCallbackRelayer   TokoCallbackRelayer
}

func NewServer(deps Dependencies) (*asynq.Server, *asynq.ServeMux) {
	server := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     deps.Config.Redis.Address(),
			Password: deps.Config.Redis.Password,
			DB:       deps.Config.Redis.DB,
		},
		asynq.Config{
			Concurrency: deps.Config.Queue.WorkerConcurrency,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	notificationService := notifications.NewService(deps.DB, deps.Logger)
	webhookService := webhooks.NewService(deps.DB, deps.Logger, deps.Queue).WithNotifications(notificationService)
	transactionService := transactions.NewService(deps.DB)

	mux := NewServeMux(deps.Logger, WorkerServices{
		QRISProcessor:         webhookService,
		DisbursementProcessor: webhookService,
		PendingExpirer:        transactionService,
		TokoCallbackRelayer:   webhookService,
	})

	return server, mux
}

func NewServeMux(logger zerolog.Logger, services WorkerServices) *asynq.ServeMux {
	handler := &Handler{
		logger:                logger.With().Str("component", "worker").Logger(),
		qrisProcessor:         services.QRISProcessor,
		disbursementProcessor: services.DisbursementProcessor,
		pendingExpirer:        services.PendingExpirer,
		tokoCallbackRelayer:   services.TokoCallbackRelayer,
	}

	mux := asynq.NewServeMux()
	mux.HandleFunc(jobs.TaskProcessQRISCallback, handler.HandleProcessQRISCallback)
	mux.HandleFunc(jobs.TaskProcessDisbursementCallback, handler.HandleProcessDisbursementCallback)
	mux.HandleFunc(jobs.TaskExpirePendingTransaction, handler.HandleExpirePendingTransaction)
	mux.HandleFunc(jobs.TaskRelayTokoCallback, handler.HandleRelayTokoCallback)
	return mux
}

func (h *Handler) HandleProcessQRISCallback(ctx context.Context, task *asynq.Task) error {
	var payload jobs.QRISCallbackPayload
	if err := decodeTaskPayload(task, &payload); err != nil {
		return err
	}

	if h.qrisProcessor == nil {
		return fmt.Errorf("qris processor is not configured")
	}

	return h.qrisProcessor.ProcessQRISCallback(ctx, payload)
}

func (h *Handler) HandleProcessDisbursementCallback(ctx context.Context, task *asynq.Task) error {
	var payload jobs.DisbursementCallbackPayload
	if err := decodeTaskPayload(task, &payload); err != nil {
		return err
	}

	if h.disbursementProcessor == nil {
		return fmt.Errorf("disbursement processor is not configured")
	}

	return h.disbursementProcessor.ProcessDisbursementCallback(ctx, payload)
}

func (h *Handler) HandleExpirePendingTransaction(ctx context.Context, task *asynq.Task) error {
	var payload jobs.ExpirePendingTransactionPayload
	if err := decodeTaskPayload(task, &payload); err != nil {
		return err
	}

	if h.pendingExpirer == nil {
		return fmt.Errorf("pending expirer is not configured")
	}

	return h.pendingExpirer.ExpirePendingTransaction(ctx, payload.TransactionID)
}

func (h *Handler) HandleRelayTokoCallback(ctx context.Context, task *asynq.Task) error {
	var payload jobs.TokoCallbackPayload
	if err := decodeTaskPayload(task, &payload); err != nil {
		return err
	}

	if h.tokoCallbackRelayer == nil {
		return fmt.Errorf("toko callback relayer is not configured")
	}

	return h.tokoCallbackRelayer.RelayTokoCallback(ctx, payload)
}

func decodeTaskPayload(task *asynq.Task, target any) error {
	if err := json.Unmarshal(task.Payload(), target); err != nil {
		return fmt.Errorf("decode task payload %s: %w", task.Type(), err)
	}

	return nil
}
