package sgtcmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/sgt"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage sub-agents",
}

var agentSpawnCmd = &cobra.Command{
	Use:   "spawn NAME",
	Short: "Spawn a new sub-agent",
	Long: `spawn creates a new sub-agent.

Interactive mode (default when inside a running sgt session):
  Creates a new pane in the shared tmux session. The agent is immediately visible.

Background mode:
  Creates a detached tmux session named "sgt-<NAME>". Useful for automation.

Examples:
  sgt agent spawn worker1
  sgt agent spawn worker1 --mode background
  sgt agent spawn coder --cmd "claude --dangerously-skip-permissions" --dir /workspace/myproject
  sgt agent spawn analyst --mode background --cmd "gemini"
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		modeFlagStr, _ := cmd.Flags().GetString("mode")
		agentCmd2, _ := cmd.Flags().GetString("cmd")
		dir, _ := cmd.Flags().GetString("dir")

		// Default cmd: claude
		if agentCmd2 == "" {
			agentCmd2 = "claude --dangerously-skip-permissions"
		}

		// Default dir: working directory
		if dir == "" {
			dir = workDir()
		}

		// Determine mode: use flag if given, else check if we have a running config
		var mode sgt.Mode
		switch modeFlagStr {
		case "interactive":
			mode = sgt.ModeInteractive
		case "background":
			mode = sgt.ModeBackground
		default:
			// Auto-detect: if orchestrator config exists, use interactive
			orch, err := sgt.LoadOrchestrator(workDir())
			if err == nil && orch.Session() != "" {
				mode = sgt.ModeInteractive
			} else {
				mode = sgt.ModeBackground
			}
		}

		reg := sgt.NewRegistry(sgtDir())

		// Load session name from orchestrator config if interactive
		sessionName := "sgt"
		if mode == sgt.ModeInteractive {
			if orch, err := sgt.LoadOrchestrator(workDir()); err == nil {
				sessionName = orch.Session()
			}
		}

		ag, err := reg.Spawn(name, mode, sessionName, agentCmd2, dir)
		if err != nil {
			return err
		}

		fmt.Printf("✓ Agent spawned: %s\n", ag.Name)
		fmt.Printf("  Mode:    %s\n", ag.Mode)
		fmt.Printf("  Session: %s\n", ag.Session)
		if ag.Pane != "" {
			fmt.Printf("  Pane:    %s\n", ag.Pane)
		}
		fmt.Printf("  Cmd:     %s\n", ag.Cmd)
		fmt.Printf("  Dir:     %s\n", ag.WorkDir)
		return nil
	},
}

var agentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		reg := sgt.NewRegistry(sgtDir())
		agents, err := reg.List()
		if err != nil {
			return err
		}

		if len(agents) == 0 {
			fmt.Println("No agents registered.")
			return nil
		}

		sort.Slice(agents, func(i, j int) bool {
			return agents[i].StartedAt.Before(agents[j].StartedAt)
		})

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tMODE\tSTATUS\tSESSION\tSTARTED\tCMD")
		for _, ag := range agents {
			started := ag.StartedAt.Format(time.DateTime)
			cmd := ag.Cmd
			if len(cmd) > 40 {
				cmd = cmd[:37] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				ag.Name, ag.Mode, ag.Status, ag.Session, started, cmd)
		}
		return w.Flush()
	},
}

var agentKillCmd = &cobra.Command{
	Use:   "kill NAME",
	Short: "Kill a sub-agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reg := sgt.NewRegistry(sgtDir())
		if err := reg.Kill(args[0]); err != nil {
			return err
		}
		fmt.Printf("✓ Agent %q killed\n", args[0])
		return nil
	},
}

var agentNudgeCmd = &cobra.Command{
	Use:   "nudge NAME MESSAGE",
	Short: "Send a message to an agent's tmux pane",
	Long: `nudge injects a message into the agent's running tmux session/pane.
This is equivalent to typing the message in the pane.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		reg := sgt.NewRegistry(sgtDir())
		if err := reg.NudgeAgent(args[0], args[1]); err != nil {
			return err
		}
		fmt.Printf("✓ Nudged agent %q\n", args[0])
		return nil
	},
}

func init() {
	agentCmd.AddCommand(agentSpawnCmd)
	agentCmd.AddCommand(agentListCmd)
	agentCmd.AddCommand(agentKillCmd)
	agentCmd.AddCommand(agentNudgeCmd)

	agentSpawnCmd.Flags().String("mode", "", "Execution mode: interactive or background (auto-detected if omitted)")
	agentSpawnCmd.Flags().String("cmd", "", "Command to run in the agent pane/session (default: claude --dangerously-skip-permissions)")
	agentSpawnCmd.Flags().String("dir", "", "Working directory for the agent (default: current dir)")
}
