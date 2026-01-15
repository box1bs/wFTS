package textHandling

import "testing"

func TestStemming(t *testing.T) {
	engStemmer := NewEnglishStemmer()
	simParts := [][]string{{"connect", "connected", "connecting", "connection", "connections"},
		{"decept", "deception"},
		{"electriciti", "electrical"},
		{"activ", "activate"},
        {"poni", "ponies"},
        {"ti", "ties"},
        {"meet", "meeting"},
	}
	for _, s := range simParts {
		base := engStemmer.stem(s[0])
		for i := 1; i < len(s); i++ {
			if stemmed := engStemmer.stem(s[i]); stemmed != base {
				t.Errorf("invalid stemming %s, instead of %s", stemmed, base)
				return
			}
		}
	}
}

func TestTokenizing(t *testing.T) {
	tokenizer := newTokenizer()
	testString := "Hello, world!"
	tokens := tokenizer.entityTokenize(testString)
	if l := len(tokens); l != 4 {
		t.Errorf("invalid len tokenized string for word test: %d, instead of %d", l, 4)
		return
	}
	if tokens[0].Type != WORD || tokens[1].Type != SYMBOL || tokens[2].Type != WORD || tokens[3].Type != SYMBOL {
		t.Errorf("invalid token type or order, %v", tokens)
		return
	}
	testString = "192.168.1.1, https://web.stanford.edu/class/cs224n/"
	tokens = tokenizer.entityTokenize(testString)
	if l := len(tokens); l != 3 {
		t.Errorf("invalid len tokenized string for ip and url test: %d, instead of %d", l, 3)
		return
	}
	if tokens[0].Type != IP_V4_ADDR || tokens[1].Type != SYMBOL || tokens[2].Type != URL_ADDR {
		t.Errorf("invalid ip and url regular expression, %v", tokens)
	}
}