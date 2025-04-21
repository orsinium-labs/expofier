package expofier

import (
	"context"
	"sync"
	"time"
)

type Promise struct {
	done chan struct{}
	err  error
	cb   func(error)
}

// Channel that will be closed when the Promise is done.
func (p *Promise) Done() <-chan struct{} {
	return p.done
}

func (p *Promise) Err() error {
	select {
	case <-p.done:
	default:
		panic("trying to get Err before Promise is Done")
	}
	return p.err
}

// Register function to be called when Promise is Done.
func (p *Promise) Callback(cb func(error)) {
	p.cb = cb
}

// Set the error (if not nil) and mark the Promise as Done.
func (p *Promise) Resolve(err error) {
	p.err = err
	close(p.done)
	if p.cb != nil {
		p.cb(err)
	}
}

type sendJob struct {
	promise *Promise
	msg     Message
}

type receiptJob struct {
	promise *Promise
	ticket  Ticket
	tried   *time.Time
}

type Deliverer struct {
	client       Client
	sendJobs     chan sendJob
	receiptJobs  chan receiptJob
	SendChunk    time.Duration
	ResolveChunk time.Duration
	Now          func() time.Time
}

func (d *Deliverer) Send(ctx context.Context, msg Message) *Promise {
	job := sendJob{
		promise: &Promise{},
		msg:     msg,
	}
	select {
	case d.sendJobs <- job:
		return job.promise
	case <-ctx.Done():
		return nil
	}
}

func (d *Deliverer) Run(ctx context.Context) {
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go d.runSender(ctx, wg)
	go d.runResolver(ctx, wg)
	wg.Wait()
}

func (d *Deliverer) runSender(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(d.SendChunk)
	chunk := make([]sendJob, 0, 100)
	for {
		select {
		case job := <-d.sendJobs:
			if len(chunk) == 0 {
				ticker.Reset(d.SendChunk)
			}
			chunk = append(chunk, job)
			if len(chunk) >= 100 {
				d.sendChunk(ctx, chunk)
				chunk = chunk[:0]
			}
		case <-ticker.C:
			if len(chunk) > 0 {
				d.sendChunk(ctx, chunk)
				chunk = chunk[:0]
			}
		case <-ctx.Done():
			return
		}
	}
}

func (d *Deliverer) sendChunk(ctx context.Context, chunk []sendJob) {
	msgs := make([]Message, 0, len(chunk))
	for _, job := range chunk {
		msgs = append(msgs, job.msg)
	}
	resps, err := d.client.SendMessages(ctx, msgs)
	if err != nil {
		for _, job := range chunk {
			job.promise.Resolve(err)
		}
		return
	}
	for i, resp := range resps {
		job := chunk[i]
		if resp.Error != nil {
			job.promise.Resolve(resp.Error)
			continue
		}
		rJob := receiptJob{
			promise: job.promise,
			ticket:  resp.Ticket,
		}
		select {
		case d.receiptJobs <- rJob:
		case <-ctx.Done():
			return
		}
	}
}

func (d *Deliverer) runResolver(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(d.ResolveChunk)
	chunk := make([]receiptJob, 0, 300)
	for {
		select {
		case job := <-d.receiptJobs:
			if len(chunk) == 0 {
				ticker.Reset(d.ResolveChunk)
			}
			chunk = append(chunk, job)
			if len(chunk) >= 300 {
				d.resolveChunk(ctx, chunk)
				chunk = chunk[:0]
			}
		case <-ticker.C:
			if len(chunk) > 0 {
				d.resolveChunk(ctx, chunk)
				chunk = chunk[:0]
			}
		case <-ctx.Done():
			return
		}
	}
}

func (d *Deliverer) resolveChunk(ctx context.Context, chunk []receiptJob) {
	tickets := make([]Ticket, 0, len(chunk))
	for _, job := range chunk {
		tickets = append(tickets, job.ticket)
	}
	resps, err := d.client.FetchReceipts(ctx, tickets)
	if err != nil {
		for _, job := range chunk {
			job.promise.Resolve(err)
		}
		return
	}
	for _, job := range chunk {
		receipt, resolved := resps[job.ticket]
		if resolved {
			job.promise.Resolve(receipt.Error)
			continue
		}
		now := d.Now()
		job.tried = &now
		select {
		case d.receiptJobs <- job:
		case <-ctx.Done():
			return
		}
	}
}
