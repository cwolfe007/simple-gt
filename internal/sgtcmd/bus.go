package sgtcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/bus"
)

var busCmd = &cobra.Command{
	Use:   "bus",
	Short: "Interact with the event bus",
	Long: `The event bus is an append-only JSONL stream at .sgt/bus.jsonl.

Any agent can publish events; any agent (or the orchestrator) can watch the stream.
Events carry a source name, a type (dot-separated, e.g. "task.completed"), and
optional JSON data.
`,
}

var busPublishCmd = &cobra.Command{
	Use:   "publish TYPE",
	Short: "Publish an event to the bus",
	Long: `Publish an event of the given type.

Examples:
  sgt bus publish task.started --source worker1
  sgt bus publish task.completed --source worker1 --data '{"result":"ok","lines":42}'
  sgt bus publish agent.error --source coder --data '{"msg":"rate limited"}'
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eventType := args[0]
		source, _ := cmd.Flags().GetString("source")
		dataStr, _ := cmd.Flags().GetString("data")

		if source == "" {
			// Default to hostname or "unknown"
			source = "unknown"
			if h, err := os.Hostname(); err == nil {
				source = h
			}
		}

		b := bus.New(sgtDir())

		var data any
		if dataStr != "" {
			var raw json.RawMessage
			if err := json.Unmarshal([]byte(dataStr), &raw); err != nil {
				return fmt.Errorf("--data must be valid JSON: %w", err)
			}
			data = raw
		}

		ev, err := b.Publish(source, eventType, data)
		if err != nil {
			return err
		}

		fmt.Printf("✓ Event published: id=%s type=%s source=%s\n", ev.ID, ev.Type, ev.Source)
		return nil
	},
}

var busWatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch the event bus stream (follow mode)",
	Long: `watch tails the event bus, printing new events as they arrive.

Examples:
  sgt bus watch
  sgt bus watch --filter task.completed
  sgt bus watch --filter worker1
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filter, _ := cmd.Flags().GetString("filter")
		interval, _ := cmd.Flags().GetDuration("interval")

		b := bus.New(sgtDir())
		fmt.Fprintf(os.Stderr, "Watching %s (Ctrl-C to stop) ...\n", b.Path())
		return b.Watch(os.Stdout, interval, filter)
	},
}

var busListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all events on the bus",
	Long: `list prints all events from the beginning of the bus log.

Examples:
  sgt bus list
  sgt bus list --filter task
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filter, _ := cmd.Flags().GetString("filter")
		b := bus.New(sgtDir())
		events, err := b.ReadAll()
		if err != nil {
			return err
		}

		if len(events) == 0 {
			fmt.Println("Bus is empty.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tTIME\tSOURCE\tTYPE\tDATA")
		for _, ev := range events {
			if filter != "" && ev.Type != filter && ev.Source != filter {
				continue
			}
			data := ""
			if ev.Data != nil {
				data = string(ev.Data)
				if len(data) > 60 {
					data = data[:57] + "..."
				}
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				ev.ID,
				ev.Timestamp.Format(time.DateTime),
				ev.Source,
				ev.Type,
				data,
			)
		}
		return w.Flush()
	},
}

func init() {
	busCmd.AddCommand(busPublishCmd)
	busCmd.AddCommand(busWatchCmd)
	busCmd.AddCommand(busListCmd)

	busPublishCmd.Flags().String("source", "", "Source agent name (default: hostname)")
	busPublishCmd.Flags().String("data", "", "JSON data payload (optional)")

	busWatchCmd.Flags().String("filter", "", "Filter by event type or source name")
	busWatchCmd.Flags().Duration("interval", 500*time.Millisecond, "Poll interval")

	busListCmd.Flags().String("filter", "", "Filter by event type or source name")
}
