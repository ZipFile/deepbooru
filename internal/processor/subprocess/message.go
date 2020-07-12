package subprocess_processor

import (
	"encoding/json"

	"deepbooru"
)

type Message struct {
	URL   string          `json:"url,omitempty"`
	Tags  []deepbooru.Tag `json:"tags,omitempty"`
	Error string          `json:"error,omitempty"`

	Shutdown bool `json:"shutdown,omitempty"`
}

func encode(m *Message) ([]byte, error) {
	return json.Marshal(m)
}

func Encode(m Message) ([]byte, error) {
	return encode(&m)
}

func Decode(data []byte) (*Message, error) {
	m := &Message{}
	err := json.Unmarshal(data, m)

	if err != nil {
		return nil, err
	}

	return m, nil
}
