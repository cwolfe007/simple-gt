// sgt is the Simple GasTown CLI: a lightweight tmux-managed sub-agent system.
//
// Usage:
//
//	sgt start [--interactive] [--session NAME] [--cmd COMMAND]
//	sgt stop  [--session NAME]
//	sgt agent spawn NAME [--mode interactive|background] [--cmd COMMAND] [--dir DIR]
//	sgt agent list
//	sgt agent kill NAME
//	sgt agent nudge NAME MESSAGE
//	sgt bus publish TYPE [--source NAME] [--data JSON]
//	sgt bus watch [--filter TYPE_OR_SOURCE]
//	sgt bus list
//	sgt mail send TO SUBJECT [--body TEXT] [--priority normal|high|urgent]
//	sgt mail inbox
package main

import (
	"os"

	"github.com/steveyegge/gastown/internal/sgtcmd"
)

func main() {
	if err := sgtcmd.Execute(); err != nil {
		os.Exit(1)
	}
}
