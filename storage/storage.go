package storage

import (
	"sync"
)

// PageData represents a single crawled web page with its metadata.
type PageData struct {
	URL       string   `json:"url"`
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Links     []string `json:"links"`
	OriginURL string   `json:"origin_url"`
	Depth     int      `json:"depth"`
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
	BackPressure  string `json:"back_pressure"` // NORMAL, MODERATE, HIGH
}

// Storage handles thread-safe in-memory storage and indexing.
type Storage struct {
	Mu      sync.RWMutex
	Pages   map[string]*PageData
	Index   map[string][]*PageData // token -> list of pages
	CrawlerStats Stats
}

// New initializes a new Storage instance.
func New() *Storage {
	return &Storage{
		Pages: make(map[string]*PageData),
		Index: make(map[string][]*PageData),
	}
}

// AddPage adds a page and updates statistics.
func (s *Storage) AddPage(p *PageData) {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	if _, exists := s.Pages[p.URL]; !exists {
		s.Pages[p.URL] = p
		s.CrawlerStats.PagesCrawled++
	}
}

// AddToIndex maps a list of unique tokens to a page.
func (s *Storage) AddToIndex(tokens []string, p *PageData) {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	for _, t := range tokens {
		s.Index[t] = append(s.Index[t], p)
	}
	s.CrawlerStats.UniqueTokens = len(s.Index)
	s.CrawlerStats.PagesIndexed++
}

// SearchIndex returns pages containing the token.
func (s *Storage) SearchIndex(token string) []*PageData {
	s.Mu.RLock()
	defer s.Mu.RUnlock()
	return s.Index[token]
}

// UpdateStats updates the crawler metrics.
func (s *Storage) UpdateStats(update func(*Stats)) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	update(&s.CrawlerStats)
	
	// Derive backpressure status
	if s.CrawlerStats.QueueDepth > 5000 {
		s.CrawlerStats.BackPressure = "HIGH"
	} else if s.CrawlerStats.QueueDepth > 1000 {
		s.CrawlerStats.BackPressure = "MODERATE"
	} else {
		s.CrawlerStats.BackPressure = "NORMAL"
	}
}

// GetStats returns a copy of current stats.
func (s *Storage) GetStats() Stats {
	s.Mu.RLock()
	defer s.Mu.RUnlock()
	return s.CrawlerStats
}
