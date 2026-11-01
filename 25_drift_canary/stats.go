package main

import (
	"math"
	"sort"
)

// logChoose returns ln(n choose k).
func logChoose(n, k int) float64 {
	a, _ := math.Lgamma(float64(n + 1))
	b, _ := math.Lgamma(float64(k + 1))
	c, _ := math.Lgamma(float64(n - k + 1))
	return a - b - c
}

// FisherExact returns the two-sided p-value of Fisher's exact test for the
// 2×2 table [[a, b], [c, d]] (rows: baseline/candidate, cols: pass/fail).
func FisherExact(a, b, c, d int) float64 {
	n := a + b + c + d
	if n == 0 {
		return 1
	}
	row1, col1 := a+b, a+c
	lo := max(0, col1-(n-row1))
	hi := min(row1, col1)
	logP := func(x int) float64 {
		return logChoose(row1, x) + logChoose(n-row1, col1-x) - logChoose(n, col1)
	}
	pObs := logP(a)
	p := 0.0
	for x := lo; x <= hi; x++ {
		lp := logP(x)
		if lp <= pObs+1e-7 {
			p += math.Exp(lp)
		}
	}
	return math.Min(1, p)
}

// McNemarExact returns the two-sided exact p-value for paired outcomes, where
// b = pass→fail pairs and c = fail→pass pairs. Use it when both runs answered
// the same fixed tasks: it is far more sensitive than an unpaired test.
func McNemarExact(b, c int) float64 {
	n := b + c
	if n == 0 {
		return 1
	}
	k := min(b, c)
	p := 0.0
	for x := 0; x <= k; x++ {
		p += math.Exp(logChoose(n, x) - float64(n)*math.Ln2)
	}
	return math.Min(1, 2*p)
}

// MannWhitney returns the two-sided p-value (normal approximation with tie
// correction) for the hypothesis that xs and ys come from the same distribution.
func MannWhitney(xs, ys []float64) float64 {
	n1, n2 := len(xs), len(ys)
	if n1 == 0 || n2 == 0 {
		return 1
	}
	type obs struct {
		v float64
		g int
	}
	all := make([]obs, 0, n1+n2)
	for _, x := range xs {
		all = append(all, obs{x, 0})
	}
	for _, y := range ys {
		all = append(all, obs{y, 1})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].v < all[j].v })
	n := float64(n1 + n2)
	r1, tieSum := 0.0, 0.0
	for i := 0; i < len(all); {
		j := i
		for j < len(all) && all[j].v == all[i].v {
			j++
		}
		rank := float64(i+j+1) / 2 // average of ranks i+1 … j
		t := float64(j - i)
		tieSum += t*t*t - t
		for k := i; k < j; k++ {
			if all[k].g == 0 {
				r1 += rank
			}
		}
		i = j
	}
	u := r1 - float64(n1*(n1+1))/2
	mu := float64(n1*n2) / 2
	sigma := math.Sqrt(float64(n1*n2) / 12 * ((n + 1) - tieSum/(n*(n-1))))
	if sigma == 0 {
		return 1
	}
	z := (math.Abs(u-mu) - 0.5) / sigma
	if z < 0 {
		z = 0
	}
	return math.Erfc(z / math.Sqrt2)
}

// median returns the median of xs (0 for empty input).
func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	m := len(s) / 2
	if len(s)%2 == 1 {
		return s[m]
	}
	return (s[m-1] + s[m]) / 2
}
