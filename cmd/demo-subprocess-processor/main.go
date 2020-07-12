package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"deepbooru/internal/processor/subprocess"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Printf("Usage: %s command [args...]\n", os.Args[0])
		return
	}

	sigs := make(chan os.Signal)
	path := os.Args[1]
	args := os.Args[2:]
	p := subprocess_processor.New(path, args, os.Environ(), 15*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	scanner := bufio.NewScanner(os.Stdin)
	var taskCtx context.Context
	var taskCancel context.CancelFunc
	done := make(chan struct{})

	signal.Notify(sigs, os.Interrupt)

	go func() {
		defer close(done)

		fmt.Println("Starting worker...")
		err := p.Run(ctx)

		if err != nil {
			panic(err)
		}
	}()

	go func() {
		for _ = range sigs {
			if taskCancel != nil {
				fmt.Println("Aborting task")
				taskCancel()
			}
		}
	}()

	for scanner.Scan() {
		url := scanner.Text()

		if url == "" {
			fmt.Println("Terminating")
			break
		}

		taskCtx, taskCancel = context.WithCancel(context.Background())
		tags, err := p.Process(ctx, taskCtx, 10*time.Second, url)

		if err != nil {
			fmt.Printf("Error: %s\n", err)
		}

		for _, tag := range tags {
			fmt.Println(tag.Name, tag.Score)
		}

		taskCancel()
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	cancel()
	<-done
}
