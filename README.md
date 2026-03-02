# sgt — Simple GasTown

Lightweight tmux-managed sub-agent orchestration. One session, one orchestrator, N agent panes. Agents communicate via a shared event bus and mail.

## Install

```bash
git clone https://github.com/cwolfe007/simple-gt.git
cd simple-gt
make build-sgt
cp sgt ~/.local/bin/   # or wherever you keep binaries
```

Requires: Go 1.24+, tmux 3.0+, [beads (bd)](https://github.com/steveyegge/beads)

## How it works

```
sgt-demo (tmux session)
├── pane 0  orchestrator shell
├── pane 1  agent: coder
└── pane 2  agent: analyst
```

- **`sgt start`** creates the orchestrator tmux session
- **`sgt agent spawn NAME`** splits a new pane and starts the agent command there
- **`sgt bus`** is an append-only JSONL event log (`.sgt/bus.jsonl`) agents use to broadcast state
- **`sgt mail`** sends direct messages between agents, backed by beads

## Quick start

```bash
# Start the orchestrator session
sgt start --session myproject

# Spawn agents into panes
sgt agent spawn coder   --cmd "claude --dangerously-skip-permissions"
sgt agent spawn analyst --cmd "claude --dangerously-skip-permissions"

# Attach to see all panes
tmux attach -t myproject

# Send work to an agent
sgt mail send sgt/coder "Implement the login endpoint" --body "See spec.md"

# Watch what agents are doing
sgt bus watch

# Agents report back via the bus
sgt bus publish task.completed --source coder --data '{"pr": 42}'

# Check agent status
sgt agent list

# Tear everything down
sgt stop
```

## Command reference

```
sgt start [--session NAME] [--cmd CMD] [--interactive]
    Create the orchestrator tmux session.
    --interactive  attach immediately after starting

sgt stop
    Kill the orchestrator session and all agent panes.

sgt agent spawn NAME [--cmd CMD] [--dir DIR]
    Add a new pane to the session running CMD.
    Default cmd: claude --dangerously-skip-permissions

sgt agent list
    Show all agents with pane ID and status.

sgt agent kill NAME
    Kill an agent's pane and remove it from the registry.

sgt agent nudge NAME MESSAGE
    Type MESSAGE into the agent's pane and press Enter.

sgt bus publish TYPE [--source NAME] [--data JSON]
    Append an event to .sgt/bus.jsonl.

sgt bus watch [--filter TYPE_OR_SOURCE] [--interval DURATION]
    Tail the event stream in real time (Ctrl-C to stop).

sgt bus list [--filter TYPE_OR_SOURCE]
    Print all past events.

sgt mail send TO SUBJECT [--body TEXT] [--priority LEVEL] [--from IDENTITY]
    Send a direct message to an agent.
    Addresses: overseer, sgt/NAME

sgt mail inbox [--identity AGENT] [--all]
    Show unread messages (default identity: overseer).

sgt mail read ID [--identity AGENT]
    Read a message and mark it read.
```

## Event bus conventions

Events are free-form but these types are recommended:

| Type | Meaning |
|---|---|
| `agent.started` | Agent finished initialising |
| `task.started` | Agent began a unit of work |
| `task.completed` | Agent finished successfully |
| `task.failed` | Agent hit an error |
| `progress` | Intermediate status update |

## State files

```
.sgt/config.json    orchestrator session info
.sgt/agents.json    agent registry
.sgt/bus.jsonl      event bus (append-only)
.beads/             mail + task database (shared with gt)
```

## License

MIT
