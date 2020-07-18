package deepbooru_ipc

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"deepbooru"
)

var testIO = `{"url":"http://example.com/test.jpg"}
{"tags":[{"name":"test","score":1},{"name":"half","score":0.5}]}
{"error":"not found"}
{"shutdown":true}
`
var testMessages = []Message{
	Message{URL: "http://example.com/test.jpg"},
	Message{Tags: []deepbooru.Tag{deepbooru.Tag{"test", 1.0}, deepbooru.Tag{"half", 0.5}}},
	Message{Error: "not found"},
	Message{Shutdown: true},
}

func TestTranslateReaderOK(t *testing.T) {
	buff := strings.NewReader(testIO)
	messages := make([]Message, 0, 4)
	out := make(chan Message)

	go func() {
		TranslateReader(buff, out)
		close(out)
	}()

	for m := range out {
		messages = append(messages, m)
	}

	if !reflect.DeepEqual(messages, testMessages) {
		t.Errorf("messages: %#v, expected: %#v", messages, testMessages)
	}
}

func TestTranslateReaderGarbage(t *testing.T) {
	buff := strings.NewReader(`{"error":"not found"}
garbage
{"shutdown":true}
stack trace`)
	out := make(chan Message)
	messages := make([]Message, 0, 2)
	expected := []Message{testMessages[2], testMessages[3]}

	go func() {
		TranslateReader(buff, out)
		close(out)
	}()

	for m := range out {
		messages = append(messages, m)
	}

	if !reflect.DeepEqual(messages, expected) {
		t.Errorf("messages: %#v, expected: %#v", messages, expected)
	}
}

func TestTranslateReaderPanic(t *testing.T) {
	defer func() { recover() }()

	TranslateReader(nil, nil)

	t.Errorf("did not panic")
}

func TestTranslateWriterOK(t *testing.T) {
	var builder strings.Builder
	in := make(chan Message, len(testMessages))

	for _, m := range testMessages {
		in <- m
	}

	close(in)

	TranslateWriter(&builder, in)

	result := builder.String()

	if result != testIO {
		t.Errorf("result: %#v, expected: %#v", result, testIO)
	}
}

type testWriter struct {
	Error error
	N     int
}

func (w *testWriter) Write(data []byte) (int, error) {
	if w.N <= 0 {
		return 0, w.Error
	}

	w.N--

	return len(data), nil
}

func TestTranslateWriterClosed(t *testing.T) {
	w := &testWriter{os.ErrClosed, 0}
	in := make(chan Message, 1)

	in <- Message{}

	close(in)

	TranslateWriter(w, in)
}

func TestTranslateWriterPanic(t *testing.T) {
	in := make(chan Message, 1)

	in <- Message{}

	close(in)

	defer func() { recover() }()

	TranslateWriter(nil, in)

	t.Errorf("did not panic")
}
