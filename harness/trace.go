package harness

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var traceWriteMu sync.Mutex

func (s *Service) RecordTrace(trace Trace) error {
	if !s.cfg.Enabled || !s.cfg.TraceEnabled {
		return nil
	}
	if trace.ID == "" {
		trace.ID = "trace-" + uuid.NewString()
	}
	trace.SchemaVersion = SchemaVersion
	if trace.HarnessVersion == "" {
		active, err := s.ActiveState()
		if err == nil {
			trace.HarnessVersion = active.Version
		}
	}
	if trace.ProjectID == "" {
		trace.ProjectID = s.cfg.ProjectID
	}
	if trace.StartedAt.IsZero() {
		trace.StartedAt = time.Now().UTC()
	}
	if trace.FinishedAt.IsZero() {
		trace.FinishedAt = time.Now().UTC()
	}
	if trace.DurationMS == 0 {
		trace.DurationMS = trace.FinishedAt.Sub(trace.StartedAt).Milliseconds()
	}
	date := trace.StartedAt.UTC().Format("2006-01-02")
	path := filepath.Join(s.tracesDir(), date+".jsonl")
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	data, err := json.Marshal(trace)
	if err != nil {
		return err
	}
	traceWriteMu.Lock()
	defer traceWriteMu.Unlock()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return file.Sync()
}

func (s *Service) ListTraces(projectID string, limit int) ([]Trace, error) {
	if limit <= 0 {
		limit = 100
	}
	entries, err := os.ReadDir(s.tracesDir())
	if err != nil {
		return nil, err
	}
	names := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jsonl") {
			names = append(names, entry.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	result := make([]Trace, 0, limit)
	for _, name := range names {
		file, err := os.Open(filepath.Join(s.tracesDir(), name))
		if err != nil {
			return nil, err
		}
		var day []Trace
		scanner := bufio.NewScanner(file)
		buffer := make([]byte, 64*1024)
		scanner.Buffer(buffer, 8*1024*1024)
		for scanner.Scan() {
			var trace Trace
			if err := json.Unmarshal(scanner.Bytes(), &trace); err != nil {
				_ = file.Close()
				return nil, fmt.Errorf("parse trace %s: %w", name, err)
			}
			if projectID == "" || trace.ProjectID == projectID {
				day = append(day, trace)
			}
		}
		if err := scanner.Err(); err != nil {
			_ = file.Close()
			return nil, err
		}
		_ = file.Close()
		for i := len(day) - 1; i >= 0 && len(result) < limit; i-- {
			result = append(result, day[i])
		}
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (s *Service) AddHumanFeedback(traceID string, feedback HumanFeedback) error {
	if traceID == "" {
		return fmt.Errorf("trace id is required")
	}
	if feedback.CreatedAt.IsZero() {
		feedback.CreatedAt = time.Now().UTC()
	}
	path := filepath.Join(s.tracesDir(), "feedback.jsonl")
	record := map[string]any{
		"schema_version": SchemaVersion,
		"trace_id":       traceID,
		"feedback":       feedback,
	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	traceWriteMu.Lock()
	defer traceWriteMu.Unlock()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return file.Sync()
}
