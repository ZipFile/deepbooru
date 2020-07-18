package deepbooru_ipc

import (
	"context"
	"testing"
	//	"time"

	"deepbooru"
)

type testBus struct {
	interrupts int
	in, out    chan Message
}

func (b *testBus) Interrupt() {
	b.interrupts++
}

func (b *testBus) IsReady() bool {
	return b.in != nil && b.out != nil
}

func (b *testBus) In() chan<- Message {
	return b.in
}

func (b *testBus) Out() <-chan Message {
	return b.out
}

func TestProcessorProcessNotRunning(t *testing.T) {
	b := testBus{}
	p := Processor{Bus: &b}

	_, err := p.Process(nil, nil, 0, "")

	if err != deepbooru.ErrTerminated {
		t.Errorf("err: %s; expected: deepbooru.ErrTerminated", err)
	}
}

func TestProcessorProcessTerminated(t *testing.T) {
	global, globalCancel := context.WithCancel(context.Background())
	local, localCancel := context.WithCancel(context.Background())

	b := testBus{
		in:  make(chan Message, 1),
		out: make(chan Message),
	}
	p := Processor{Bus: &b}

	globalCancel()

	_, err := p.Process(global, local, 0, "http://exampole.com/test.jpg")

	localCancel()

	if err != deepbooru.ErrTerminated {
		t.Errorf("err: %s; expected: deepbooru.ErrTerminated", err)
	}

	if b.interrupts != 1 {
		t.Errorf("Not interrupted")
	}
}

func TestProcessorProcessCancelled(t *testing.T) {
	global, globalCancel := context.WithCancel(context.Background())
	local, localCancel := context.WithCancel(context.Background())

	b := testBus{
		in:  make(chan Message, 1),
		out: make(chan Message),
	}
	p := Processor{Bus: &b}

	localCancel()

	_, err := p.Process(global, local, 0, "http://exampole.com/test.jpg")

	globalCancel()

	if err != deepbooru.ErrCancelled {
		t.Errorf("err: %s; expected: deepbooru.ErrCancelled", err)
	}

	if b.interrupts != 1 {
		t.Errorf("Not interrupted")
	}
}

func TestProcessorProcessTimeout(t *testing.T) {
	global, globalCancel := context.WithCancel(context.Background())
	local, localCancel := context.WithCancel(context.Background())

	b := testBus{
		in:  make(chan Message, 1),
		out: make(chan Message),
	}
	p := Processor{Bus: &b}

	localCancel()

	_, err := p.Process(global, local, 0, "http://exampole.com/test.jpg")

	globalCancel()

	if err != deepbooru.ErrCancelled {
		t.Errorf("err: %s; expected: deepbooru.ErrCancelled", err)
	}

	if b.interrupts != 1 {
		t.Errorf("Not interrupted")
	}
}

func TestProcessorCapacity(t *testing.T) {
	p := Processor{}

	t.Run("busy", func(t *testing.T) {
		p.setBusy(true)

		if p.Capacity() != 0 {
			t.Error("Non zero capacity")
		}
	})

	t.Run("free", func(t *testing.T) {
		p.setBusy(false)

		if p.Capacity() == 0 {
			t.Error("Zero capacity")
		}
	})
}

func TestProcessorIsReady(t *testing.T) {
	b := testBus{}
	p := Processor{Bus: &b}

	t.Run("not ready", func(t *testing.T) {
		if p.IsReady() {
			t.Error("Is ready")
		}
	})

	t.Run("ready", func(t *testing.T) {
		b.in = make(chan Message)
		b.out = b.in

		if !p.IsReady() {
			t.Error("Not ready")
		}
	})
}
