package worker

import (
	"clustercron/internal/broker"
	"clustercron/internal/storage"
	"context"
	"log"
)

type Worker struct {
	db       *storage.DB
	broker   *broker.Redis
	executor *Executor
	queue    string
	nodeID   string
}

func NewWorker(db *storage.DB, b *broker.Redis, executor *Executor, nodeID string) *Worker {
	return &Worker{
		db:       db,
		broker:   b,
		executor: executor,
		queue:    broker.WorkerQueue(nodeID),
		nodeID:   nodeID,
	}
}

func (w *Worker) Run(ctx context.Context) {
	log.Printf("[worker %s] starting, listening on queue %s", w.nodeID, w.queue)

	for {
		// Check for shutdown before blocking on the queue.
		if ctx.Err() != nil {
			log.Printf("[worker %s] stopped", w.nodeID)
			return
		}

		w.processNext(ctx)
	}
}

func (w *Worker) processNext(ctx context.Context) {
	data, err := w.broker.BlockPop(ctx, w.queue)
	if err != nil {
		log.Printf("[worker %s] ERROR pop: %v", w.nodeID, err)
		return
	}
	if data == nil {
		// Context was cancelled — normal shutdown.
		return
	}
	msg, err := broker.DecodeJobMessage(data)
	if err != nil {
		log.Printf("[worker %s] ERROR decode message: %v (raw: %s)", w.nodeID, err, string(data))
		return
	}
	log.Printf("[worker %s] received job %s (run %s)", w.nodeID, msg.JobName, msg.RunID)

	job, err := w.db.GetJob(ctx, msg.JobID)
	if err != nil {
		log.Printf("[worker %s] ERROR get job %s: %v", w.nodeID, msg.JobID, err)
		// Job may have been deleted between enqueue and now — not a fatal error.
		return
	}

	scheduledAt := msg.ScheduledAt
	job.NextFireAt = &scheduledAt

	// 5. Execute.
	w.executor.Execute(ctx, job, msg.RunID)

}
