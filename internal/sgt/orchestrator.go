package sgt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/steveyegge/gastown/internal/tmux"
)

const defaultSession = "sgt"

// Config is persisted to .sgt/config.json.
type Config struct {
	Session     string `json:"session"`      // tmux session name for the orchestrator
	Interactive bool   `json:"interactive"`  // whether interactive mode is active
	WorkDir     string `json:"work_dir"`     // root working directory
	SgtDir      string `json:"sgt_dir"`      // .sgt directory path
}

// Orchestrator manages the top-level session and delegates to the registry.
type Orchestrator struct {
	cfg *Config
	t   *tmux.Tmux
	reg *Registry
}

// LoadOrchestrator reads .sgt/config.json from workDir.
func LoadOrchestrator(workDir string) (*Orchestrator, error) {
	sgtDir := filepath.Join(workDir, ".sgt")
	cfg, err := loadConfig(sgtDir)
	if err != nil {
		return nil, err
	}
	return &Orchestrator{
		cfg: cfg,
		t:   tmux.NewTmux(),
		reg: NewRegistry(sgtDir),
	}, nil
}

// Start initialises a new orchestrator session.
// If interactive is true, the session is created and the caller should attach.
// If interactive is false, the orchestrator runs in a background tmux session.
func Start(workDir string, sessionName string, interactive bool, orchCmd string) (*Orchestrator, error) {
	if sessionName == "" {
		sessionName = defaultSession
	}
	sgtDir := filepath.Join(workDir, ".sgt")
	if err := os.MkdirAll(sgtDir, 0o755); err != nil {
		return nil, fmt.Errorf("start: mkdir sgt dir: %w", err)
	}

	t := tmux.NewTmux()

	// Kill existing session if any, to get a fresh start.
	_ = t.KillSession(sessionName)

	if orchCmd != "" {
		if err := t.NewSessionWithCommand(sessionName, workDir, orchCmd); err != nil {
			return nil, fmt.Errorf("start: create session: %w", err)
		}
	} else {
		if err := t.NewSession(sessionName, workDir); err != nil {
			return nil, fmt.Errorf("start: create session: %w", err)
		}
	}

	// Label the first pane as "orchestrator"
	_, _ = runTmux("select-pane", "-t", sessionName+":0", "-T", "orchestrator")

	cfg := &Config{
		Session:     sessionName,
		Interactive: interactive,
		WorkDir:     workDir,
		SgtDir:      sgtDir,
	}
	if err := saveConfig(sgtDir, cfg); err != nil {
		return nil, err
	}

	return &Orchestrator{
		cfg: cfg,
		t:   t,
		reg: NewRegistry(sgtDir),
	}, nil
}

// Session returns the tmux session name.
func (o *Orchestrator) Session() string { return o.cfg.Session }

// SgtDir returns the .sgt directory.
func (o *Orchestrator) SgtDir() string { return o.cfg.SgtDir }

// Registry returns the agent registry.
func (o *Orchestrator) Registry() *Registry { return o.reg }

// Attach switches the current tmux client to the orchestrator session.
func (o *Orchestrator) Attach() error {
	return o.t.AttachSession(o.cfg.Session)
}

// Stop kills the orchestrator session and all managed agents.
func (o *Orchestrator) Stop() error {
	agents, _ := o.reg.List()
	for _, ag := range agents {
		_ = o.reg.Kill(ag.Name)
	}
	return o.t.KillSession(o.cfg.Session)
}

func loadConfig(sgtDir string) (*Config, error) {
	data, err := os.ReadFile(filepath.Join(sgtDir, "config.json"))
	if err != nil {
		return nil, fmt.Errorf("load config: %w (is 'sgt start' needed?)", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("load config: parse: %w", err)
	}
	return &cfg, nil
}

func saveConfig(sgtDir string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return os.WriteFile(filepath.Join(sgtDir, "config.json"), data, 0o644)
}
