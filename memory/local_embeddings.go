package memory

import (
	"fmt"
	"hash/fnv"
	"math"
	"strings"
	"unicode"
)

// LocalHashProvider is a deterministic, offline feature-hashing embedder. It
// is intentionally modest: it keeps the builtin backend functional without an
// API key, while the governed catalog provides metadata, lifecycle, and
// provenance-aware retrieval.
type LocalHashProvider struct {
	dimension int
}

func NewLocalHashProvider() *LocalHashProvider {
	return &LocalHashProvider{dimension: 384}
}

func (p *LocalHashProvider) Embed(text string) ([]float32, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("text must not be empty")
	}
	vector := make([]float32, p.dimension)
	tokens := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_')
	})
	if len(tokens) == 0 {
		tokens = []string{strings.ToLower(strings.TrimSpace(text))}
	}
	for _, token := range tokens {
		addHashedFeature(vector, "w:"+token, 1)
		runes := []rune(token)
		for index := 0; index < len(runes); index++ {
			addHashedFeature(vector, "r:"+string(runes[index]), 0.35)
			if index+1 < len(runes) {
				addHashedFeature(vector, "b:"+string(runes[index:index+2]), 0.55)
			}
			if index+2 < len(runes) {
				addHashedFeature(vector, "t:"+string(runes[index:index+3]), 0.25)
			}
		}
	}
	var norm float64
	for _, value := range vector {
		norm += float64(value * value)
	}
	if norm == 0 {
		return nil, fmt.Errorf("could not derive local embedding")
	}
	norm = math.Sqrt(norm)
	for index := range vector {
		vector[index] = float32(float64(vector[index]) / norm)
	}
	return vector, nil
}

func (p *LocalHashProvider) EmbedBatch(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided")
	}
	if len(texts) > p.MaxBatchSize() {
		return nil, fmt.Errorf("batch size %d exceeds maximum %d", len(texts), p.MaxBatchSize())
	}
	result := make([][]float32, len(texts))
	for index, text := range texts {
		vector, err := p.Embed(text)
		if err != nil {
			return nil, fmt.Errorf("embed item %d: %w", index, err)
		}
		result[index] = vector
	}
	return result, nil
}

func (p *LocalHashProvider) Dimension() int {
	return p.dimension
}

func (p *LocalHashProvider) MaxBatchSize() int {
	return 4096
}

func addHashedFeature(vector []float32, feature string, weight float32) {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(feature))
	value := hash.Sum64()
	index := int(value % uint64(len(vector)))
	if value&(1<<63) != 0 {
		weight = -weight
	}
	vector[index] += weight
}
