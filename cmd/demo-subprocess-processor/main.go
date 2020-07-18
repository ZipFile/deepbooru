package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"deepbooru/internal/nurse"
	"deepbooru/ipc"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Printf("Usage: %s command [args...]\n", os.Args[0])
		return
	}

	sigs := make(chan os.Signal)
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	var taskCtx context.Context
	var taskCancel context.CancelFunc

	signal.Notify(sigs, os.Interrupt)

	n := nurse.Nurse{
		Path:        os.Args[1],
		Args:        os.Args[2:],
		Environ:     os.Environ(),
		KillTimeout: 15 * time.Second,
	}

	go func() {
		defer close(done)

		fmt.Println("Starting worker...")
		err := n.Run(ctx)

		if err != nil {
			panic(err)
		}
	}()

	go func() {
		stop := false

		for _ = range sigs {
			if taskCancel == nil {
				if stop {
					cancel()
				} else {
					stop = true
				}
			} else {
				stop = false
				fmt.Println("Aborting task")
				taskCancel()
			}
		}
	}()

	p := deepbooru_ipc.Processor{Bus: &n}
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		url := scanner.Text()

		if url == "" {
			fmt.Println("Terminating")
			break
		} else if !p.IsReady() {
			fmt.Println("Not ready yet")
			continue
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
		taskCancel = nil
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	cancel()
	<-done
}
