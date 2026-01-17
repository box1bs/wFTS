package indexer

import (
	"math/rand"
	"testing"
)

func TestWordHandlingFunction(t *testing.T) {
	tests := []struct {
        name     string
        input    []string
        expected []string
    }{
        {
            name: "ngram example",
            input: []string{"this", "is", "a", "simple", "example", "for", "ngram", "testing"},
            expected: []string{
                "this is a simple",
                "is a simple example", 
                "a simple example for",
                "simple example for ngram",
                "example for ngram testing",
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tokens := getWordNGrams(tt.input)
            if l := len(tokens); l != len(tt.expected) {
                t.Errorf("invalid tokens len: %d", l)
                return
            }
            for i, token := range tokens {
                if token != tt.expected[i] {
                    t.Errorf("unexpected token: %s, instead of: %s", token, tt.expected[i])
                    return
                }
            }
        })
    }
}

func TestHashSimilarity(t *testing.T) {
	testArray := [128]uint64{}
	for i := range 128 {
		testArray[i] = rand.Uint64()
	}
	tests := []struct {
        name      string
        base      [128]uint64
        others    [][128]uint64
        expected  float64
    }{
        {
            name: "perfect match",
            base: testArray,
            others: [][128]uint64{testArray},
            expected: 1.0,
        },
        {
            name: "one difference",
            base: testArray,
            others: func() [][128]uint64 {
                t := testArray
                t[127] -= 1
                u := t
                u[1] -= 6
                u[0] -= 2
                return [][128]uint64{t, u}
            }(),
            expected: 127.0 / 128.0,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            sim := calcSim(tt.base, tt.others)
            if sim != tt.expected {
                t.Errorf("unexpected sim %f, instead of %f", sim, tt.expected)
            }
        })
    }
}