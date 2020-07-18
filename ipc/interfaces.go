package deepbooru_ipc

import (
	"deepbooru"
)

type Message struct {
	URL   string          `json:"url,omitempty"`
	Tags  []deepbooru.Tag `json:"tags,omitempty"`
	Error string          `json:"error,omitempty"`

	Shutdown bool `json:"shutdown,omitempty"`
}

type Bus interface {
	In() chan<- Message
	Out() <-chan Message
	Interrupt()
	IsReady() bool
}
