package webhook

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type Worker struct {
	river.WorkerDefaults[WebhookArgs]
	DB         *pgxpool.Pool
	HttpClient *http.Client
	Engine     DeliveryEngine
}

func NewWorker(db *pgxpool.Pool) *Worker {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	return &Worker{
		DB:         db,
		HttpClient: httpClient,
		Engine:     NewDefaultDeliveryEngine(db, httpClient),
	}
}

func (w *Worker) Work(ctx context.Context, job *river.Job[WebhookArgs]) error {
	engine := w.engine()
	_, err := engine.DeliverEvent(ctx, DeliverEventRequest{
		EventID:  job.Args.EventID,
		LedgerID: job.Args.LedgerID,
		Attempt:  job.Attempt,
	})
	return err
}

func (w *Worker) engine() DeliveryEngine {
	if w.Engine != nil {
		return w.Engine
	}
	return NewDefaultDeliveryEngine(w.DB, w.httpClient())
}

func (w *Worker) httpClient() *http.Client {
	if w.HttpClient != nil {
		return w.HttpClient
	}
	return &http.Client{Timeout: 10 * time.Second}
}
