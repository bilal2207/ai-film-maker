package media

import (
	"context"
	"errors"
	"log"
	"sync"
)

type Job struct {
	MediaID   string
	ProjectID string
}

type JobHandler func(ctx context.Context, job Job) error

type Queue interface {
	Enqueue(job Job) error
	Start(workers int, handler JobHandler)
	Stop()
}

// MemoryQueue provides a bounded worker pool for background media processing.
// It defines a clean interface boundary designed to be swapped with Temporal workflows in future hops.
type MemoryQueue struct {
	jobChan chan Job
	handler JobHandler
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	once    sync.Once
}

func NewMemoryQueue(bufferSize int) *MemoryQueue {
	if bufferSize <= 0 {
		bufferSize = 100
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &MemoryQueue{
		jobChan: make(chan Job, bufferSize),
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (q *MemoryQueue) Enqueue(job Job) error {
	select {
	case <-q.ctx.Done():
		return errors.New("queue is shutting down")
	case q.jobChan <- job:
		return nil
	default:
		return errors.New("processing queue buffer is full")
	}
}

func (q *MemoryQueue) Start(workers int, handler JobHandler) {
	if workers <= 0 {
		workers = 2
	}
	q.handler = handler

	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}
}

func (q *MemoryQueue) worker(id int) {
	defer q.wg.Done()
	log.Printf("[MediaWorker %d] Started processing worker.", id)

	for {
		select {
		case <-q.ctx.Done():
			log.Printf("[MediaWorker %d] Stopping worker.", id)
			return
		case job, ok := <-q.jobChan:
			if !ok {
				return
			}
			log.Printf("[MediaWorker %d] Processing media %s (project: %s)...", id, job.MediaID, job.ProjectID)
			if err := q.handler(q.ctx, job); err != nil {
				log.Printf("[MediaWorker %d] Job failed for media %s: %v", id, job.MediaID, err)
			} else {
				log.Printf("[MediaWorker %d] Job completed for media %s.", id, job.MediaID)
			}
		}
	}
}

func (q *MemoryQueue) Stop() {
	q.once.Do(func() {
		q.cancel()
		close(q.jobChan)
		q.wg.Wait()
		log.Println("Media processing queue stopped gracefully.")
	})
}
