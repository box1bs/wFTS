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

	allTokens := []string{}
	for _, passage := range passages {
		orig, stemmed, err := idx.stemmer.TokenizeAndStem(passage.Text)
		if err != nil {
			return err
		}
		if len(stemmed) == 0 {
			continue
		}

		allTokens = append(allTokens, orig...)
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

	sign := idx.minHash.CreateSignature(allTokens)
	conds, err := idx.repository.GetSimilarSignatures(sign)
	if err != nil {
		return err
	}
	if simRate := calcSim(sign, conds); simRate > 64 { // > 50% нграмм
		idx.logger.Write(logger.NewMessage(logger.INDEX_LAYER, logger.DEBUG, "finded %d/128 similar page", simRate))
		return fmt.Errorf("page already indexed")
	}
	if err := idx.repository.IndexDocShingles(sign); err != nil {
		return err
	}

	bigrams := make(map[[2]uint64]int)
	for i := 1; i < doc.WordCount; i++ {
		bigrams[[2]uint64{idx.minHash.Hash64(allTokens[i - 1]), idx.minHash.Hash64(allTokens[i])}]++
	}
	if err := idx.repository.UpdateBiFreq(bigrams); err != nil {
		return err
	}
	if err := idx.repository.SaveDocument(doc); err != nil {
		idx.logger.Write(logger.NewMessage(logger.INDEX_LAYER, logger.CRITICAL_ERROR, "error saving document: %v", err))
		return err
	}
	if err := idx.repository.IndexNGrams(allTokens, idx.sc.NGramCount); err != nil {
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
	wordp := 0

	for i, lemma := range stemmed {
		documents, err := idx.repository.GetDocumentsByWord(lemma.Value)
		if err != nil {
			return nil, nil, err
		}
		if len(documents) == 0 && lemma.Type == textHandling.WORD && len(words) > i {
			conds, err := idx.repository.GetWordsByNGram(words[wordp], idx.sc.NGramCount)
			if err != nil {
				return nil, nil, err
			}
			lenCandidates := len(conds)
			scores := make([][2]float64, lenCandidates)
			left := uint64(0)
			if i > 0 && i < len(words) {
				left = idx.minHash.Hash64(words[i - 1])
			}
			right := uint64(0)
			if i + 1 < lenWords {
				right = idx.minHash.Hash64(words[i + 1])
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
			replacement := idx.sc.BestReplacement(words[wordp], conds, scores)
			idx.logger.Write(logger.NewMessage(logger.INDEX_LAYER, logger.DEBUG, "word '%s' replaced with '%s' in query", words[wordp], replacement))
			_, stem, err := idx.stemmer.TokenizeAndStem(replacement)
			if err != nil {
				return nil, nil, err
			}
			if stem[0].Value == "" { // если заменяется на стоп слово
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
			wordp++
		}
	}

	return stemmedTokens, reverthIndex, err
}

func calcSim(curSign [128]uint64, condidates [][128]uint64) int {
	result := 0
	l := len(condidates)
	for i := range l {
		sum := 0
		for j := range 128 {
			if curSign[j] == condidates[i][j] {
				sum++
			}
		}
		result = max(result, sum)
	}
	return result
}