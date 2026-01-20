package textHandling

import "testing"

func TestStemming(t *testing.T) {
	engStemmer := NewEnglishStemmer()
		tests := []struct {
		name     string
		toStem   []string
		expected string
	}{
		{
			name:     "connected forms",
			toStem:   []string{"connected", "connecting", "connection", "connections"},
			expected: "connect",
		},
		{
			name:     "deception",
			toStem:   []string{"deception"},
			expected: "decept",
		},
		{
			name:     "activate",
			toStem:   []string{"activate"},
			expected: "activ",
		},
		{
			name:     "meeting",
			toStem:   []string{"meeting"},
			expected: "meet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, word := range tt.toStem {
				if stemmed := engStemmer.stem(word); stemmed != tt.expected {
					t.Errorf("stem() = %s, want %s", stemmed, tt.expected)
				}
			}
		})
	}
}

func TestTokenizing(t *testing.T) {
	tokenizer := newTokenizer()
	tests := []struct {
		name     string
		in       string
		expected []token
	}{
		{
			name: "base case",
			in:   "Hello, world!",
			expected: []token{
				{
					Type: WORD, 
					Value: "hello", 
					startPos: 0, 
					endPos: 5,
				},
				{
					Type: SYMBOL, 
					Value: ",", 
					startPos: 5, 
					endPos: 6,
				},
				{
					Type: WORD, 
					Value: "world", 
					startPos: 7, 
					endPos: 12,
				},
				{
					Type: SYMBOL, 
					Value: "!", 
					startPos: 12, 
					endPos: 13,
				},
			},
		},
		{
			name: "ip and url case",
			in:   "192.168.1.1, https://web.stanford.edu/class/cs224n/",
			expected: []token{
				{
					Type: IP_V4_ADDR, 
					Value: "192.168.1.1", 
					startPos: 0, 
					endPos: 11,
				},
				{
					Type: SYMBOL, 
					Value: ",", 
					startPos: 11, 
					endPos: 12,
				},
				{
					Type: URL_ADDR, 
					Value: "https://web.stanford.edu/class/cs224n/", 
					startPos: 13, 
					endPos: 51,
				},
			},
		},
	}
	for iter := range tests {
		t.Run(tests[0].name, func(t *testing.T) {
			for i, token := range tokenizer.entityTokenize(tests[iter].in) {
				if token.Type != tests[iter].expected[i].Type {
					t.Errorf("%d, want %d", token.Type, tests[iter].expected[i].Type)
				}
				if token.Value != tests[iter].expected[i].Value {
					t.Errorf("%s, want %s", token.Value, tests[iter].expected[i].Value)
				}
				if token.startPos != tests[iter].expected[i].startPos {
					t.Errorf("%d, want %d", token.startPos, tests[iter].expected[i].startPos)
				}
				if token.endPos != tests[iter].expected[i].endPos {
					t.Errorf("%d, want %d", token.endPos, tests[iter].expected[i].endPos)
				}
			}
		})
	}
}