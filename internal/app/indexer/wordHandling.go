package indexer

import (
	"fmt"
	"math"

	"github.com/box1bs/wFTS/internal/app/indexer/textHandling"
	"github.com/box1bs/wFTS/internal/model"
	"github.com/box1bs/wFTS/pkg/logger"
)

func (idx *indexer) HandleDocumentWords(doc *model.Document, passages []model.Passage) error {
	stem := map[string]int{}
	i := 0
	pos := map[string][]model.Position{}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	allWordTokens := []string{}
	for _, passage := range passages {
		orig, stemmed, err := idx.stemmer.TokenizeAndStem(passage.Text)
		if err != nil {
			return err
		}
		if len(stemmed) == 0 {
			continue
		}

		allWordTokens = append(allWordTokens, orig...)
		for _, w := range stemmed {
			if w.Type == textHandling.NUMBER || len(w.Value) > 64 {
				continue
			}
			stem[w.Value]++
			pos[w.Value] = append(pos[w.Value], model.NewTypeTextObj[model.Position](passage.Type, "", i))
			i++
		}
	}
	doc.WordCount = i

	if len(allWordTokens) > 4 {
		sign := idx.minHash.CreateSignature(allWordTokens)
		conds, err := idx.repository.GetSimilarSignatures(sign)
		if err != nil {
			return err
		}
		if simRate := calcSim(sign, conds); simRate > 0.8 {
			idx.logger.Write(logger.NewMessage(logger.INDEX_LAYER, logger.DEBUG, "finded %f similar page: %s, with word tokens len: %d", simRate, doc.URL, len(allWordTokens)))
			return fmt.Errorf("page already indexed")
		}
		if err := idx.repository.IndexDocShingles(sign); err != nil {
			return err
		}
	}

	bigrams := make(map[[2]uint64]int)
	for j := 1; j < len(allWordTokens); j++ {
		bigrams[[2]uint64{idx.minHash.Hash64(allWordTokens[j - 1]), idx.minHash.Hash64(allWordTokens[j])}]++
	}
	if err := idx.repository.UpdateBiFreq(bigrams); err != nil {
		return err
	}
	if err := idx.repository.SaveDocument(doc); err != nil {
		idx.logger.Write(logger.NewMessage(logger.INDEX_LAYER, logger.CRITICAL_ERROR, "error saving document: %v", err))
		return err
	}
	if err := idx.repository.IndexNGrams(allWordTokens, idx.sc.NGramCount); err != nil {
		idx.logger.Write(logger.NewMessage(logger.INDEX_LAYER, logger.CRITICAL_ERROR, "error indexing ngrams: %v", err))
		return err
	}
	if err := idx.repository.IndexDocumentWords(doc.Id, stem, pos); err != nil {
		idx.logger.Write(logger.NewMessage(logger.INDEX_LAYER, logger.CRITICAL_ERROR, "error indexing document words: %v", err))
		return err
	}

	return nil
}

func (idx *indexer) HandleTextQuery(text string) ([]string, []map[[32]byte]model.WordCountAndPositions, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	reverthIndex := []map[[32]byte]model.WordCountAndPositions{}
	words, stemmed, err := idx.stemmer.TokenizeAndStem(text)
	if len(stemmed) == 0 {
		return nil, nil, fmt.Errorf("empty tokens")
	}
	lenWords := len(words)
	stemmedTokens := []string{}
	wordPos := 0

	for i, lemma := range stemmed {
		documents, err := idx.repository.GetDocumentsByWord(lemma.Value)
		if err != nil {
			return nil, nil, err
		}
		if len(documents) == 0 && lemma.Type == textHandling.WORD { // исправляем только слова
			conds, err := idx.repository.GetWordsByNGram(words[wordPos], idx.sc.NGramCount)
			if err != nil {
				return nil, nil, err
			}
			lenCandidates := len(conds)
			scores := make([][2]float64, lenCandidates)
			left := uint64(0)
			if wordPos > 0 {
				left = idx.minHash.Hash64(words[wordPos - 1])
			}
			right := uint64(0)
			if wordPos + 1 < lenWords {
				right = idx.minHash.Hash64(words[wordPos + 1])
			}
			if right != 0 || left != 0 {
				for j := range lenCandidates {
					cond := idx.minHash.Hash64(conds[j])
					lscore := 0
					if left != 0 {
						lscore, err = idx.repository.GetFreq(left, cond)
						if err != nil {
							return nil, nil, err
						}
					}
					rscore := 0
					if right != 0 {
						rscore, err = idx.repository.GetFreq(cond, right)
						if err != nil {
							return nil, nil, err
						}
					}
					scores[j][0], scores[j][1] = math.Log(float64(1 + lscore)), math.Log(float64(1 + rscore)) // снижаем зависимость результата от контекстуального совпадения
				}
			}
			replacement := idx.sc.BestReplacement(words[wordPos], conds, scores)
			idx.logger.Write(logger.NewMessage(logger.INDEX_LAYER, logger.DEBUG, "word '%s' replaced with '%s' in query", words[wordPos], replacement))
			_, stem, err := idx.stemmer.TokenizeAndStem(replacement)
			if err != nil {
				return nil, nil, err
			}
			if stem[0].Value == "" { // если заменяется на стоп слово
				wordPos++
				continue
			}
			stemmed[i] = stem[0]
			documents, err = idx.repository.GetDocumentsByWord(stem[0].Value)
			if err != nil {
				return nil, nil, err
			}
		}
		stemmedTokens = append(stemmedTokens, stemmed[i].Value)
		reverthIndex = append(reverthIndex, documents)
		if lemma.Type == textHandling.WORD {
			wordPos++
		}
	}

	return stemmedTokens, reverthIndex, err
}

func calcSim(curSign [128]uint64, condidates [][128]uint64) float64 {
	result := 0.0
	l := len(condidates)
	for i := range l {
		sum := 0
		for j := range 128 {
			if curSign[j] == condidates[i][j] {
				sum++
			}
		}
		sim := float64(sum) / 128.0
		result = max(result, sim)
	}
	return result
}