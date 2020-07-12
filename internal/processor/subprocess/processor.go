package subprocess_processor

import (
	"context"
	"errors"
	"log"
	"time"

	"deepbooru"
)

type ExecProcessor struct {
	Path    string
	Args    []string
	Environ []string

	KillTimeout time.Duration

	busy bool
	n    *Nurse
}

func New(path string, args []string, environ []string, killTimeout time.Duration) *ExecProcessor {
	return &ExecProcessor{
		Path:    path,
		Args:    args,
		Environ: environ,

		KillTimeout: killTimeout,
	}
}

func (p *ExecProcessor) Run(ctx context.Context) error {
	if p.n != nil {
		return errors.New("Already running")
	}

	n, err := NewNurse(p.Path, p.Args, p.Environ, ctx, p.KillTimeout)

	if err != nil {
		return err
	}

	p.n = n
	done := make(chan struct{})

	go func() {
		p.n.Start()
		close(done)

		p.n = nil
	}()

	<-done

	return nil
}

func (p *ExecProcessor) setBusy(b bool) {
	p.busy = b
}

func (p *ExecProcessor) Process(global, local context.Context, timeout time.Duration, url string) ([]deepbooru.Tag, error) {
	if p.n == nil {
		return nil, deepbooru.ErrTerminated
	}

	p.setBusy(true)
	defer p.setBusy(false)

	ctx, cancel := context.WithTimeout(local, timeout)

	defer cancel()

	log.Printf("Sending %s", url)
	p.n.In() <- Message{URL: url}

	for {
		select {
		case <-global.Done():
			return nil, deepbooru.ErrTerminated
		case <-local.Done():
			return nil, deepbooru.ErrCancelled
		case <-ctx.Done():
			return nil, deepbooru.ErrTimeout
		case m := <-p.n.Out():
			log.Printf("Recieved result")

			if m.Error != "" {
				return nil, errors.New(m.Error)
			} else {
				return m.Tags, nil
			}
		}
	}
}

func (p *ExecProcessor) Capacity() int {
	if p.busy {
		return 0
	}

	return 1
}

func (p *ExecProcessor) IsReady() bool {
	return p.n.IsReady()
}
