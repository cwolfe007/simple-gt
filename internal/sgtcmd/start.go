package sgtcmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/sgt"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the orchestrator tmux session",
	Long: `start creates a new tmux session for the orchestrator.

In interactive mode (--interactive), run 'sgt attach' or use tmux directly to view the session.
In background mode (default), the orchestrator session is detached.

Examples:
  sgt start                          # background orchestrator session named "sgt"
  sgt start --interactive            # interactive session (attach to view)
  sgt start --session myproject      # custom session name
  sgt start --cmd "claude --dangerously-skip-permissions"  # run agent as orchestrator
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		interactive, _ := cmd.Flags().GetBool("interactive")
		session, _ := cmd.Flags().GetString("session")
		orchCmd, _ := cmd.Flags().GetString("cmd")

		wd := workDir()
		orch, err := sgt.Start(wd, session, interactive, orchCmd)
		if err != nil {
			return err
		}

		fmt.Printf("✓ Orchestrator started: session=%q dir=%s\n", orch.Session(), wd)
		fmt.Printf("  Attach: tmux attach -t %s\n", orch.Session())
		fmt.Printf("  State:  %s/config.json\n", orch.SgtDir())

		if interactive {
			fmt.Println("\nAttaching to session (detach with Ctrl-b d) ...")
			return orch.Attach()
		}
		return nil
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the orchestrator session and all agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		orch, err := sgt.LoadOrchestrator(workDir())
		if err != nil {
			return err
		}
		if err := orch.Stop(); err != nil {
			return err
		}
		fmt.Println("✓ Orchestrator stopped")
		return nil
	},
}

func init() {
	startCmd.Flags().Bool("interactive", false, "Interactive mode: attach to session on start")
	startCmd.Flags().String("session", "sgt", "Tmux session name")
	startCmd.Flags().String("cmd", "", "Command to run as the orchestrator (default: shell)")
}
