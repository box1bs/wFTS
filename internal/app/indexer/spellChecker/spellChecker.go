package spellChecker

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
    bscore := -1e18 // без импорта math
    orig := []rune(s1)
	for i, candidate := range candidates {
		distance := s.levenshteinDistance(orig, []rune(candidate))
		if score := -1 * float64(distance) + 2 * scores[i][0] + 3 * scores[i][1]; score > bscore && distance <= s.maxTypo { // -dist + log(P(c | left) + 1) + log(P(right | c) + 1)
            best = candidate
            bscore = score
        }
	}
    return best
}

func (s *SpellChecker) levenshteinDistance(word1 []rune, word2 []rune) int {
    w1, w2 := len(word1), len(word2)
    if w1 - w2 > s.maxTypo || w2 - w1 > s.maxTypo {
        return s.maxTypo + 1
    }
    dp := make([][]int, w1 + 1)
    for i := range w1 + 1 {
        dp[i] = make([]int, w2 + 1)
    }

    for i := 1; i <= w1; i++ {
        dp[i][0] = i
    }
    for j := 1; j <= w2; j++ {
        dp[0][j] = j
    }

    for i := 1; i <= w1; i++ {
        for j := 1; j <= w2; j++ {
            if word1[i - 1] == word2[j - 1] {
                dp[i][j] = dp[i - 1][j - 1]
            } else {
                insert := dp[i - 1][j]
                delete := dp[i][j - 1]
                replace := dp[i - 1][j - 1]
                dp[i][j] = min(delete, insert, replace) + 1
            }
        }
    }

    return dp[w1][w2]
}