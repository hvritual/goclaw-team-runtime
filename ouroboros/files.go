package ouroboros

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var safeIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$`)

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func validateID(id string) error {
	if !safeIDPattern.MatchString(id) {
		return fmt.Errorf("invalid id %q", id)
	}
	return nil
}

func safeJoin(root, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("absolute path is not allowed: %s", rel)
	}
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	target, err := filepath.Abs(filepath.Join(cleanRoot, filepath.Clean(rel)))
	if err != nil {
		return "", err
	}
	if target != cleanRoot && !strings.HasPrefix(target, cleanRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes root: %s", rel)
	}
	return target, nil
}

func writeJSONAtomic(path string, value any, mode os.FileMode) error {
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}

func appendJSONLine(path string, value any) error {
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
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

func sha256Bytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func readEventLines(path string) ([]SessionEvent, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var events []SessionEvent
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 32*1024*1024)
	for scanner.Scan() {
		var event SessionEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, fmt.Errorf("decode event line %d: %w", len(events)+1, err)
		}
		events = append(events, event)
	}
	return events, scanner.Err()
}

func validateEventChain(events []SessionEvent) error {
	var previousHash string
	for index, event := range events {
		if event.Sequence != int64(index+1) {
			return fmt.Errorf("event sequence mismatch at %d", index+1)
		}
		if event.PreviousHash != previousHash {
			return fmt.Errorf("event hash chain mismatch at sequence %d", event.Sequence)
		}
		hashable := event
		hashable.Hash = ""
		encoded, err := json.Marshal(hashable)
		if err != nil {
			return err
		}
		if sha256Bytes(encoded) != event.Hash {
			return fmt.Errorf("event content hash mismatch at sequence %d", event.Sequence)
		}
		previousHash = event.Hash
	}
	return nil
}

func immutableWriteJSON(path string, value any) error {
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			var existing any
			if readErr := readJSON(path, &existing); readErr != nil {
				return fmt.Errorf("immutable artifact exists but cannot be read: %w", readErr)
			}
			var proposed any
			if unmarshalErr := json.Unmarshal(data, &proposed); unmarshalErr != nil {
				return unmarshalErr
			}
			existingJSON, _ := json.Marshal(existing)
			proposedJSON, _ := json.Marshal(proposed)
			if string(existingJSON) == string(proposedJSON) {
				return nil
			}
			return fmt.Errorf("immutable artifact already exists with different content: %s", path)
		}
		return err
	}
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return err
	}
	return file.Sync()
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
