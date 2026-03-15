// Package search provides the query interface for the search engine.
package search

import (
	"sort"

	"webcrawler/indexer"
	"webcrawler/ranking"
	"webcrawler/storage"
)

// Result represents a single search result with the required triple plus a score.
type Result struct {
	RelevantURL string  `json:"relevant_url"` // The page matching the query
	OriginURL   string  `json:"origin_url"`   // The page that discovered it
	Depth       int     `json:"depth"`        // Crawl depth
	Score       float64 `json:"score"`        // Relevancy score
}

// Engine is the search engine that queries the inverted index and ranks results.
type Engine struct {
	store  *storage.Storage
	ranker *ranking.Ranker
}

// New creates a new search Engine.
func New(store *storage.Storage, ranker *ranking.Ranker) *Engine {
	return &Engine{
		store:  store,
		ranker: ranker,
	}
}

// Search performs a search for the given query string.
// It tokenizes the query, looks up each token in the inverted index,
// merges results, scores them, and returns sorted results.
func (e *Engine) Search(query string) []Result {
	queryTokens := indexer.Tokenize(query)
	if len(queryTokens) == 0 {
		return nil
	}

	// Collect unique matching pages from the inverted index
	seen := make(map[string]bool)
	var candidates []*storage.PageData

	for _, token := range queryTokens {
		pages := e.store.SearchIndex(token)
		for _, page := range pages {
			if !seen[page.URL] {
				seen[page.URL] = true
				candidates = append(candidates, page)
			}
		}
	}

	// Score each candidate
	results := make([]Result, 0, len(candidates))
	for _, page := range candidates {
		score := e.ranker.Score(page, queryTokens)
		results = append(results, Result{
			RelevantURL: page.URL,
			OriginURL:   page.OriginURL,
			Depth:       page.Depth,
			Score:       score,
		})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}
