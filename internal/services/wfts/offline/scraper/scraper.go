package scraper

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
	"log/slog"

	"wfts/internal/model"
	"wfts/internal/services/wfts/offline/scraper/lruCache"
	"wfts/internal/utils/parser"

	"context"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type indexer interface {
    HandleDocumentWords(*model.Document, []model.Passage) error
	SaveUrlsToBank([32]byte, []byte) error
	GetUrlsByHash([32]byte) ([]byte, error)
}

type workerPool interface {
	Submit(model.CrawlNode)
	Wait()
	Stop()
}

type WebScraper struct {
	client         	*http.Client
	visited        	*sync.Map
	cfg 		  	*ConfigData
	log 			*slog.Logger
	rlMu         	*sync.RWMutex
	lru 			*lrucache.LRUCache
	pool           	workerPool
	idx 			indexer
	globalCtx		context.Context
	rlMap			map[string]*rateLimiter
	rulesMap		map[string]*parser.RobotsTxt
}

type ConfigData struct {
	StartURLs     	[]string
	CacheCap 		int
	Depth       	int
	OnlySameDomain  bool
}

const (
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:140.0) Gecko/20100101 Firefox/140.0"
 	crawlTime = 600 * time.Second
 	deadlineTime = 30 * time.Second
	numOfTries = 3 // если кто то решил поменять это на 0, чтож, удачи
)

func NewScraper(mp *sync.Map, cfg *ConfigData, l *slog.Logger, wp workerPool, idx indexer, c context.Context) *WebScraper {
	return &WebScraper{
		client: &http.Client{
			Timeout: deadlineTime,
			Transport: &http.Transport{
				IdleConnTimeout:   15 * time.Second,
				DisableKeepAlives: false,
				ForceAttemptHTTP2: true,
			},
		},
		visited:        mp,
		cfg: 			cfg,
		log:			l,
		rlMu:           new(sync.RWMutex),
		lru: 			lrucache.NewLRUCache(cfg.CacheCap),
		pool:           wp,
		idx: 			idx,
		globalCtx:		c,
		rlMap: 			make(map[string]*rateLimiter),
		rulesMap: 		make(map[string]*parser.RobotsTxt),
	}
}

func (ws *WebScraper) Run() {
	defer ws.putDownLimiters()
	for _, uri := range ws.cfg.StartURLs {
		parsed, err := url.Parse(uri)
		if err != nil {
			ws.log.Error("parsing url failed: " + err.Error())
			continue
		}
		ws.pool.Submit(model.CrawlNode{Activation: func() {
			ctx, cancel := context.WithTimeout(ws.globalCtx, crawlTime)
			defer cancel()
			ws.rlMu.Lock()
			rl := NewRateLimiter(DefaultDelay)
			ws.rlMap[parsed.Host] = rl
			ws.rlMu.Unlock()
			ws.ScrapeWithContext(ctx, parsed, 0)
		}})
	}
	ws.pool.Wait()
	ws.log.Debug("waiting for stoppnig worker pool")
	ws.pool.Stop()
}

func (ws *WebScraper) ScrapeWithContext(ctx context.Context, currentURL *url.URL, depth int) {
    if ws.checkContext(ctx, currentURL.String()) {return}

    if depth >= ws.cfg.Depth {
        return
    }
	
    normalized, err := normalizeUrl(currentURL.String())
    if err != nil {
		return
    }
	
	links, rls, err := ws.fetchPageRulesAndOffers(ctx, currentURL)
	if err.Error() == BaseXMLPageError || ws.checkContext(ctx, currentURL.String()) {
		return
	}
	host := truncatePort(currentURL)
	ws.rlMu.Lock()
	if rls != nil && ws.rulesMap[host] == nil {
		ws.rulesMap[host] = rls
	}
	ws.rlMu.Unlock()
	hashed := sha256.Sum256([]byte(normalized))
	load := false
    
	if len(links) == 0 && (err == nil || err.Error() != BaseXMLPageError) {
		if prevDepth, loaded := ws.visited.LoadOrStore(normalized, depth); loaded && prevDepth.(int) <= depth {
			return
		} else if loaded {
			load = true
			if v := ws.lru.Get(hashed); v != nil {
				log.Println("I was called!")
				links = v.([]*linkToken)
			} else {
				encoded, err := ws.idx.GetUrlsByHash(hashed)
				if err != nil {
					if err.Error() != "Key not found" {
						ws.log.Error("error getting urls, from db: " + err.Error())
					}
					return
				}
				if err := gob.NewDecoder(bytes.NewBuffer(encoded)).Decode(&links); err != nil {
					ws.log.Error("error unmarshalling urls from db: " + err.Error())
					return
				}
				if len(links) != 0 {
					ws.lru.Put(hashed, links)
				}
			}
		} else {
			links, err = ws.fetchHTMLcontent(currentURL, ctx, normalized, depth)
			if err != nil {
				return
			}
		}
		
		if len(links) == 0 {
			ws.log.Debug("empty links in page " + currentURL.String())
			return
		}

		if !load {
			var buf bytes.Buffer
			if err := gob.NewEncoder(&buf).Encode(links); err != nil {
				ws.log.Error("error marshalling urls: " + err.Error())
				return
			}

			if err := ws.idx.SaveUrlsToBank(hashed, buf.Bytes()); err != nil {
				ws.log.Error("error saving urls: " + err.Error())
				return
			}
		}
	}
	
	for _, link := range links {	
		if ws.cfg.OnlySameDomain && !link.SameDomain {
			continue
		}

		if ws.checkContext(ws.globalCtx, currentURL.String()) { return }

        ws.pool.Submit(model.CrawlNode{Activation: func() {
			ws.rlMu.Lock()
			if ws.rlMap[link.Link.Host] == nil {
				ws.rlMap[link.Link.Host] = NewRateLimiter(DefaultDelay)
			}
			c, cancel := context.WithTimeout(ws.globalCtx, crawlTime)
			defer cancel()
			ws.rlMu.Unlock()
			ws.ScrapeWithContext(c, link.Link, depth+1)
		},
			Depth: depth,
			SameDomain: link.SameDomain,
		})
    }
}

func (ws *WebScraper) putDownLimiters() {
	ws.rlMu.Lock()
	defer ws.rlMu.Unlock()
	for _, limiter := range ws.rlMap {
		limiter.Shutdown()
	}
}

func (ws *WebScraper) checkContext(ctx context.Context, currentURL string) bool {
	select {
		case <-ctx.Done():
			select {
			case <-ws.globalCtx.Done(): // чтоб не выводилась куча логов при остановке кровлинга
				return true
			default:
			}
			ws.log.Debug("context canceled while parsing page: " + currentURL)
			return true
		default:
	}
	return false
}