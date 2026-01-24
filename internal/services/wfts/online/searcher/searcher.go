package searcher

import (
	"math"
	"sort"
	"strings"
	"sync"

	"wfts/internal/model"
)

type index interface {
	HandleTextQuery(string) ([]string, []map[[32]byte]model.WordCountAndPositions, error)
	GetAVGLen() (float64, error)
}

type resitory interface {
	GetDocumentsCount() (int, error)
	GetDocumentByID([32]byte) (*model.Document, error)
}

type Searcher struct {
	log 		*model.Logger
	mu         	*sync.RWMutex
	idx 		index
	repo 	 	resitory
}

func NewSearcher(log *model.Logger, idx index, repo resitory) *Searcher {
	return &Searcher{
		log: 		log,
		mu:        	&sync.RWMutex{},
		idx:       	idx,
		repo: 	 	repo,
	}
}

type requestRanking struct {
	tf_idf 				float64
	bm25 				float64
	logLenWordInURL 	float64
	termProximity 		int
	hasWordInHeader 	bool
	//any ranking scores
}

func (s *Searcher) Search(query string, maxLen int) []*model.Document {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	words, index, err := s.idx.HandleTextQuery(query)
	if err != nil {
		s.log.Errorf(NewSearchAttrs(query), "handling words error: %v",  err)
		return nil
	}
	
	queryLen := len(words)
	
	avgLen, err := s.idx.GetAVGLen()
	if err != nil {
		s.log.Errorf(NewSearchAttrs(query), "%v", err)
		return nil
	}
	
	length, err := s.repo.GetDocumentsCount()
	if err != nil {
		s.log.Errorf(NewSearchAttrs(query), "%v", err)
		return nil
	}
	
	rank := make(map[[32]byte]requestRanking)
	result := make([]*model.Document, 0)
	alreadyIncluded := make(map[[32]byte]struct{})
	var wg sync.WaitGroup
	var rankMu sync.RWMutex
	var resultMu sync.Mutex
	tokenFreq := make([]int, queryLen)
	done := make(chan struct{})

	for i := range words {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
	
			idf := math.Log(float64(length) / float64(len(index[i]) + 1)) + 1
			s.log.Infof(NewSearchAttrs(query), "len documents with word: %s, %d", words[i], len(index[i]))
			for _, item := range index[i] {
				tokenFreq[i] += item.Count
			}
	
			for docID, item := range index[i] {
				rankMu.RLock()
				doc, err := s.repo.GetDocumentByID(docID)
				if err != nil || doc == nil {
					rankMu.RUnlock()
					s.log.Infof(NewSearchAttrs(query), "error: %v, doc: %v", err, doc)
					continue
				}
				rankMu.RUnlock()
				
				rankMu.Lock()
				r := rank[docID]
				tf := float64(tokenFreq[i]) / float64(doc.TokenCount)
				r.tf_idf += tf * idf
				r.bm25 += calcBM25(idf, tf, doc, avgLen)
				if r.termProximity == 0 {
					positions := [][]model.Position{}
					for i := range words {
						positions = append(positions, index[i][docID].Positions)
					}
					r.termProximity = getMinQueryDistInDoc(positions, queryLen)
					_, r.logLenWordInURL = boyerMoorAlgorithm(strings.ToLower(doc.URL), words)
					for i := 0; i < item.Count && !r.hasWordInHeader; i++ {
						r.hasWordInHeader = item.Positions[i].Type == model.HeaderType
					}
				}
				rank[docID] = r
				rankMu.Unlock()

				resultMu.Lock()
				if _, exists := alreadyIncluded[doc.Id]; exists {
					resultMu.Unlock()
					continue
				}
				alreadyIncluded[docID] = struct{}{}
				result = append(result, doc)
				resultMu.Unlock()
			}
		}(i)
	}
	
	go func() {
		wg.Wait()
		close(done)
	}()
	
	s.log.Infof(NewSearchAttrs(query), "result len: %d", len(result))
	
	<-done

	length = len(result)
	if length == 0 {
		s.log.Infof(NewSearchAttrs(query), "empty result")
		return nil
	}

	sort.Slice(result, func(i, j int) bool {
		if rank[result[i].Id].bm25 != rank[result[j].Id].bm25 {
			return rank[result[i].Id].bm25 > rank[result[j].Id].bm25
		}
		if rank[result[i].Id].tf_idf != rank[result[j].Id].tf_idf {
			return rank[result[i].Id].tf_idf > rank[result[j].Id].tf_idf
		}
		return rank[result[i].Id].termProximity > rank[result[j].Id].termProximity
	})

	topN := result[:min(length, maxLen)]
	sort.Slice(topN, func(i, j int) bool {
		if rank[topN[i].Id].logLenWordInURL != rank[topN[j].Id].logLenWordInURL {
			return rank[topN[i].Id].logLenWordInURL > rank[topN[i].Id].logLenWordInURL
		}
		return rank[topN[i].Id].hasWordInHeader && !rank[topN[j].Id].hasWordInHeader
	})

	return topN
}