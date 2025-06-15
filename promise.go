package expofier

import "context"

type Promise struct {
	done chan struct{}
	err  error
	cb   func(error)
}

// Wait for the Promise to be Done.
func (p *Promise) Wait(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-p.done:
	case <-ctx.Done():
	}
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
	// The callback might be set after Promise is already done.
	// There is still a race-condition which might cause callback
	// to be executed twice.
	select {
	case <-p.done:
		p.cb(p.err)
	default:
	}
}

// Set the error (if not nil) and mark the Promise as Done.
func (p *Promise) Resolve(err error) {
	p.err = err
	close(p.done)
	if p.cb != nil {
		p.cb(err)
	}
}
