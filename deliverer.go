package expofier

import (
	"context"
	"time"
)

type Promise struct {
	Done  bool
	Error error
}

func (p *Promise) Resolve(err error) {
	p.Error = err
	p.Done = true
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
	d.runSender(ctx)
}

func (d *Deliverer) runSender(ctx context.Context) {
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
