package expofier

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type sendJob struct {
	promise *Promise
	msg     Message
}

type receiptJob struct {
	promise *Promise
	ticket  Ticket

	// The last time when we tried to resolve the ticket.
	//
	// If nil, we haven't tried yet.
	tried *time.Time
}

// The background service taking care of delivering notifications, batching requests,
// retries, resolving promises, etc.
type Service struct {
	// The underlying client to use for sending requests.
	Client *Client

	// The function returning the current time.
	//
	// Default: [time.Now]
	Now func() time.Time

	// For how long to aggregate messages into chunks.
	//
	// In other words, it's the time since calling Send for the first message
	// in a chunk to sending the whole chunk to the server.
	//
	// Default: 1 second.
	SendChunk time.Duration

	// For how long to aggregate tickets into chunks.
	//
	// Default: 1 second.
	ResolveChunk time.Duration

	// How often to re-check the status of a ticket.
	//
	// Default: 1 second.
	ResolveInterval time.Duration

	sendJobs    chan sendJob
	receiptJobs chan receiptJob
	constructed bool
}

func NewService() *Service {
	return &Service{
		Client:          NewClient(),
		sendJobs:        make(chan sendJob),
		receiptJobs:     make(chan receiptJob),
		SendChunk:       time.Second,
		ResolveChunk:    time.Second,
		ResolveInterval: time.Second,
		Now:             time.Now,
		constructed:     true,
	}
}

// Send the message in background, wait for its status to update, and resolve the Promise.
func (s *Service) Send(ctx context.Context, msg Message) *Promise {
	job := sendJob{
		promise: &Promise{
			done: make(chan struct{}),
		},
		msg: msg,
	}
	select {
	case s.sendJobs <- job:
		return job.promise
	case <-ctx.Done():
		job.promise.Resolve(ctx.Err())
		return job.promise
	}
}

// Start the background service.
//
// Exits when the context is cancelled.
func (s *Service) Run(ctx context.Context) {
	if !s.constructed {
		panic("Service must be constructed using NewService")
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go s.runSender(ctx, wg)
	go s.runResolver(ctx)
	wg.Wait()
}

func (d *Service) runSender(ctx context.Context, wg *sync.WaitGroup) {
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

func (s *Service) sendChunk(ctx context.Context, chunk []sendJob) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	msgs := make([]Message, 0, len(chunk))
	for _, job := range chunk {
		msgs = append(msgs, job.msg)
	}
	resps, err := s.Client.SendMessages(ctx, msgs)
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
		case s.receiptJobs <- rJob:
		case <-ctx.Done():
			err := fmt.Errorf("send receipt job: %w", ctx.Err())
			job.promise.Resolve(err)
		}
	}
}

func (s *Service) runResolver(ctx context.Context) {
	ticker := time.NewTicker(s.ResolveChunk)
	chunk := make([]receiptJob, 0, 300)
	for {
		select {
		case job := <-s.receiptJobs:
			if len(chunk) == 0 {
				ticker.Reset(s.ResolveChunk)
			}
			chunk = append(chunk, job)
			if len(chunk) >= 300 {
				s.resolveChunk(ctx, chunk)
				chunk = chunk[:0]
			}
		case <-ticker.C:
			if len(chunk) > 0 {
				s.resolveChunk(ctx, chunk)
				chunk = chunk[:0]
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) resolveChunk(ctx context.Context, chunk []receiptJob) {
	tickets := make([]Ticket, 0, len(chunk))
	for _, job := range chunk {
		if job.tried != nil {
			now := s.Now()
			nextTry := job.tried.Add(s.ResolveInterval)
			wait := nextTry.Sub(now)
			if wait > 0 {
				time.Sleep(wait)
			}
		}
		tickets = append(tickets, job.ticket)
	}
	resps, err := s.Client.FetchReceipts(ctx, tickets)
	if err != nil {
		for _, job := range chunk {
			job.promise.Resolve(err)
		}
		return
	}
	unresolved := make([]receiptJob, 0, len(chunk)-len(resps))
	for _, job := range chunk {
		receipt, resolved := resps[job.ticket]
		if resolved {
			job.promise.Resolve(receipt.Error)
			continue
		}
		now := s.Now()
		job.tried = &now
		unresolved = append(unresolved, job)
	}
	go s.rescheduleTickets(ctx, unresolved)
}

// Put back into queue unresolved receipt jobs.
func (s *Service) rescheduleTickets(ctx context.Context, chunk []receiptJob) {
	for _, job := range chunk {
		select {
		case s.receiptJobs <- job:
		case <-ctx.Done():
			return
		}
	}
}
