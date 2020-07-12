package subprocess_processor

import (
	"context"
	"os/exec"
	"syscall"
	"time"
)

func TerminateOrKill(cmd *exec.Cmd, cancel context.CancelFunc, timeout time.Duration) (killed bool) {
	cmd.Process.Signal(syscall.SIGTERM)

	done := make(chan struct{})

	go func() {
		select {
		case <-time.After(timeout):
			killed = true
		case <-done:
		}

		cancel()
	}()

	cmd.Wait()
	close(done)

	return
}
