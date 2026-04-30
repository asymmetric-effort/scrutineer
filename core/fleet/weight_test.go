package fleet

import (
	mrand "math/rand/v2"
	"testing"
)

func TestSelectByWeight_Distribution(t *testing.T) {
	rng := mrand.New(mrand.NewPCG(42, 0))
	weights := []int{80, 20}
	counts := make(map[int]int)
	for i := 0; i < 10000; i++ {
		idx := SelectByWeight(weights, rng)
		counts[idx]++
	}
	ratio := float64(counts[0]) / 10000.0
	if ratio < 0.7 || ratio > 0.9 {
		t.Errorf("item 0 selected %.1f%% (expected ~80%%)", ratio*100)
	}
}

func TestSelectByWeight_SingleItem(t *testing.T) {
	rng := mrand.New(mrand.NewPCG(42, 0))
	idx := SelectByWeight([]int{100}, rng)
	if idx != 0 {
		t.Errorf("got %d, want 0", idx)
	}
}

func TestSelectByWeight_ZeroTotal(t *testing.T) {
	rng := mrand.New(mrand.NewPCG(42, 0))
	idx := SelectByWeight([]int{0, 0}, rng)
	if idx != 0 {
		t.Errorf("got %d, want 0", idx)
	}
}

func TestDistributeByWeight_Even(t *testing.T) {
	result := DistributeByWeight([]int{50, 50}, 100)
	if result[0] != 50 || result[1] != 50 {
		t.Errorf("got %v, want [50, 50]", result)
	}
}

func TestDistributeByWeight_Uneven(t *testing.T) {
	result := DistributeByWeight([]int{70, 30}, 10)
	if result[0] != 7 || result[1] != 3 {
		t.Errorf("got %v, want [7, 3]", result)
	}
}

func TestDistributeByWeight_WithRemainder(t *testing.T) {
	result := DistributeByWeight([]int{60, 40}, 11)
	total := result[0] + result[1]
	if total != 11 {
		t.Errorf("total = %d, want 11", total)
	}
}

func TestDistributeByWeight_Zero(t *testing.T) {
	result := DistributeByWeight([]int{50, 50}, 0)
	if result[0] != 0 || result[1] != 0 {
		t.Errorf("got %v", result)
	}
}

func TestDistributeByWeight_Empty(t *testing.T) {
	result := DistributeByWeight(nil, 10)
	if len(result) != 0 {
		t.Errorf("got %v", result)
	}
}

func TestDistributeByWeight_ZeroWeights(t *testing.T) {
	result := DistributeByWeight([]int{0, 0}, 10)
	if result[0] != 0 || result[1] != 0 {
		t.Errorf("got %v", result)
	}
}
