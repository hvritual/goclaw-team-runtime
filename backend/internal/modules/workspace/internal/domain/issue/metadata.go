package issue

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
)

const (
	MaxMetadataKeys = 50
	MaxMetadataSize = 8 * 1024
)

var metadataKeyPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_.-]{0,63}$`)

type MetadataBag struct {
	values map[string]any
}

func ValidateMetadataKey(key string) error {
	if !metadataKeyPattern.MatchString(key) {
		return errors.New("metadata key must start with a letter or underscore and contain only letters, numbers, underscores, dots, or hyphens (max 64 characters)")
	}
	return nil
}

func ParseMetadataValueJSON(raw string) (any, error) {
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, errors.New("metadata value must be a string, number, or boolean")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, errors.New("metadata value must be a string, number, or boolean")
	}
	if !validMetadataValue(value) {
		return nil, errors.New("metadata value must be a string, number, or boolean")
	}
	return value, nil
}

func NewMetadataBag(values map[string]any) MetadataBag {
	return MetadataBag{values: cloneMetadata(values)}
}

func (b MetadataBag) Snapshot() map[string]any {
	return cloneMetadata(b.values)
}

func (b *MetadataBag) Put(key string, value any) (map[string]any, error) {
	if err := ValidateMetadataKey(key); err != nil {
		return nil, err
	}
	if !validMetadataValue(value) {
		return nil, errors.New("metadata value must be a string, number, or boolean")
	}
	if _, exists := b.values[key]; !exists && len(b.values) >= MaxMetadataKeys {
		return nil, errors.New("metadata cannot exceed 50 keys")
	}
	candidate := cloneMetadata(b.values)
	candidate[key] = value
	if err := validateMetadataSize(candidate); err != nil {
		return nil, err
	}
	b.values = candidate
	return b.Snapshot(), nil
}

func (b *MetadataBag) Delete(key string) (map[string]any, error) {
	if err := ValidateMetadataKey(key); err != nil {
		return nil, err
	}
	candidate := cloneMetadata(b.values)
	delete(candidate, key)
	b.values = candidate
	return b.Snapshot(), nil
}

func validateMetadataSize(values map[string]any) error {
	encoded, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("encode metadata: %w", err)
	}
	if len(encoded) > MaxMetadataSize {
		return errors.New("metadata exceeds the 8KB size limit")
	}
	return nil
}

func validMetadataValue(value any) bool {
	switch value.(type) {
	case string, bool, json.Number, float64, float32,
		int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}

func cloneMetadata(values map[string]any) map[string]any {
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
