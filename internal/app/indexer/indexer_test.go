package indexer

import (
	"math/rand"
	"testing"
)

func TestWordHandlingFunction(t *testing.T) {
	textTokens := []string{"this", "is", "a", "simple", "example", "for", "ngram", "testing"}
	expectedResult := []string{
		"this is a simple",
        "is a simple example",
        "a simple example for",
        "simple example for ngram",
        "example for ngram testing",
	}
	tokens := getWordNGrams(textTokens)
	if l := len(tokens); l != len(expectedResult) {
		t.Errorf("invalid tokens len: %d", l)
		return
	}
	for i, token := range tokens {
		if token != expectedResult[i] {
			t.Errorf("unexpected token: %s, instead of: %s", token, expectedResult[i])
			return
		}
	}
}

func TestHashSimilarity(t *testing.T) {
	t1 := [128]uint64{}
	t2 := [128]uint64{}
	t3 := [128]uint64{}
	for i := range 128 {
		randomed := rand.Uint64()
		t1[i], t2[i], t3[i] = randomed, randomed, randomed
	}
	if sim := calcSim(t1, [][128]uint64{t2}); sim != 1 {
		t.Errorf("unexpected sim %f, instead of 1.0", sim)
		return
	}
	t2[127] -= 1
	t3[1] -= 6
	t3[0] -= 2
	if sim := calcSim(t1, [][128]uint64{t2, t3}); sim != float64(127) / float64(128) {
		t.Errorf("unexpected sim %f, instead if 127 / 128", sim)
	}
}