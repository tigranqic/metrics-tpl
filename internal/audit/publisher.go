package audit

import (
	"context"

	"go.uber.org/zap"
)

type Publisher struct {
	log       *zap.Logger
	observers []Observer
	JobCh     chan Event
	cancel    context.CancelFunc
}

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

func (p *Publisher) NotifyAllAsync(event Event) {
	select {
	case p.JobCh <- event:
	default:
		p.log.Warn("audit queue full, dropping event")
	}
}

func (p *Publisher) Shutdown() {
	p.cancel()
	close(p.JobCh)
}

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
