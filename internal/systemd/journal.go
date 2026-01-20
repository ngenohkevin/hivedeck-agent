package systemd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

// JournalReader reads systemd journal logs
type JournalReader struct{}

// NewJournalReader creates a new journal reader
func NewJournalReader() *JournalReader {
	return &JournalReader{}
}

// Query reads journal entries based on the query parameters
func (r *JournalReader) Query(ctx context.Context, query JournalQuery) (*LogStream, error) {
	args := []string{"--output=json", "--no-pager"}

	if query.Unit != "" {
		args = append(args, "-u", query.Unit)
	}

	if query.Priority >= 0 && query.Priority <= 7 {
		args = append(args, "-p", strconv.Itoa(query.Priority))
	}

	lines := query.Lines
	if lines <= 0 {
		lines = 100
	}
	args = append(args, "-n", strconv.Itoa(lines))

	if query.Since != "" {
		args = append(args, "--since", query.Since)
	}

	if query.Until != "" {
		args = append(args, "--until", query.Until)
	}

	cmd := exec.CommandContext(ctx, "journalctl", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read journal: %w", err)
	}

	entries, err := r.parseJSONOutput(output)
	if err != nil {
		return nil, err
	}

	return &LogStream{
		Entries: entries,
		Unit:    query.Unit,
	}, nil
}

// Follow streams journal entries in real-time
func (r *JournalReader) Follow(ctx context.Context, unit string, entryChan chan<- JournalEntry) error {
	args := []string{"--output=json", "--no-pager", "-f"}

	if unit != "" {
		args = append(args, "-u", unit)
	}

	cmd := exec.CommandContext(ctx, "journalctl", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start journalctl: %w", err)
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			entry, err := r.parseJSONLine(scanner.Bytes())
			if err != nil {
				continue
			}
			select {
			case entryChan <- *entry:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		<-ctx.Done()
		cmd.Process.Kill()
	}()

	return nil
}

// GetRecentLogs returns recent log entries for a unit
func (r *JournalReader) GetRecentLogs(ctx context.Context, unit string, lines int) ([]JournalEntry, error) {
	stream, err := r.Query(ctx, JournalQuery{
		Unit:  unit,
		Lines: lines,
	})
	if err != nil {
		return nil, err
	}
	return stream.Entries, nil
}

func (r *JournalReader) parseJSONOutput(output []byte) ([]JournalEntry, error) {
	var entries []JournalEntry

	// Each line is a separate JSON object
	scanner := bufio.NewScanner(
		&byteReader{data: output},
	)

	for scanner.Scan() {
		entry, err := r.parseJSONLine(scanner.Bytes())
		if err != nil {
			continue
		}
		entries = append(entries, *entry)
	}

	return entries, nil
}

func (r *JournalReader) parseJSONLine(line []byte) (*JournalEntry, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil, err
	}

	entry := &JournalEntry{}

	// Parse timestamp (microseconds since epoch)
	if ts, ok := raw["__REALTIME_TIMESTAMP"].(string); ok {
		if usec, err := strconv.ParseInt(ts, 10, 64); err == nil {
			entry.Timestamp = time.UnixMicro(usec)
		}
	}

	if unit, ok := raw["_SYSTEMD_UNIT"].(string); ok {
		entry.Unit = unit
	}

	if msg, ok := raw["MESSAGE"].(string); ok {
		entry.Message = msg
	}

	if prio, ok := raw["PRIORITY"].(string); ok {
		if p, err := strconv.Atoi(prio); err == nil {
			entry.Priority = p
		}
	}

	if pid, ok := raw["_PID"].(string); ok {
		entry.PID = pid
	}

	if hostname, ok := raw["_HOSTNAME"].(string); ok {
		entry.Hostname = hostname
	}

	return entry, nil
}

// byteReader implements io.Reader for a byte slice
type byteReader struct {
	data []byte
	pos  int
}

func (r *byteReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, fmt.Errorf("EOF")
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
