package sgt

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// runTmux executes a tmux command and returns trimmed stdout.
// This is needed for tmux operations not exposed by the tmux.Tmux type
// (e.g. split-window with pane ID output).
func runTmux(args ...string) (string, error) {
	allArgs := append([]string{"-u"}, args...)
	cmd := exec.Command("tmux", allArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tmux %s: %s", args[0], strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}
