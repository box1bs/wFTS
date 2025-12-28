package spellChecker

import "math"

const ( // константы зашумленного канала
    a = 2
    b = 3
    c = 2
)

type SpellChecker struct {
	maxTypo     int
    NGramCount  int
}

func NewSpellChecker(maxTypoLen, ngc int) *SpellChecker {
	return &SpellChecker{
		maxTypo: maxTypoLen,
        NGramCount: ngc,
	}
}

func (s *SpellChecker) BestReplacement(s1 string, candidates []string, scores [][2]float64) string {
    if len(candidates) == 0 {
        return s1
    }
    best := candidates[0]
    bscore := -math.MaxFloat32
    orig := []rune(s1)
	for i, candidate := range candidates {
		distance := s.levenshteinDistance(orig, []rune(candidate))
		if score := math.Pow(c, -float64(distance)) * max((a * scores[i][0] + b * scores[i][1]), 0.00001); score > bscore && distance <= s.maxTypo { // c ** -dist * max((a * log(P(c | left) + 1) + b * log(P(right | c) + 1)), const) c, a и b веса контекста, const минимальная вероятность для сочетаний которых нет в базе, чтобы они не отсекались полностью
            best = candidate
            bscore = score
        }
	}
    return best
}

func (s *SpellChecker) levenshteinDistance(word1 []rune, word2 []rune) int {
    l1, l2 := len(word1), len(word2)
    if l1 == 0 {
        return l2
    }
    if l2 == 0 {
        return l1
    }
	dp := make([]int, l2 + 1)
	for i := 0; i < l2 + 1; i++ {
		dp[i] = i
	}

	for i := range l1 {
		dp[0] = i
		upLeft := dp[0]
		minRow := i
		for j := 1; j <= l2; j++ {
			nextUpLeft := dp[j]
			if word1[i] == word2[j-1] {
				dp[j] = upLeft
			} else {
				dp[j] = min(dp[j - 1], upLeft, dp[j]) + 1
			}
			upLeft = nextUpLeft
			minRow = min(minRow, dp[j])
		}
        if minRow > s.maxTypo {
            return s.maxTypo + 1
        }
	}
	return dp[l2]
}