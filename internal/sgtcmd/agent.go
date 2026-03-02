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
	Short: "Spawn a new sub-agent pane",
	Long: `spawn creates a new pane in the orchestrator session for the named agent.

Requires 'sgt start' to have been run first.

Examples:
  sgt agent spawn worker1
  sgt agent spawn coder --cmd "claude --dangerously-skip-permissions"
  sgt agent spawn analyst --cmd "gemini" --dir /workspace/myproject
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		agentCmd2, _ := cmd.Flags().GetString("cmd")
		dir, _ := cmd.Flags().GetString("dir")

		if agentCmd2 == "" {
			agentCmd2 = "claude --dangerously-skip-permissions"
		}
		if dir == "" {
			dir = workDir()
		}

		orch, err := sgt.LoadOrchestrator(workDir())
		if err != nil {
			return fmt.Errorf("no orchestrator running (run 'sgt start' first): %w", err)
		}

		reg := sgt.NewRegistry(sgtDir())
		ag, err := reg.Spawn(name, orch.Session(), agentCmd2, dir)
		if err != nil {
			return err
		}

		fmt.Printf("✓ Agent spawned: %s\n", ag.Name)
		fmt.Printf("  Session: %s\n", ag.Session)
		fmt.Printf("  Pane:    %s\n", ag.Pane)
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
		fmt.Fprintln(w, "NAME\tSTATUS\tSESSION\tPANE\tSTARTED\tCMD")
		for _, ag := range agents {
			started := ag.StartedAt.Format(time.DateTime)
			c := ag.Cmd
			if len(c) > 40 {
				c = c[:37] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				ag.Name, ag.Status, ag.Session, ag.Pane, started, c)
		}
		return w.Flush()
	},
}

var agentKillCmd = &cobra.Command{
	Use:   "kill NAME",
	Short: "Kill a sub-agent pane",
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
	Short: "Send a message to an agent's pane",
	Long:  `nudge types the message into the agent's pane and presses Enter.`,
	Args:  cobra.ExactArgs(2),
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

	agentSpawnCmd.Flags().String("cmd", "", "Command to run in the pane (default: claude --dangerously-skip-permissions)")
	agentSpawnCmd.Flags().String("dir", "", "Working directory for the agent (default: current dir)")
}
