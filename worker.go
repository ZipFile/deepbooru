package deepbooru

import (
	"context"
	"log"
	"sync"
	"time"
)

type JobContext struct {
	ID int64

	Context context.Context
	Cancel  context.CancelFunc
}

type Worker struct {
	sync.Mutex

	Name string

	BusFactory BusFactory
	Processor  Processor

	TickInterval   time.Duration
	BeatInterval   time.Duration
	ProcessTimeout time.Duration

	jobs map[int64]JobContext
	ctx  context.Context
}

type WorkerBus struct {
	Worker *Worker
}

func (*WorkerBus) Beat(int64) error {
	return nil
}

func (b *WorkerBus) Cancel(id int64) error {
	return b.Worker.OnCancel(id)
}

func (*WorkerBus) Done(id int64, tags []Tag) error {
	return nil
}

func (*WorkerBus) Error(int64, ErrorCode, string) error {
	return nil
}

func (*WorkerBus) Deschedule([]int64) error {
	return nil
}

func (b *WorkerBus) Schedule(node string, tasks []Info) error {
	return b.Worker.OnSchedule(node, tasks)
}

func (b *WorkerBus) WakeUp() error {
	return b.Worker.OnWakeUp()
}

func (*WorkerBus) WorkerStatus(string, int) error {
	return nil
}

func NewWorker(bf BusFactory) *Worker {
	return &Worker{
		BusFactory: bf,

		TickInterval:   10 * time.Second,
		BeatInterval:   15 * time.Second,
		ProcessTimeout: 60 * time.Second,

		jobs: make(map[int64]JobContext),
	}
}

func (w *Worker) Run(ctx context.Context) error {
	if w.ctx != nil {
		return ErrAlreadyRunning
	}

	var err error
	wb := &WorkerBus{w}
	unsubscribeJobs, err := w.BusFactory.SubscribeAll(
		wb,
		true,
		"schedule",
		"wakeup",
	)

	if err != nil {
		return err
	}

	defer unsubscribeJobs()

	unsubscribeCancel, err := w.BusFactory.SubscribeAll(wb, false, "cancel")

	if err != nil {
		return err
	}

	defer unsubscribeCancel()

	w.ctx = ctx
	ticker := time.NewTicker(w.TickInterval)

	for {
		select {
		case <-ticker.C:
			w.tick()
		case <-ctx.Done():
			ticker.Stop()
			w.ctx = nil

			log.Printf("shutting down")

			w.waitForJobsToFinish()

			return nil
		}
	}
}

func (w *Worker) Beat(id int64) error {
	return w.BusFactory.Publish().Beat(id)
}

func (w *Worker) Done(id int64, tags []Tag) error {
	return w.BusFactory.Publish().Done(id, tags)
}

func (w *Worker) Error(id int64, code ErrorCode, reason string) error {
	return w.BusFactory.Publish().Error(id, code, reason)
}

func (w *Worker) OnSchedule(node string, tasks []Info) error {
	if node != w.Name {
		return nil
	}

	for i := range tasks {
		w.process(&tasks[i])
	}

	return nil
}

func (w *Worker) OnCancel(id int64) error {
	w.Lock()

	job, ok := w.jobs[id]

	if !ok {
		return nil
	}

	delete(w.jobs, id)
	w.Unlock()
	job.Cancel()

	return nil
}

func (w *Worker) OnWakeUp() error {
	return w.publishStatus()
}

func (w *Worker) tick() {
	w.gc()

	err := w.publishStatus()

	log.Printf("failed to send worker status: %s", err)
}

func (w *Worker) publishStatus() error {
	return w.BusFactory.Publish().WorkerStatus(w.Name, w.Processor.Capacity())
}

func (w *Worker) process(info *Info) {
	id := info.ID

	w.Lock()
	job, ok := w.jobs[id]

	if ok {
		w.Unlock()
		log.Printf("ignoring %d, already scheduled", id)
		return
	}

	job.ID = id
	job.Context, job.Cancel = context.WithCancel(context.Background())
	w.jobs[id] = job

	w.Unlock()

	go w.startHeartbeat(id, job.Context)

	tags, err := w.Processor.Process(w.ctx, job.Context, w.ProcessTimeout, info.URL)

	switch err {
	case nil:
		w.Done(id, tags)
	case ErrCancelled:
	case ErrTimeout:
		w.Error(id, Timeout, "timeout")
	case ErrInvalid:
		w.Error(id, Invalid, "invalid")
	case ErrTerminated:
		w.Error(id, Terminated, "terminated")
	default:
		w.Error(id, InternalError, err.Error())
	}

	job.Cancel()
	w.Lock()
	delete(w.jobs, id)
	w.Unlock()
}

func (w *Worker) startHeartbeat(id int64, ctx context.Context) {
	var err error
	heart := time.NewTicker(w.BeatInterval)

	for {
		select {
		case <-ctx.Done():
			heart.Stop()

			return
		case <-heart.C:
			err = w.Beat(id)

			if err != nil {
				log.Printf("failed to send heart beat for %d", id)
			}
		}
	}
}

func (w *Worker) waitForJobsToFinish() {
	for {
		w.Lock()

		if len(w.jobs) == 0 {
			w.Unlock()
			return
		}

		tmp := make([]JobContext, 0, len(w.jobs))

		for _, job := range w.jobs {
			tmp = append(tmp, job)
		}

		w.Unlock()

		for _, job := range tmp {
			<-job.Context.Done()

			w.Lock()
			delete(w.jobs, job.ID)
			w.Unlock()
		}
	}
}

func (w *Worker) gc() {
	w.Lock()

	for id, job := range w.jobs {
		err := job.Context.Err()

		if err != nil {
			log.Printf("gc-ing %d job", id)
			delete(w.jobs, id)
		}
	}

	w.Unlock()
}
