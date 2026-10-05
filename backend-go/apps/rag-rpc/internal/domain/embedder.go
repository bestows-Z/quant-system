package domain

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
	"strings"
	"unicode"
)

const (
	localEmbeddingProvider = "local"
	localEmbeddingModel    = "feature-hashing-zh-v1"
	localEmbeddingDim      = 384
)

type Embedder interface {
	Provider() string
	Model() string
	Dimension() int
	Embed(text string) []float64
}

type LocalEmbedder struct{}

func (LocalEmbedder) Provider() string { return localEmbeddingProvider }
func (LocalEmbedder) Model() string    { return localEmbeddingModel }
func (LocalEmbedder) Dimension() int   { return localEmbeddingDim }

func (LocalEmbedder) Embed(text string) []float64 {
	vec := make([]float64, localEmbeddingDim)
	normalized := strings.ToLower(text)
	for _, token := range tokenize(normalized) {
		hash := sha256.Sum256([]byte(token))
		idx := int(binary.BigEndian.Uint16(hash[:2]) % localEmbeddingDim)
		sign := 1.0
		if hash[2]%2 == 0 {
			sign = -1
		}
		weight := 1.0 + float64(len([]rune(token))%7)/10.0
		vec[idx] += sign * weight
	}
	normalize(vec)
	return vec
}

func tokenize(text string) []string {
	var tokens []string
	var current []rune
	flush := func() {
		if len(current) > 0 {
			tokens = append(tokens, string(current))
			current = nil
		}
	}
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r >= 0x4e00 && r <= 0x9fff {
			current = append(current, r)
			if r >= 0x4e00 && r <= 0x9fff {
				flush()
			}
			continue
		}
		flush()
	}
	flush()
	return tokens
}

func normalize(vec []float64) {
	var sum float64
	for _, v := range vec {
		sum += v * v
	}
	if sum == 0 {
		return
	}
	norm := math.Sqrt(sum)
	for i := range vec {
		vec[i] = vec[i] / norm
	}
}

func cosine(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, aa, bb float64
	for i := range a {
		dot += a[i] * b[i]
		aa += a[i] * a[i]
		bb += b[i] * b[i]
	}
	if aa == 0 || bb == 0 {
		return 0
	}
	return dot / (math.Sqrt(aa) * math.Sqrt(bb))
}
