package nurse

import (
	"bufio"
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestTerminateOrKillOK(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	t.Cleanup(cancel)

	cmd := exec.CommandContext(ctx, "sleep", "5")
	err := cmd.Start()

	if err != nil {
		t.Errorf("Failed to start subprocess: %s", err)
		return
	}

	killed, _ := TerminateOrKill(cmd, cancel, 10*time.Second)

	if killed {
		t.Errorf("Killed")
	}

	if ctx.Err() == nil {
		t.Errorf("Context is not cancelled")
	}
}

func TestTerminateOrKillTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	t.Cleanup(cancel)

	cmd := exec.CommandContext(ctx, "sh", "-c", "trap '' TERM; echo ready; exec sleep 5")
	stdout, err := cmd.StdoutPipe()

	if err != nil {
		t.Errorf("Failed to create stdout pipe: %s", err)
		return
	}

	err = cmd.Start()

	if err != nil {
		t.Errorf("Failed to start subprocess: %s", err)
		return
	}

	// make sure trap is set
	scanner := bufio.NewScanner(stdout)

	if scanner.Scan() {
		s := scanner.Text()

		if s != "ready" {
			t.Errorf("Unexpected command ouput: %s", s)
			return
		}
	}

	killed, _ := TerminateOrKill(cmd, cancel, time.Second)

	if !killed {
		t.Errorf("Finished succesfully")
	}

	if ctx.Err() == nil {
		t.Errorf("Context is not cancelled")
	}
}
