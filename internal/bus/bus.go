// Package bus provides a simple JSONL-based event bus for inter-agent communication.
//
// The bus is an append-only log at .sgt/bus.jsonl. Any agent can publish events;
// any agent can watch the stream. Events are identified by source agent, type, and
// optional JSON data payload.
package bus

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Event is a single entry on the bus.
type Event struct {
	ID        string          `json:"id"`
	Timestamp time.Time       `json:"timestamp"`
	Source    string          `json:"source"`
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data,omitempty"`
}

// Bus manages the event log file.
type Bus struct {
	path string // path to bus.jsonl
}

// New creates a Bus rooted at dir (the .sgt directory).
func New(sgtDir string) *Bus {
	return &Bus{path: filepath.Join(sgtDir, "bus.jsonl")}
}

// Publish appends an event to the bus. source is the agent name, eventType is a
// dot-separated string like "agent.started" or "task.completed". data may be nil.
func (b *Bus) Publish(source, eventType string, data any) (*Event, error) {
	ev := &Event{
		ID:        generateID(),
		Timestamp: time.Now(),
		Source:    source,
		Type:      eventType,
	}

	if data != nil {
		raw, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("bus: marshal data: %w", err)
		}
		ev.Data = raw
	}

	line, err := json.Marshal(ev)
	if err != nil {
		return nil, fmt.Errorf("bus: marshal event: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(b.path), 0o755); err != nil {
		return nil, fmt.Errorf("bus: mkdir: %w", err)
	}

	f, err := os.OpenFile(b.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("bus: open: %w", err)
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "%s\n", line); err != nil {
		return nil, fmt.Errorf("bus: write: %w", err)
	}

	return ev, nil
}

// Tail reads events from offset (byte position) onward.
// Returns events and the new offset to pass on the next call.
// Pass offset=0 to read from the start.
func (b *Bus) Tail(offset int64) ([]Event, int64, error) {
	f, err := os.Open(b.path)
	if os.IsNotExist(err) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, offset, fmt.Errorf("bus: open: %w", err)
	}
	defer f.Close()

	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return nil, offset, fmt.Errorf("bus: seek: %w", err)
		}
	}

	var events []Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		var ev Event
		if err := json.Unmarshal(line, &ev); err != nil {
			continue // skip malformed lines
		}
		events = append(events, ev)
	}

	newOffset, _ := f.Seek(0, io.SeekCurrent)
	return events, newOffset, scanner.Err()
}

// Watch streams events to out, blocking until ctx is cancelled.
// Calls handler for each new event. Polls every pollInterval.
func (b *Bus) Watch(out io.Writer, pollInterval time.Duration, filter string) error {
	var offset int64
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")

	for {
		events, newOffset, err := b.Tail(offset)
		if err != nil {
			return err
		}
		offset = newOffset

		for _, ev := range events {
			if filter != "" && ev.Type != filter && ev.Source != filter {
				continue
			}
			_ = enc.Encode(ev)
		}

		time.Sleep(pollInterval)
	}
}

// ReadAll reads all events from the beginning.
func (b *Bus) ReadAll() ([]Event, error) {
	events, _, err := b.Tail(0)
	return events, err
}

// Path returns the path to the bus.jsonl file.
func (b *Bus) Path() string {
	return b.path
}

func generateID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("ev-%x", time.Now().UnixNano())
	}
	return "ev-" + hex.EncodeToString(b)
}
