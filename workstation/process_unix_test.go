//go:build !windows

package workstation

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestPrepareLocalCommandCancelsWholeProcessGroup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	command := exec.CommandContext(
		ctx,
		"/bin/sh",
		"-c",
		"sleep 60 & child=$!; printf '%s' \"$child\"; wait",
	)
	prepareLocalCommand(command)
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	childBytes := make([]byte, 32)
	count, err := stdout.Read(childBytes)
	if err != nil {
		t.Fatalf("read child pid: %v", err)
	}
	childPID, err := strconv.Atoi(strings.TrimSpace(string(childBytes[:count])))
	if err != nil {
		t.Fatalf("parse child pid: %v", err)
	}
	cancel()
	_ = command.Wait()

	deadline := time.Now().Add(2 * time.Second)
	for {
		err = syscall.Kill(childPID, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("child process %d survived group cancellation: %v", childPID, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
