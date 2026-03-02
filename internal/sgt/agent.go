// Package sgt provides simplified agent management for simple-gt.
//
// There are two execution modes:
//
//   - Interactive: all agents share a single tmux session; each agent
//     gets its own pane. The user can switch between panes normally.
//
//   - Background: each agent gets its own tmux session. Useful for
//     headless / non-interactive workflows.
//
// Agent state is persisted in .sgt/agents.json.
package sgt

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/steveyegge/gastown/internal/tmux"
)

// Mode controls how an agent is hosted.
type Mode string

const (
	// ModeInteractive puts the agent in a pane inside the shared orchestrator session.
	ModeInteractive Mode = "interactive"

	// ModeBackground gives the agent its own detached tmux session.
	ModeBackground Mode = "background"
)

// Status of an agent.
type Status string

const (
	StatusRunning Status = "running"
	StatusStopped Status = "stopped"
)

// Agent represents a managed sub-agent.
type Agent struct {
	Name      string    `json:"name"`
	Mode      Mode      `json:"mode"`
	Session   string    `json:"session"`   // tmux session name
	Pane      string    `json:"pane"`      // tmux pane ID (interactive mode only)
	Cmd       string    `json:"cmd"`       // command running in pane/session
	WorkDir   string    `json:"work_dir"`  // working directory
	Status    Status    `json:"status"`
	StartedAt time.Time `json:"started_at"`
}

// Registry manages the set of known agents.
type Registry struct {
	sgtDir string
	t      *tmux.Tmux
}

type registryFile struct {
	Agents map[string]*Agent `json:"agents"`
}

// NewRegistry creates a Registry rooted at sgtDir.
func NewRegistry(sgtDir string) *Registry {
	return &Registry{
		sgtDir: sgtDir,
		t:      tmux.NewTmux(),
	}
}

func (r *Registry) filePath() string {
	return filepath.Join(r.sgtDir, "agents.json")
}

func (r *Registry) load() (*registryFile, error) {
	data, err := os.ReadFile(r.filePath())
	if os.IsNotExist(err) {
		return &registryFile{Agents: make(map[string]*Agent)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("registry: read: %w", err)
	}
	var rf registryFile
	if err := json.Unmarshal(data, &rf); err != nil {
		return nil, fmt.Errorf("registry: parse: %w", err)
	}
	if rf.Agents == nil {
		rf.Agents = make(map[string]*Agent)
	}
	return &rf, nil
}

func (r *Registry) save(rf *registryFile) error {
	if err := os.MkdirAll(r.sgtDir, 0o755); err != nil {
		return fmt.Errorf("registry: mkdir: %w", err)
	}
	data, err := json.MarshalIndent(rf, "", "  ")
	if err != nil {
		return fmt.Errorf("registry: marshal: %w", err)
	}
	return os.WriteFile(r.filePath(), data, 0o644)
}

// Spawn starts a new sub-agent.
//
// Interactive mode: creates a new pane in the named orchestrator session.
// Background mode: creates a new detached tmux session named "sgt-<name>".
//
// cmd is the command to run (e.g. "claude --dangerously-skip-permissions").
// workDir is the working directory; leave empty to inherit.
func (r *Registry) Spawn(name string, mode Mode, sessionName, cmd, workDir string) (*Agent, error) {
	rf, err := r.load()
	if err != nil {
		return nil, err
	}

	if _, exists := rf.Agents[name]; exists {
		return nil, fmt.Errorf("agent %q already exists", name)
	}

	ag := &Agent{
		Name:      name,
		Mode:      mode,
		Cmd:       cmd,
		WorkDir:   workDir,
		Status:    StatusRunning,
		StartedAt: time.Now(),
	}

	switch mode {
	case ModeInteractive:
		paneID, err := r.spawnPane(sessionName, name, cmd, workDir)
		if err != nil {
			return nil, err
		}
		ag.Session = sessionName
		ag.Pane = paneID

	case ModeBackground:
		sessName := "sgt-" + name
		if err := r.t.NewSessionWithCommand(sessName, workDir, cmd); err != nil {
			return nil, fmt.Errorf("spawn background agent %q: %w", name, err)
		}
		ag.Session = sessName

	default:
		return nil, fmt.Errorf("unknown mode %q", mode)
	}

	rf.Agents[name] = ag
	if err := r.save(rf); err != nil {
		return nil, err
	}
	return ag, nil
}

// spawnPane adds a new named pane to an existing tmux session.
func (r *Registry) spawnPane(session, name, cmd, workDir string) (string, error) {
	// Split window to create new pane
	args := []string{"split-window", "-t", session, "-P", "-F", "#{pane_id}"}
	if workDir != "" {
		args = append(args, "-c", workDir)
	}
	if cmd != "" {
		args = append(args, cmd)
	} else {
		args = append(args, "bash")
	}

	out, err := runTmux(args...)
	if err != nil {
		return "", fmt.Errorf("split-window in session %q: %w", session, err)
	}

	paneID := out
	// Rename the pane title to the agent name
	_, _ = runTmux("select-pane", "-t", paneID, "-T", name)

	return paneID, nil
}

// Kill stops an agent and removes it from the registry.
func (r *Registry) Kill(name string) error {
	rf, err := r.load()
	if err != nil {
		return err
	}

	ag, exists := rf.Agents[name]
	if !exists {
		return fmt.Errorf("agent %q not found", name)
	}

	switch ag.Mode {
	case ModeInteractive:
		if ag.Pane != "" {
			// Kill just this pane
			_, _ = runTmux("kill-pane", "-t", ag.Pane)
		}
	case ModeBackground:
		if err := r.t.KillSession(ag.Session); err != nil && !errors.Is(err, tmux.ErrSessionNotFound) {
			return fmt.Errorf("kill session %q: %w", ag.Session, err)
		}
	}

	delete(rf.Agents, name)
	return r.save(rf)
}

// List returns all registered agents.
func (r *Registry) List() ([]*Agent, error) {
	rf, err := r.load()
	if err != nil {
		return nil, err
	}
	agents := make([]*Agent, 0, len(rf.Agents))
	for _, ag := range rf.Agents {
		agents = append(agents, ag)
	}
	return agents, nil
}

// Get returns a single agent by name.
func (r *Registry) Get(name string) (*Agent, error) {
	rf, err := r.load()
	if err != nil {
		return nil, err
	}
	ag, ok := rf.Agents[name]
	if !ok {
		return nil, fmt.Errorf("agent %q not found", name)
	}
	return ag, nil
}

// NudgeAgent sends a message to an agent's tmux pane/session.
// For interactive agents, it sends to the specific pane.
// For background agents, it sends to the session.
func (r *Registry) NudgeAgent(name, message string) error {
	ag, err := r.Get(name)
	if err != nil {
		return err
	}
	switch ag.Mode {
	case ModeInteractive:
		if ag.Pane != "" {
			return r.t.NudgePane(ag.Pane, message)
		}
		return r.t.NudgeSession(ag.Session, message)
	case ModeBackground:
		return r.t.NudgeSession(ag.Session, message)
	}
	return fmt.Errorf("unknown mode %q", ag.Mode)
}
