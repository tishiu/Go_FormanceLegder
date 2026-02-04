package main

import (
	"Go_FormanceLegder/internal/config"
	"Go_FormanceLegder/internal/db"
	"Go_FormanceLegder/internal/projector"
	"Go_FormanceLegder/internal/webhook"
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Load()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Setup River workers
	workers := river.NewWorkers()
	river.AddWorker(workers, &webhook.Worker{DB: pool})

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 100},
		},
		Workers: workers,
	})
	if err != nil {
		log.Fatalf("failed to create river client: %v", err)
	}

	// Start River
	if err := riverClient.Start(ctx); err != nil {
		log.Fatalf("failed to start river: %v", err)
	}

	// Start projector
	proj := projector.NewProjector(pool)
	go func() {
		log.Println("Projector worker starting...")
		if err := proj.Run(ctx); err != nil {
			log.Printf("projector error: %v", err)
		}
	}()

	log.Println("Worker processes started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("Shutting down workers...")
	cancel()
	riverClient.Stop(ctx)
	log.Println("Workers stopped")
}
