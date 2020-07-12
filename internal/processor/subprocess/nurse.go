package subprocess_processor

import (
	"context"
	"log"
	"os"
	"os/exec"
	"time"
)

type Nurse struct {
	path    string
	args    []string
	environ []string

	in  chan Message
	out chan Message

	nurseStdin    *os.File
	nurseStdout   *os.File
	processStdin  *os.File
	processStdout *os.File

	globalCtx   context.Context
	killTimeout time.Duration

	started bool
	ready   bool
}

func NewNurse(path string, args []string, environ []string, ctx context.Context, killTimeout time.Duration) (*Nurse, error) {
	_, err := exec.LookPath(path)

	if err != nil {
		return nil, err
	}

	ir, iw, err := os.Pipe()

	if err != nil {
		return nil, err
	}

	or, ow, err := os.Pipe()

	if err != nil {
		return nil, err
	}

	return &Nurse{
		path:    path,
		args:    args,
		environ: environ,

		in:  make(chan Message),
		out: make(chan Message),

		nurseStdin:    iw,
		nurseStdout:   or,
		processStdin:  ir,
		processStdout: ow,

		globalCtx:   ctx,
		killTimeout: killTimeout,
	}, nil
}

func (n *Nurse) Start() {
	if n.started {
		return
	}

	n.started = true

	go TranslateWriter(n.nurseStdin, n.in)
	go TranslateReader(n.nurseStdout, n.out)

	for n.loop() {
		log.Println("Restarting dead process...")
	}

	n.In() <- Message{Shutdown: true}

	n.clean()

	n.started = false
}

func (n *Nurse) loop() bool {
	done := make(chan bool)
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, n.path, n.args...)

	cmd.Env = n.environ
	cmd.Stdin = n.processStdin
	cmd.Stdout = n.processStdout
	cmd.Stderr = os.Stderr

	go n.run(cmd, done)

	select {
	case <-n.globalCtx.Done():
		killed := TerminateOrKill(cmd, cancel, n.killTimeout)

		if killed {
			log.Println("Process %d killed", cmd.Process.Pid)
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

func (n *Nurse) clean() {
	close(n.in)
	n.nurseStdout.Close()
	n.nurseStdin.Close()
	close(n.out)
	n.processStdin.Close()
	n.processStdout.Close()
}

func (n *Nurse) IsReady() bool {
	if n == nil {
		return false
	}

	return n.ready
}

func (n *Nurse) In() chan<- Message {
	return n.in
}

func (n *Nurse) Out() <-chan Message {
	return n.out
}
