package subprocess_processor

import (
	"bufio"
	"errors"
	"io"
	"log"
	"os"
)

var newline = []byte{0x0A}

func TranslateReader(r io.Reader, out chan<- Message) {
	var m *Message
	var err error
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		m, err = Decode(scanner.Bytes())

		if err != nil {
			log.Printf("failed to decode message: %s", err)
			return
		}

		out <- *m
	}

	err = scanner.Err()

	if err == nil || errors.Is(err, os.ErrClosed) {
		return
	}

	panic(err)
}

func write(w io.Writer, m *Message) error {
	if m == nil {
		return nil
	}

	data, err := encode(m)

	if err != nil {
		return err
	}

	_, err = w.Write(data)

	if err != nil {
		return err
	}

	_, err = w.Write(newline)

	return err
}

func TranslateWriter(w io.Writer, in <-chan Message) {
	for m := range in {
		err := write(w, &m)

		if err == nil {
			continue
		}

		if errors.Is(err, os.ErrClosed) {
			return
		}

		panic(err)
	}
}
