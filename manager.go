package deepbooru

import (
	"context"
	"log"
	"time"
)

type Manager struct {
	Authorizer Authorizer
	BusFactory BusFactory
	Storage    Storage

	TickInterval    time.Duration
	StalledInterval time.Duration

	ctx context.Context
}

type ManagerBus struct {
	Manager *Manager
}

func (b *ManagerBus) Beat(id int64) error {
	return b.Manager.OnBeat(id)
}

func (b *ManagerBus) Cancel(id int64) error {
	return b.Manager.OnCancel(id)
}

func (b *ManagerBus) Done(id int64, tags []Tag) error {
	return b.Manager.OnDone(id, tags)
}

func (b *ManagerBus) Error(id int64, code ErrorCode, reason string) error {
	return b.Manager.OnError(id, code, reason)
}

func (b *ManagerBus) Deschedule(ids []int64) error {
	return b.Manager.OnDeschedule(ids)
}

func (*ManagerBus) Schedule(node string, tasks []Info) error {
	return nil
}

func (*ManagerBus) WakeUp() error {
	return nil
}

func (b *ManagerBus) WorkerStatus(node string, capacity int) error {
	return b.Manager.OnWorkerStatus(node, capacity)
}

func NewManager(a Authorizer, bf BusFactory, s Storage) *Manager {
	return &Manager{
		Authorizer: a,
		BusFactory: bf,
		Storage:    s,

		TickInterval:    3 * time.Second,
		StalledInterval: 30 * time.Second,
	}
}

func (m *Manager) Run(ctx context.Context) error {
	if m.ctx != nil {
		return ErrAlreadyRunning
	}

	unsubscribe, err := m.BusFactory.SubscribeAll(
		&ManagerBus{m},
		true,
		"beat",
		"cancel",
		"done",
		"error",
		"deschedule",
		"worker_status",
	)

	if err != nil {
		return err
	}

	m.ctx = ctx
	ticker := time.NewTicker(m.TickInterval)

	defer unsubscribe()

	for {
		select {
		case <-ticker.C:
			m.tick()
		case <-ctx.Done():
			ticker.Stop()
			m.ctx = nil

			log.Printf("shutting down")

			return nil
		}
	}
}

func (m *Manager) tick() {
	aborted, err := m.Storage.AbortStalled(m.StalledInterval)

	if err != nil {
		log.Printf("failed to abort stalled jobs: %s", err)

		return
	}

	for i := range aborted {
		id := aborted[i].ID
		err = m.BusFactory.Publish().Cancel(id)

		if err != nil {
			log.Printf("failed to send cancel event (id=%d): %s", id, err)

			return
		}

		err = m.BusFactory.Publish().Error(id, Timeout, "timeout")

		if err != nil {
			log.Printf("failed to send error event (id=%d): %s", id, err)

			return
		}
	}

	return
}

func (m *Manager) OnBeat(id int64) error {
	return m.Storage.Beat(id)
}

func (m *Manager) OnCancel(id int64) error {
	return m.Storage.Error(id, Canceled, "")
}

func (m *Manager) OnDone(id int64, tags []Tag) error {
	return m.Storage.Done(id, tags)
}

func (m *Manager) OnError(id int64, code ErrorCode, reason string) error {
	return m.Storage.Error(id, code, reason)
}

func (m *Manager) OnDeschedule(ids []int64) error {
	err := m.Storage.Reset(ids)

	if err != nil {
		return err
	}

	return m.BusFactory.Publish().WakeUp()
}

func (m *Manager) OnWorkerStatus(node string, capacity int) error {
	if capacity <= 0 {
		return nil
	}

	toSchedule := capacity + capacity/3
	todo, err := m.Storage.Pop(toSchedule)

	if err != nil {
		return err
	}

	if len(todo) == 0 {
		return nil
	}

	return m.BusFactory.Publish().Schedule(node, todo)
}
