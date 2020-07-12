package deepbooru

import (
	"context"
	"time"
)

type pooledProcessor struct {
	pool chan Processor
	free int
}

func NewPooledProcessor(processors []Processor) Processor {
	free := len(processors)
	pool := make(chan Processor, free)

	for _, p := range processors {
		pool <- p
	}

	return &pooledProcessor{
		free: free,
		pool: pool,
	}
}

func (pp *pooledProcessor) getFromPool() Processor {
	p := <-pp.pool

	pp.free--

	return p
}

func (pp *pooledProcessor) returnToPool(p Processor) {
	pp.free++
	pp.pool <- p
}

func (pp *pooledProcessor) Process(global, local context.Context, timeout time.Duration, url string) ([]Tag, error) {
	p := pp.getFromPool()
	defer pp.returnToPool(p)

	return p.Process(global, local, timeout, url)
}

func (pp *pooledProcessor) Capacity() int {
	return pp.free
}

func (pp *pooledProcessor) IsReady() bool {
	for p := range pp.pool {
		if !p.IsReady() {
			return false
		}
	}

	return len(pp.pool) > 0
}
