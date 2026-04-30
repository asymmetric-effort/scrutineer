package fleet

import mrand "math/rand/v2"

// SelectByWeight picks an index from a slice of weights using weighted
// random selection. Weights must sum to a positive value.
func SelectByWeight(weights []int, rng *mrand.Rand) int {
	total := 0
	for _, w := range weights {
		total += w
	}
	if total == 0 {
		return 0
	}
	r := rng.IntN(total)
	cumulative := 0
	for i, w := range weights {
		cumulative += w
		if r < cumulative {
			return i
		}
	}
	return len(weights) - 1
}

// DistributeByWeight distributes total items across bins proportionally
// to their weights. Remainders are distributed round-robin to the
// highest-weight bins.
func DistributeByWeight(weights []int, total int) []int {
	result := make([]int, len(weights))
	if len(weights) == 0 || total == 0 {
		return result
	}

	weightSum := 0
	for _, w := range weights {
		weightSum += w
	}
	if weightSum == 0 {
		return result
	}

	assigned := 0
	for i, w := range weights {
		result[i] = total * w / weightSum
		assigned += result[i]
	}

	// Distribute remainders.
	remainder := total - assigned
	for i := 0; remainder > 0; i++ {
		idx := i % len(weights)
		if weights[idx] > 0 {
			result[idx]++
			remainder--
		}
	}

	return result
}
