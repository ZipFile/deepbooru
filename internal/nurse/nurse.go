package nurse

import (
	"context"
	"errors"
	"log"
	"os"
	"os/exec"
	"time"

	"deepbooru"
	"deepbooru/ipc"
)

const MinAliveTime = 5 * time.Second
const MaxFailedRestarts = 5

type Nurse struct {
	Path    string
	Args    []string
	Environ []string

	KillTimeout time.Duration

	in  chan deepbooru_ipc.Message
	out chan deepbooru_ipc.Message

	interrupt chan bool

	nurseStdin    *os.File
	nurseStdout   *os.File
	processStdin  *os.File
	processStdout *os.File

	cmd *exec.Cmd

	started bool
	ready   bool

	now func() time.Time
}

func (n *Nurse) Run(ctx context.Context) error {
	if n.started {
		return deepbooru.ErrAlreadyRunning
	}

	err := n.init()

	if err != nil {
		return err
	}

	restart := true
	failedRestarts := 0
	var start, end time.Time

	for restart && failedRestarts < MaxFailedRestarts {
		start = n.now()
		restart = n.loop(ctx)
		end = n.now()

		if end.Sub(start) < MinAliveTime {
			failedRestarts++
		} else {
			failedRestarts = 0
		}

		log.Println("Restarting dead process...")
	}

	n.in <- deepbooru_ipc.Message{Shutdown: true}

	n.clean()

	if failedRestarts == MaxFailedRestarts {
		return errors.New("Failed to start process")
	}

	return nil
}

func (n *Nurse) loop(globalCtx context.Context) bool {
	done := make(chan bool)
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, n.Path, n.Args...)

	cmd.Env = n.Environ
	cmd.Stdin = n.processStdin
	cmd.Stdout = n.processStdout
	cmd.Stderr = os.Stderr

	n.cmd = cmd

	go n.run(cmd, done)

	select {
	case <-globalCtx.Done():
		killed, _ := TerminateOrKill(cmd, cancel, n.KillTimeout)

		if killed {
			log.Printf("Process %d killed", cmd.Process.Pid)
		}

		return false
	case started := <-done:
		if !started {
			return false
		}
	}

	return true
}

func (n *Nurse) run(cmd *exec.Cmd, done chan<- bool) {
	log.Println("exec", cmd.Path, cmd.Args)

	err := cmd.Start()

	if err != nil {
		log.Printf("Failed to start process")
		done <- false
		return
	}

	log.Println("started")

	n.ready = true
	err = cmd.Wait()
	n.ready = false

	if err == nil {
		log.Printf("Process %d finished without errors", cmd.Process.Pid)
	} else if eerr, ok := err.(*exec.ExitError); ok {
		log.Printf("Process %d finished with code %d", eerr.Pid(), eerr.ExitCode())
	} else {
		log.Printf("Process %d finished with error: %s", cmd.Process.Pid, err)
	}

	done <- true
}

func (n *Nurse) init() error {
	_, err := exec.LookPath(n.Path)

	if err != nil {
		return err
	}

	n.processStdin, n.nurseStdin, err = os.Pipe()

	if err != nil {
		return err
	}

	n.nurseStdout, n.processStdout, err = os.Pipe()

	if err != nil {
		return err
	}

	if n.now == nil {
		n.now = time.Now
	}

	n.in = make(chan deepbooru_ipc.Message)
	n.out = make(chan deepbooru_ipc.Message)
	n.interrupt = make(chan bool)
	n.started = true

	go deepbooru_ipc.TranslateWriter(n.nurseStdin, n.in)
	go deepbooru_ipc.TranslateReader(n.nurseStdout, n.out)
	go n.handleInterrupts()

	return nil
}

func (n *Nurse) clean() {
	close(n.interrupt)
	close(n.in)
	n.nurseStdout.Close()
	n.nurseStdin.Close()
	close(n.out)
	n.processStdin.Close()
	n.processStdout.Close()

	n.in, n.out = nil, nil
	n.interrupt = nil
	n.processStdin, n.nurseStdin = nil, nil
	n.nurseStdout, n.processStdout = nil, nil
	n.cmd = nil
	n.started = false
}

func (n *Nurse) handleInterrupts() {
	for _ = range n.interrupt {
		if n.cmd == nil || n.cmd.Process == nil {
			log.Printf("No process to send interrupt to")

			continue
		}

		err := n.cmd.Process.Signal(os.Interrupt)

		if err != nil {
			log.Printf("Failed to send interrupt signal: %s", err)
		}
	}
}

func (n *Nurse) Interrupt() {
	if n == nil {
		return
	}

	n.interrupt <- true
}

func (n *Nurse) IsReady() bool {
	return n != nil && n.started && n.ready
}

func (n *Nurse) In() chan<- deepbooru_ipc.Message {
	return n.in
}

func (n *Nurse) Out() <-chan deepbooru_ipc.Message {
	return n.out
}
