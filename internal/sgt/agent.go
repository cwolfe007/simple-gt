// Package sgt provides simplified agent management for simple-gt.
//
// All agents run as panes inside the shared orchestrator tmux session.
// Agent state is persisted in .sgt/agents.json.
package sgt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Status of an agent.
type Status string

const (
	StatusRunning Status = "running"
	StatusStopped Status = "stopped"
)

// Agent represents a managed sub-agent pane.
type Agent struct {
	Name      string    `json:"name"`
	Session   string    `json:"session"`  // tmux session name (orchestrator session)
	Pane      string    `json:"pane"`     // tmux pane ID
	Cmd       string    `json:"cmd"`      // command running in the pane
	WorkDir   string    `json:"work_dir"` // working directory
	Status    Status    `json:"status"`
	StartedAt time.Time `json:"started_at"`
}

// Registry manages the set of known agents.
type Registry struct {
	sgtDir string
}

type registryFile struct {
	Agents map[string]*Agent `json:"agents"`
}

// NewRegistry creates a Registry rooted at sgtDir.
func NewRegistry(sgtDir string) *Registry {
	return &Registry{sgtDir: sgtDir}
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

// Spawn creates a new pane in the orchestrator session and registers the agent.
// cmd is the command to run; workDir is the working directory (empty = inherit).
func (r *Registry) Spawn(name, sessionName, cmd, workDir string) (*Agent, error) {
	rf, err := r.load()
	if err != nil {
		return nil, err
	}

	if _, exists := rf.Agents[name]; exists {
		return nil, fmt.Errorf("agent %q already exists", name)
	}

	paneID, err := spawnPane(sessionName, name, cmd, workDir)
	if err != nil {
		return nil, err
	}

	ag := &Agent{
		Name:      name,
		Session:   sessionName,
		Pane:      paneID,
		Cmd:       cmd,
		WorkDir:   workDir,
		Status:    StatusRunning,
		StartedAt: time.Now(),
	}

	rf.Agents[name] = ag
	if err := r.save(rf); err != nil {
		return nil, err
	}
	return ag, nil
}

// spawnPane splits the window to add a new pane, returns the pane ID.
func spawnPane(session, name, cmd, workDir string) (string, error) {
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

	// Label the pane with the agent name
	_, _ = runTmux("select-pane", "-t", out, "-T", name)
	return out, nil
}

// Kill kills the agent's pane and removes it from the registry.
func (r *Registry) Kill(name string) error {
	rf, err := r.load()
	if err != nil {
		return err
	}

	ag, exists := rf.Agents[name]
	if !exists {
		return fmt.Errorf("agent %q not found", name)
	}

	if ag.Pane != "" {
		_, _ = runTmux("kill-pane", "-t", ag.Pane)
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

// NudgeAgent injects a message into the agent's pane (text + Enter).
func (r *Registry) NudgeAgent(name, message string) error {
	ag, err := r.Get(name)
	if err != nil {
		return err
	}
	if _, err := runTmux("send-keys", "-t", ag.Pane, "-l", message); err != nil {
		return fmt.Errorf("nudge send text: %w", err)
	}
	_, err = runTmux("send-keys", "-t", ag.Pane, "Enter")
	return err
}
