package spellChecker

import "testing"

func TestReplacement(t *testing.T) {
	sc := NewSpellChecker(3, 3)
	testQuery := []string{"test", "qury", "text"}
	testReplacements := []string{"query", "qures", "quit"}
	testScores := [][2]float64{{0.99, 0.4}, {0.25, 0.25}, {0.9, 0}}
	sc.BestReplacement(&testQuery, 1, testReplacements, testScores)
	if testQuery[1] != testReplacements[0] {
		t.Errorf("invalid replacement value %s, instead if %s", testQuery[1], testReplacements[0])
	}
}

func TestSplittingReplacement(t *testing.T) {
	sc := NewSpellChecker(3, 3)
	testQuery := []string{"test", "querytext", "replacement"}
	testReplacements := []string{"query", "qures", "quit", "text"}
	testScores := [][2]float64{{0.99, 0.4}, {0.1, 0.2}, {0.9, 0}, {0.55, 0.6}}
	baseLen := len(testQuery)
	sc.BestReplacement(&testQuery, 1, testReplacements, testScores)
	if baseLen == len(testQuery) {
		t.Errorf("word: %s, wasn't replaced", testQuery[1])
		return
	}
	if testQuery[1] != testReplacements[0] && testQuery[2] != testReplacements[3] {
		t.Errorf("invalid replacement value %s : %s, instead if %s : %s", testQuery[1], testQuery[2], testReplacements[0], testReplacements[3])
	}
}

func TestLevenshteinDistance(t *testing.T) {
	w1Test := "testWord"
	w2Test := "WordWithBigDistance"
	if dist := levenshteinDistance([]rune(w1Test), []rune(w2Test), 3); dist != 4 {
		t.Errorf("invalid levenshtein distance %d, instead of %d, with words: %s : %s", dist, 4, w1Test, w2Test)
		return
	}

	w1Test = "первое"
	w2Test = "первое"
	if dist := levenshteinDistance([]rune(w1Test), []rune(w2Test), 3); dist != 0 {
		t.Errorf("invalid levenshtein distance %d, instead of %d, with words: %s : %s", dist, 4, w1Test, w2Test)
	}
}