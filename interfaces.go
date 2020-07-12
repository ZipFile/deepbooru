package deepbooru

import (
	"context"
	"errors"
	"time"
)

var ErrAlreadyRunning = errors.New("already running")
var ErrCancelled = errors.New("cancelled")
var ErrInvalid = errors.New("invalid")
var ErrTerminated = errors.New("terminated")
var ErrTimeout = errors.New("timeout")

type Status int8
type ErrorCode int8

const (
	Pending Status = iota
	Processing
	Done
	Failed
	OK ErrorCode = iota
	Canceled
	NotFound
	Invalid
	Terminated
	Timeout
	InternalError
)

type Tag struct {
	Name  string
	Score float32
}

type Info struct {
	ID       int64
	URL      string
	Status   Status
	Tags     []Tag
	Priority int

	LastActivity time.Time
	ErrorReason  string
	ErrorCode    ErrorCode
}

type Storage interface {
	AbortStalled(timeout time.Duration) ([]Info, error)
	ListActive() ([]Info, error)
	QueueSize() (int, error)
	Position(id int64) (int, error)

	Push(url string, priority int) (*Info, error)
	Pop(n int) ([]Info, error)
	Reset(ids []int64) error
	Get(id int64) (*Info, error)

	Beat(id int64) error
	Done(id int64, tags []Tag) error
	Error(id int64, code ErrorCode, reason string) error
}

type Bus interface {
	Beat(id int64) error
	Cancel(id int64) error
	Done(id int64, tags []Tag) error
	Error(id int64, code ErrorCode, reason string) error

	Deschedule(ids []int64) error
	Schedule(node string, tasks []Info) error
	WakeUp() error
	WorkerStatus(node string, capacity int) error
}

type Terminator func()

type BusFactory interface {
	Publish() Bus
	SubscribeAll(bus Bus, consume bool, types ...string) (Terminator, error)
	SubscribeOne(bus Bus, consume bool, id int64, types ...string) (Terminator, error)
}

type Processor interface {
	Process(global, local context.Context, timeout time.Duration, url string) ([]Tag, error)
	Capacity() int
	IsReady() bool
}

type Ticker interface {
	OnTick(t time.Time) error
}

type Client interface {
	Cancel(id int64) error
	Get(id int64) (*Info, error)
	Identify(url string, priority int) (int64, error)

	OnBeat(id int64) error
	OnCancel(id int64) error
	OnDone(id int64, tags []Tag) error
	OnError(id int64, code ErrorCode, reason string) error
}
