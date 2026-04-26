package fuzz

import (
	"math"
	"math/rand"
)

// Generator produces fuzzed inputs based on a target's base parameters.
type Generator struct {
	rng    *rand.Rand
	target *Target
}

// NewGenerator creates a Generator seeded with the given value.
func NewGenerator(target *Target, seed int64) *Generator {
	return &Generator{
		rng:    rand.New(rand.NewSource(seed)),
		target: target,
	}
}

// Next produces the next fuzzed input. It copies the target's base parameters
// and mutates only the fields listed in target.FuzzFields.
func (g *Generator) Next() map[string]any {
	result := copyMap(g.target.Parameters)
	for _, field := range g.target.FuzzFields {
		result[field] = g.mutateValue(result[field])
	}
	return result
}

// mutateValue produces a mutated version of the given value based on its type.
func (g *Generator) mutateValue(v any) any {
	switch val := v.(type) {
	case string:
		return g.mutateString(val)
	case int:
		return g.mutateInt(val)
	case int64:
		return int64(g.mutateInt(int(val)))
	case float64:
		return g.mutateFloat(val)
	case bool:
		return !val
	case nil:
		// Generate a random type.
		switch g.rng.Intn(3) {
		case 0:
			return g.randomString(g.rng.Intn(20))
		case 1:
			return g.rng.Intn(1000)
		default:
			return g.rng.Float64() < 0.5
		}
	default:
		// For unknown types, return as-is.
		return v
	}
}

// mutateString applies a random mutation to a string.
func (g *Generator) mutateString(s string) string {
	if len(s) == 0 {
		return g.randomString(g.rng.Intn(10) + 1)
	}

	switch g.rng.Intn(5) {
	case 0: // random replacement chars
		bs := []byte(s)
		idx := g.rng.Intn(len(bs))
		bs[idx] = byte(g.rng.Intn(95) + 32) // printable ASCII
		return string(bs)
	case 1: // insert a random char
		bs := []byte(s)
		idx := g.rng.Intn(len(bs) + 1)
		ch := byte(g.rng.Intn(95) + 32)
		result := make([]byte, len(bs)+1)
		copy(result, bs[:idx])
		result[idx] = ch
		copy(result[idx+1:], bs[idx:])
		return string(result)
	case 2: // delete a char
		if len(s) <= 1 {
			return ""
		}
		bs := []byte(s)
		idx := g.rng.Intn(len(bs))
		return string(append(bs[:idx], bs[idx+1:]...))
	case 3: // bit flip
		bs := []byte(s)
		idx := g.rng.Intn(len(bs))
		bit := byte(1 << uint(g.rng.Intn(8)))
		bs[idx] ^= bit
		return string(bs)
	default: // completely random string
		return g.randomString(g.rng.Intn(50) + 1)
	}
}

// mutateInt applies a random mutation to an integer.
func (g *Generator) mutateInt(n int) int {
	switch g.rng.Intn(5) {
	case 0: // boundary: 0
		return 0
	case 1: // boundary: max
		return math.MaxInt32
	case 2: // boundary: min
		return math.MinInt32
	case 3: // small delta
		delta := g.rng.Intn(21) - 10 // -10 to +10
		return n + delta
	default: // random
		return g.rng.Int()
	}
}

// mutateFloat applies a random mutation to a float.
func (g *Generator) mutateFloat(f float64) float64 {
	switch g.rng.Intn(5) {
	case 0:
		return 0.0
	case 1:
		return math.MaxFloat64
	case 2:
		return -math.MaxFloat64
	case 3:
		return f + (g.rng.Float64()*20 - 10)
	default:
		return g.rng.Float64() * 1000
	}
}

// randomString generates a random printable ASCII string of the given length.
func (g *Generator) randomString(length int) string {
	bs := make([]byte, length)
	for i := range bs {
		bs[i] = byte(g.rng.Intn(95) + 32) // printable ASCII
	}
	return string(bs)
}
