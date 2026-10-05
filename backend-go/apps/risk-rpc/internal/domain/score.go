package domain

func Level(score float64) string {
	switch {
	case score > 80:
		return RiskLevelExtreme
	case score > 60:
		return RiskLevelHigh
	case score > 30:
		return RiskLevelMedium
	default:
		return RiskLevelLow
	}
}

func Overall(items []scoredItem) (float64, string) {
	maxScore := 0.0
	for _, item := range items {
		if item.score > maxScore {
			maxScore = item.score
		}
	}
	return maxScore, Level(maxScore)
}

func clamp(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

type scoredItem struct {
	score float64
}
