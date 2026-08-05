package ir

import "math"

// ReadinessScore estimates how safe an automated migration is (0-100).
//
// Scoring philosophy:
//   - Start at 100
//   - Each L1 finding is free
//   - Each L2 finding costs a little (policy CRDs must exist)
//   - Each L3 finding costs a lot (human rewrite required)
//   - Partial snippet promotions cost medium
// Every migration annotation should produce a finding; nothing is omitted quietly.
func (b *MigrationBundle) ReadinessScore() int {
	if b == nil || len(b.Findings) == 0 {
		return 100
	}
	score := 100.0
	for _, f := range b.Findings {
		switch f.Status {
		case StatusDirect:
			// no penalty
		case StatusRequiresPolicy:
			score -= 6
			if f.Level == 2 && containsFold(f.Message, "Partial snippet") {
				score -= 8
			}
		case StatusUntranslatable:
			score -= 25
		}
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return int(math.Round(score))
}

// ReadinessLabel is a human bucket for the score.
func (b *MigrationBundle) ReadinessLabel() string {
	s := b.ReadinessScore()
	switch {
	case s >= 85:
		return "READY"
	case s >= 60:
		return "READY_WITH_POLICIES"
	case s >= 35:
		return "NEEDS_REVIEW"
	default:
		return "BLOCKED"
	}
}

func containsFold(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (s == sub || indexFold(s, sub) >= 0))
}

func indexFold(s, sub string) int {
	ls, lsub := len(s), len(sub)
	for i := 0; i+lsub <= ls; i++ {
		ok := true
		for j := 0; j < lsub; j++ {
			a, b := s[i+j], sub[j]
			if a >= 'A' && a <= 'Z' {
				a += 32
			}
			if b >= 'A' && b <= 'Z' {
				b += 32
			}
			if a != b {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}
