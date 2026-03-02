package sgtcmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/mail"
)

var mailCmd = &cobra.Command{
	Use:   "mail",
	Short: "Send and receive mail between agents",
	Long: `mail provides direct messaging between agents, backed by the beads database.

Agent identities in sgt follow the same convention as gastown:
  orchestrator     the main orchestrator
  sgt/worker1      a sub-agent named "worker1"
  overseer         the human operator

Messages are stored in the .beads database and can be queried with the 'bd' CLI too.
`,
}

var mailSendCmd = &cobra.Command{
	Use:   "send TO SUBJECT",
	Short: "Send a mail message",
	Long: `Send a message to an agent or the orchestrator.

Examples:
  sgt mail send sgt/worker1 "Start analysing data"
  sgt mail send sgt/worker1 "Your task" --body "Please analyse the CSV at data.csv"
  sgt mail send orchestrator "Done" --body "Task complete" --priority high
`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		to := args[0]
		subject := args[1]
		body, _ := cmd.Flags().GetString("body")
		priorityStr, _ := cmd.Flags().GetString("priority")
		from, _ := cmd.Flags().GetString("from")

		if from == "" {
			from = "overseer"
		}

		msg := mail.NewMessage(from, to, subject, body)
		msg.Priority = mail.ParsePriority(priorityStr)

		router := mail.NewRouter(workDir())
		if err := router.Send(msg); err != nil {
			return fmt.Errorf("send mail: %w", err)
		}

		fmt.Printf("✓ Message sent: id=%s to=%s subject=%q\n", msg.ID, to, subject)
		return nil
	},
}

var mailInboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "Show unread messages",
	Long: `inbox lists unread messages for the given identity.

Examples:
  sgt mail inbox                           # inbox for "overseer"
  sgt mail inbox --identity sgt/worker1    # inbox for a sub-agent
  sgt mail inbox --all                     # show all messages (read + unread)
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		identity, _ := cmd.Flags().GetString("identity")
		showAll, _ := cmd.Flags().GetBool("all")

		if identity == "" {
			identity = "overseer"
		}

		wd := workDir()
		mb := mail.NewMailboxFromAddress(identity, wd)

		var messages []*mail.Message
		var err error
		if showAll {
			messages, err = mb.List()
		} else {
			messages, err = mb.ListUnread()
		}
		if err != nil {
			return fmt.Errorf("inbox: %w", err)
		}

		if len(messages) == 0 {
			fmt.Printf("No %smessages for %s\n", map[bool]string{false: "unread "}[showAll], identity)
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "📬 Inbox: %s (%d messages)\n\n", identity, len(messages))
		fmt.Fprintln(w, "  ID\tFROM\tSUBJECT\tDATE")
		for _, m := range messages {
			unread := "  "
			if !m.Read {
				unread = "● "
			}
			date := m.Timestamp.Format(time.DateTime)
			from := m.From
			if len(from) > 20 {
				from = from[:17] + "..."
			}
			subj := m.Subject
			if len(subj) > 50 {
				subj = subj[:47] + "..."
			}
			fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\n", unread, m.ID, from, subj, date)
		}
		return w.Flush()
	},
}

var mailReadCmd = &cobra.Command{
	Use:   "read ID",
	Short: "Read a message by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		identity, _ := cmd.Flags().GetString("identity")
		if identity == "" {
			identity = "overseer"
		}

		wd := workDir()
		mb := mail.NewMailboxFromAddress(identity, wd)
		msgs, err := mb.List()
		if err != nil {
			return err
		}

		id := args[0]
		for _, m := range msgs {
			if m.ID == id {
				fmt.Printf("From:    %s\n", m.From)
				fmt.Printf("To:      %s\n", m.To)
				fmt.Printf("Subject: %s\n", m.Subject)
				fmt.Printf("Date:    %s\n", m.Timestamp.Format(time.DateTime))
				fmt.Printf("ID:      %s\n", m.ID)
				fmt.Println()
				fmt.Println(m.Body)
				// Mark as read
				_ = mb.MarkRead(id)
				return nil
			}
		}
		return fmt.Errorf("message %q not found", id)
	},
}

func init() {
	mailCmd.AddCommand(mailSendCmd)
	mailCmd.AddCommand(mailInboxCmd)
	mailCmd.AddCommand(mailReadCmd)

	mailSendCmd.Flags().String("body", "", "Message body")
	mailSendCmd.Flags().String("priority", "normal", "Priority: low, normal, high, urgent")
	mailSendCmd.Flags().String("from", "", "Sender identity (default: overseer)")

	mailInboxCmd.Flags().String("identity", "", "Agent identity to check (default: overseer)")
	mailInboxCmd.Flags().Bool("all", false, "Show all messages, not just unread")

	mailReadCmd.Flags().String("identity", "", "Agent identity (default: overseer)")
}
