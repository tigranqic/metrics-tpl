// Package audit provides asynchronous audit event publishing to multiple observers.
// It supports a worker pool and buffered channel to handle high-throughput event logging.
package audit

import (
	"context"

	"go.uber.org/zap"
)

// Publisher manages a list of observers and dispatches audit events asynchronously.
type Publisher struct {
	log       *zap.Logger        // Logger for errors and warnings
	observers []Observer         // List of registered audit observers
	JobCh     chan Event         // Buffered channel of audit events
	cancel    context.CancelFunc // Cancels all worker goroutines
}

// NewPublisherWithPool creates a new Publisher with a fixed-size worker pool.
// Parameters:
//   - log: zap.Logger instance for logging errors and warnings
//   - observers: slice of Observer to notify on each event
//   - poolSize: number of worker goroutines to process events concurrently
//
// Returns a configured Publisher ready to accept events.
func NewPublisherWithPool(log *zap.Logger, observers []Observer, poolSize int) *Publisher {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Publisher{
		log:       log,
		observers: observers,
		JobCh:     make(chan Event, 1000),
		cancel:    cancel,
	}

	for i := 0; i < poolSize; i++ {
		go p.worker(ctx)
	}

	return p
}

// NotifyAllAsync enqueues an audit event to be sent to all observers asynchronously.
// If the internal queue is full, the event will be dropped and a warning is logged.
func (p *Publisher) NotifyAllAsync(event Event) {
	select {
	case p.JobCh <- event:
	default:
		p.log.Warn("audit queue full, dropping event")
	}
}

// Shutdown cancels all worker goroutines and closes the event channel.
// After calling Shutdown, the Publisher cannot be used again.
func (p *Publisher) Shutdown() {
	p.cancel()
	close(p.JobCh)
}

// worker processes audit events from the JobCh channel and notifies all observers.
// This method runs in a goroutine and exits when the context is cancelled.
func (p *Publisher) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case e := <-p.JobCh:
			for _, o := range p.observers {
				if err := o.Notify(e); err != nil {
					p.log.Error("observer notify failed", zap.Error(err))
				}
			}
		}
	}
}
