package deepbooru_ipc

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
)

func TranslateReader(r io.Reader, out chan<- Message) {
	var err error
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		var m Message
		err = json.Unmarshal(scanner.Bytes(), &m)

		if err != nil {
			log.Printf("failed to decode message: %s", err)
			continue
		}

		out <- m
	}

	err = scanner.Err()

	if err == nil || errors.Is(err, os.ErrClosed) {
		return
	}

	panic(err)
}

func TranslateWriter(w io.Writer, in <-chan Message) {
	var err error
	encoder := json.NewEncoder(w)

	for m := range in {
		err = encoder.Encode(&m)

		if err == nil {
			continue
		} else if errors.Is(err, os.ErrClosed) {
			return
		}

		panic(err)
	}
}
