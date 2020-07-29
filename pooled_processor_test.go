package deepbooru

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"
)

type testProcessorResult struct {
	tags []Tag
	err  error
	i    int
}

type testProcessorCall struct {
	global, local context.Context
	timeout       time.Duration
	url           string
}

type testProcessor struct {
	results chan testProcessorResult
	calls   []testProcessorCall
	ready   bool
}

func (p *testProcessor) Process(global, local context.Context, timeout time.Duration, url string) ([]Tag, error) {
	p.calls = append(p.calls, testProcessorCall{global, local, timeout, url})
	r := <-p.results

	return r.tags, r.err
}

func (p *testProcessor) Capacity() int {
	return 1
}

func (p *testProcessor) IsReady() bool {
	return p.ready
}

func makeTestProcessorPool(count int) ([]testProcessor, []Processor) {
	xpool := make([]testProcessor, count)
	pool := make([]Processor, count)

	for i := 0; i < count; i++ {
		xpool[i] = testProcessor{
			results: make(chan testProcessorResult),
			calls:   make([]testProcessorCall, 0, 2),
			ready:   true,
		}
		pool[i] = &xpool[i]
	}

	return xpool, pool
}

func TestPooledProcessorProcess(t *testing.T) {
	testError := errors.New("test")
	results := make([]testProcessorResult, 0, 5)
	xpool, pool := makeTestProcessorPool(3)
	p := NewPooledProcessor(pool)
	done := make(chan bool)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		wg.Add(1)

		go func(i int) {
			done <- true

			url := fmt.Sprintf("http://x.com/%d.jpg", i)
			tags, err := p.Process(nil, nil, 0, url)

			mu.Lock()

			results = append(results, testProcessorResult{tags, err, i})

			mu.Unlock()
			wg.Done()
		}(i)

		<-done
	}

	processedResults := []testProcessorResult{
		testProcessorResult{i: 2, err: testError},
		testProcessorResult{i: 1, tags: []Tag{Tag{"xxx", 1}}},
		testProcessorResult{i: 0, tags: []Tag{Tag{"yyy", 1}}},
		testProcessorResult{i: 2, err: testError},
		testProcessorResult{i: 1, tags: []Tag{Tag{"zzz", 1}}},
	}

	for i := range processedResults {
		xpool[processedResults[i].i].results <- processedResults[i]
		runtime.Gosched()
	}

	wg.Wait()

	order := make([]int, len(results))

	for i := range results {
		order[i] = results[i].i
	}

	expected := []int{2, 1, 0, 3, 4}

	if !reflect.DeepEqual(order, expected) {
		t.Errorf("order: %v; expected: %v", order, expected)
	}
}

func TestPooledProcessorCapacity(t *testing.T) {
	var wg sync.WaitGroup
	xpool, pool := makeTestProcessorPool(3)
	p := NewPooledProcessor(pool)
	capacity := p.Capacity()

	if capacity != 3 {
		t.Errorf("capacity: %d; expected: 3", capacity)
		return
	}

	wg.Add(2)

	go func() {
		_, _ = p.Process(nil, nil, 0, "http://example.com")
		wg.Done()
	}()
	go func() {
		_, _ = p.Process(nil, nil, 0, "http://example.net")
		wg.Done()
	}()

	runtime.Gosched()
	capacity = p.Capacity()

	if capacity != 1 {
		t.Errorf("capacity: %d; expected: 1", capacity)
		return
	}

	for i := range xpool {
		close(xpool[i].results)
	}

	wg.Wait()
	capacity = p.Capacity()

	if capacity != 3 {
		t.Errorf("capacity: %d; expected: 3", capacity)
	}
}

func TestPooledProcessorIsReady(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		_, pool := makeTestProcessorPool(3)
		p := NewPooledProcessor(pool)

		if !p.IsReady() {
			t.Errorf("is not ready")
		}
	})

	t.Run("nil", func(t *testing.T) {
		var pp *pooledProcessor

		if pp.IsReady() {
			t.Errorf("is ready")
		}
	})

	t.Run("non initialized", func(t *testing.T) {
		var pp pooledProcessor

		if pp.IsReady() {
			t.Errorf("is ready")
		}
	})
}
