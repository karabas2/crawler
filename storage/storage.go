// Package storage provides thread-safe in-memory storage for crawled pages
// and an inverted index for full-text search.
package storage

import (
	"strings"
	"sync"
)

// PageData represents a single crawled web page with its metadata.
type PageData struct {
	URL       string   // The URL of this page
	Title     string   // The <title> extracted from the HTML
	Body      string   // The text content of the page (tags stripped)
	Links     []string // All outgoing links found on this page
	OriginURL string   // The page from which the crawler discovered this URL
	Depth     int      // The crawl depth at which this page was found
}

// Storage is the central, thread-safe data store.
// It holds crawled pages and an inverted index for search.
type Storage struct {
	// pages stores all crawled pages keyed by URL.
	pages   map[string]*PageData
	pagesMu sync.RWMutex

	// index is an inverted index: token -> list of pages containing that token.
	index   map[string][]*PageData
	indexMu sync.RWMutex

	// stats tracks crawl/index statistics.
	stats   Stats
	statsMu sync.Mutex
}

// Stats holds runtime statistics about the crawl and index state.
type Stats struct {
	PagesCrawled  int    `json:"pages_crawled"`
	PagesIndexed  int    `json:"pages_indexed"`
	UniqueTokens  int    `json:"unique_tokens"`
	URLsQueued    int    `json:"urls_queued"`
	URLsProcessed int    `json:"urls_processed"`
	URLsFailed    int    `json:"urls_failed"`
	QueueDepth    int    `json:"queue_depth"`
	BackPressure  string `json:"back_pressure"`
}

// New creates and returns a new initialized Storage instance.
func New() *Storage {
	return &Storage{
		pages: make(map[string]*PageData),
		index: make(map[string][]*PageData),
	}
}

// StorePage saves a crawled page into the pages map.
// It is safe for concurrent use.
func (s *Storage) StorePage(page *PageData) {
	s.pagesMu.Lock()
	s.pages[page.URL] = page
	s.pagesMu.Unlock()

	s.statsMu.Lock()
	s.stats.PagesCrawled++
	s.statsMu.Unlock()
}

// GetPage retrieves a page by URL. Returns nil if not found.
// It is safe for concurrent use.
func (s *Storage) GetPage(url string) *PageData {
	s.pagesMu.RLock()
	defer s.pagesMu.RUnlock()
	return s.pages[url]
}

// HasPage checks whether a URL has already been crawled.
// It is safe for concurrent use.
func (s *Storage) HasPage(url string) bool {
	s.pagesMu.RLock()
	defer s.pagesMu.RUnlock()
	_, exists := s.pages[url]
	return exists
}

// AddToIndex inserts a page into the inverted index for every given token.
// It is safe for concurrent use.
func (s *Storage) AddToIndex(tokens []string, page *PageData) {
	s.indexMu.Lock()
	for _, token := range tokens {
		token = strings.ToLower(token)
		s.index[token] = append(s.index[token], page)
	}
	s.indexMu.Unlock()

	s.statsMu.Lock()
	s.stats.PagesIndexed++
	s.stats.UniqueTokens = s.uniqueTokenCountLocked()
	s.statsMu.Unlock()
}

// SearchIndex returns all pages that contain the given token.
// It is safe for concurrent use.
func (s *Storage) SearchIndex(token string) []*PageData {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()
	return s.index[strings.ToLower(token)]
}

// GetStats returns a snapshot of the current crawl/index statistics.
func (s *Storage) GetStats() Stats {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()
	return s.stats
}

// uniqueTokenCountLocked returns the number of unique tokens in the index.
// Caller must hold statsMu (or indexMu).
func (s *Storage) uniqueTokenCountLocked() int {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()
	return len(s.index)
}

// GetAllPages returns a snapshot of all stored pages.
// It is safe for concurrent use.
func (s *Storage) GetAllPages() []*PageData {
	s.pagesMu.RLock()
	defer s.pagesMu.RUnlock()
	pages := make([]*PageData, 0, len(s.pages))
	for _, p := range s.pages {
		pages = append(pages, p)
	}
	return pages
}

// IncrURLsQueued increments the URLs queued counter.
func (s *Storage) IncrURLsQueued() {
	s.statsMu.Lock()
	s.stats.URLsQueued++
	s.statsMu.Unlock()
}

// IncrURLsProcessed increments the URLs processed counter.
func (s *Storage) IncrURLsProcessed() {
	s.statsMu.Lock()
	s.stats.URLsProcessed++
	s.statsMu.Unlock()
}

// IncrURLsFailed increments the URLs failed counter.
func (s *Storage) IncrURLsFailed() {
	s.statsMu.Lock()
	s.stats.URLsFailed++
	s.statsMu.Unlock()
}

// UpdateQueueDepth sets the current queue depth.
func (s *Storage) UpdateQueueDepth(depth int) {
	s.statsMu.Lock()
	s.stats.QueueDepth = depth
	s.statsMu.Unlock()
}

// UpdateBackPressure sets the current back-pressure status.
func (s *Storage) UpdateBackPressure(status string) {
	s.statsMu.Lock()
	s.stats.BackPressure = status
	s.statsMu.Unlock()
}

// Clear resets all storage state (pages, index, and stats).
func (s *Storage) Clear() {
	s.pagesMu.Lock()
	s.pages = make(map[string]*PageData)
	s.pagesMu.Unlock()

	s.indexMu.Lock()
	s.index = make(map[string][]*PageData)
	s.indexMu.Unlock()

	s.statsMu.Lock()
	s.stats = Stats{}
	s.statsMu.Unlock()
}
