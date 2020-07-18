package deepbooru_ipc

import (
	"context"
	"errors"
	"log"
	"time"

	"deepbooru"
)

type Processor struct {
	Bus  Bus
	busy bool
}

func (p *Processor) setBusy(b bool) {
	p.busy = b
}

func (p *Processor) Process(global, local context.Context, timeout time.Duration, url string) ([]deepbooru.Tag, error) {
	if !p.Bus.IsReady() {
		return nil, deepbooru.ErrTerminated
	}

	p.setBusy(true)
	defer p.setBusy(false)

	var ctx context.Context
	var cancel context.CancelFunc

	if timeout > 0 {
		ctx, cancel = context.WithTimeout(local, timeout)

		defer cancel()
	} else {
		ctx = context.TODO()
	}

	log.Printf("Sending %s", url)
	p.Bus.In() <- Message{URL: url}

	for {
		select {
		case <-global.Done():
			p.Bus.Interrupt()

			return nil, deepbooru.ErrTerminated
		case <-local.Done():
			p.Bus.Interrupt()

			return nil, deepbooru.ErrCancelled
		case <-ctx.Done():
			p.Bus.Interrupt()

			return nil, deepbooru.ErrTimeout
		case m := <-p.Bus.Out():
			log.Printf("Recieved result")

			if m.Error != "" {
				return nil, errors.New(m.Error)
			} else {
				return m.Tags, nil
			}
		}
	}
}

func (p *Processor) Capacity() int {
	if p.Bus.IsReady() && !p.busy {
		return 1
	}

	return 0
}

func (p *Processor) IsReady() bool {
	return p.Bus.IsReady()
}
