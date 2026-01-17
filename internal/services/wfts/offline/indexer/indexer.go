package indexer

import (
	"context"
	"fmt"
	"sync"

	"wfts/configs"
	"wfts/internal/services/wfts/offline/indexer/spellChecker"
	"wfts/internal/services/wfts/offline/indexer/textHandling"
	"wfts/internal/services/wfts/offline/scraper"
	"wfts/internal/model"
	"wfts/pkg/logger"
	"wfts/internal/utils/workerPool"
)

type repository interface {
	IndexUrlsByHash([32]byte, []byte) error
	GetPageUrlsByHash([32]byte) ([]byte, error)

	LoadVisitedUrls(*sync.Map) error
	SaveVisitedUrls(*sync.Map) error

	IndexNGrams([]string, int) error
	GetWordsByNGram(string, int) ([]string, error)
	IndexDocShingles([128]uint64) error
	GetSimilarSignatures([128]uint64) ([][128]uint64, error)
	FlushAll()

	UpdateBiFreq(map[[2]uint64]int) error
	GetFreq(uint64, uint64) (int, error)

	SaveSaltArrays([128]uint64, [128]uint64) error
	UploadSaltArrays() ([128]uint64, [128]uint64, error)

	IndexDocumentWords([32]byte, map[string]int, map[string][]model.Position) error
	GetDocumentsByWord(string) (map[[32]byte]model.WordCountAndPositions, error)

	SaveDocument(*model.Document) error
	GetDocumentByID([32]byte) (*model.Document, error)
	GetAllDocuments() ([]*model.Document, error)
	GetDocumentsCount() (int, error)
}

type indexer struct {
	spider 		*scraper.WebScraper
	stemmer 	*textHandling.EnglishStemmer
	sc 			*spellChecker.SpellChecker
	logger 		*logger.Logger
	vectorizer 	*textHandling.Vectorizer
	minHash 	*minHash
	mu 			*sync.RWMutex
	repository 	repository
}

func NewIndexer(repo repository, vec *textHandling.Vectorizer, log *logger.Logger, config *configs.ConfigData) *indexer {
	return &indexer{
		vectorizer: vec,
		stemmer:   	textHandling.NewEnglishStemmer(),
		mu: 		new(sync.RWMutex),
		repository: repo,
		sc: 		spellChecker.NewSpellChecker(config.MaxTypo, config.NGramCount),
		logger:    	log,
	}
}

func (idx *indexer) Index(config *configs.ConfigData, global context.Context) error {
	vis := &sync.Map{}
	if err := idx.repository.LoadVisitedUrls(vis); err != nil {
		return err
	}
	defer idx.repository.SaveVisitedUrls(vis)
	defer idx.repository.FlushAll()
	if a, b, err := idx.repository.UploadSaltArrays(); err != nil && err.Error() != "Key not found" {
		return err
	} else if err != nil && err.Error() == "Key not found" {
		if c, err := idx.repository.GetDocumentsCount(); err != nil {
			return err
		} else if c != 0 {
			return fmt.Errorf("index isn't empty, but salt arrays is")
		}
		idx.minHash = NewHasher(a, b, true) // пересоздаем
	} else {
		idx.minHash = NewHasher(a, b, false) // просто получаем структуру
	}
	defer idx.repository.SaveSaltArrays(idx.minHash.a, idx.minHash.b)

	idx.spider = scraper.NewScraper(vis, &scraper.ConfigData{
		StartURLs:     	config.BaseURLs,
		CacheCap: 		config.CacheCap,	
		Depth:       	config.MaxDepth,
		OnlySameDomain: config.OnlySameDomain,
	}, idx.logger,
		workerPool.NewWorkerPool(config.WorkersCount, config.TasksCount, global, idx.logger),
		idx, global, idx.vectorizer.PutDocQuery)
	idx.spider.Run()
	return nil
}

func (idx *indexer) GetAVGLen() (float64, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var wordCount int
	docs, err := idx.repository.GetAllDocuments()
	if err != nil {
		return 0, err
	}

	for _, doc := range docs {
		wordCount += doc.WordCount
	}

	return float64(wordCount) / (float64(len(docs)) + 1), nil
}

func (idx *indexer) SaveUrlsToBank(key [32]byte, data []byte) error {
	return idx.repository.IndexUrlsByHash(key, data)
}

func (idx *indexer) GetUrlsByHash(key [32]byte) ([]byte, error) {
	return idx.repository.GetPageUrlsByHash(key)
}