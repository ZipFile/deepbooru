package nurse

import (
	"context"
	"os/exec"
	"syscall"
	"time"
)

func TerminateOrKill(cmd *exec.Cmd, cancel context.CancelFunc, timeout time.Duration) (killed bool, err error) {
	cmd.Process.Signal(syscall.SIGTERM)

	go func() {
		time.Sleep(timeout)
		killed = true
		cancel()
	}()

	err = cmd.Wait()

	cancel()

	return
}
