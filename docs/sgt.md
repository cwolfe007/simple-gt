# sgt — Simple GasTown

A lightweight tmux-managed sub-agent orchestration system.
No complex roles, no convoys, no rig hierarchy — just one orchestrator and N sub-agents.

## Concepts

### Orchestrator
The single coordinator process. Lives in a tmux session. Spawns agents, reads the event bus, delegates work via mail.

### Sub-agents
Named worker processes. Each gets either a **pane** (interactive) or a **session** (background).
Agents report progress by publishing events to the bus and/or sending mail.

### Event Bus
An append-only JSONL file at `.sgt/bus.jsonl`. Any agent can publish; any agent can watch.
Events have: `id`, `timestamp`, `source`, `type`, `data` (arbitrary JSON).

### Mail
Direct messages backed by the beads database (`.beads/`).
Compatible with the full `gt mail` command set. Agents use addresses like `sgt/worker1`.

## Modes

### Interactive
```
sgt start --interactive
```
- Creates one tmux session (default: `sgt`)
- Orchestrator in pane 0, each spawned agent in a new pane
- User sees all panes; switch with `Ctrl-b <arrow>` or `tmux select-pane`

### Background
```
sgt start
sgt agent spawn worker1 --mode background
```
- Orchestrator in a detached session
- Each agent gets its own detached session named `sgt-<name>`
- Communicate via mail + bus; capture output with `tmux capture-pane`

## Quick Start

```bash
# Build
make build-sgt

# Start orchestrator (interactive: you'll attach to tmux)
./sgt start --interactive --cmd "claude --dangerously-skip-permissions"

# (In another terminal or pane) spawn two workers
./sgt agent spawn coder   --cmd "claude --dangerously-skip-permissions"
./sgt agent spawn analyst --cmd "claude --dangerously-skip-permissions"

# Send work via mail
./sgt mail send sgt/coder "Implement feature X" --body "See spec.md"
./sgt mail send sgt/analyst "Analyse data.csv" --priority high

# Watch what agents are doing
./sgt bus watch

# Agents publish their own events
./sgt bus publish task.completed --source coder --data '{"result":"done"}'

# List all agents
./sgt agent list
```

## CLI Reference

```
sgt start [--interactive] [--session NAME] [--cmd COMMAND]
    Start the orchestrator tmux session.

sgt stop
    Stop the orchestrator and all registered agents.

sgt agent spawn NAME [--mode interactive|background] [--cmd CMD] [--dir DIR]
    Spawn a sub-agent pane (interactive) or session (background).

sgt agent list
    List all registered agents with status.

sgt agent kill NAME
    Kill a sub-agent and remove it from the registry.

sgt agent nudge NAME MESSAGE
    Inject a message into the agent's tmux pane/session.

sgt bus publish TYPE [--source NAME] [--data JSON]
    Publish an event to the bus.

sgt bus watch [--filter TYPE_OR_SOURCE] [--interval DURATION]
    Tail the event bus in real time.

sgt bus list [--filter TYPE_OR_SOURCE]
    Print all events from the start.

sgt mail send TO SUBJECT [--body TEXT] [--priority LEVEL] [--from IDENTITY]
    Send a mail message to an agent.

sgt mail inbox [--identity AGENT] [--all]
    Show unread messages.

sgt mail read ID [--identity AGENT]
    Read a specific message by ID.
```

## File Layout

```
.sgt/
  config.json     orchestrator session config
  agents.json     agent registry
  bus.jsonl       event bus (append-only)
.beads/           beads task/mail database (shared with gt)
```

## Agent Identity

| Address          | Meaning                        |
|------------------|--------------------------------|
| `overseer`       | Human operator (default sender)|
| `orchestrator`   | The sgt orchestrator process   |
| `sgt/worker1`    | Sub-agent named "worker1"      |

## Event Types (conventions)

| Type                  | Meaning                              |
|-----------------------|--------------------------------------|
| `agent.started`       | Agent finished initialisation        |
| `agent.stopped`       | Agent exited                         |
| `task.started`        | Agent began a unit of work           |
| `task.completed`      | Agent finished a unit of work        |
| `task.failed`         | Agent encountered an error           |
| `message.received`    | Agent read a mail message            |
| `progress`            | Intermediate progress update         |

Types are free-form strings; the above are conventions only.
