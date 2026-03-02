// Package sgtcmd implements the sgt CLI commands.
package sgtcmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sgt",
	Short: "Simple GasTown: lightweight tmux-managed sub-agent orchestration",
	Long: `sgt manages AI sub-agents in tmux panes with a shared event bus and mail system.

Two modes:
  interactive   Each agent gets a pane in the shared tmux session (visible to user)
  background    Each agent gets its own detached tmux session

Agents communicate via:
  mail    Direct messages backed by beads (gt mail compatible)
  bus     Append-only JSONL event stream at .sgt/bus.jsonl
`,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// sgtDir returns the .sgt directory for the current working directory.
func sgtDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return ".sgt"
	}
	return filepath.Join(wd, ".sgt")
}

// workDir returns the current working directory, exiting on error.
func workDir() string {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: cannot determine working directory:", err)
		os.Exit(1)
	}
	return wd
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(busCmd)
	rootCmd.AddCommand(mailCmd)
}
