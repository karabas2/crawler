package search

import (
	"sort"
	"webcrawler/indexer"
	"webcrawler/ranking"
	"webcrawler/storage"
)

// Result represents a single search result with the required triples.
type Result struct {
	RelevantURL string  `json:"relevant_url"`
	OriginURL   string  `json:"origin_url"`
	Depth       int     `json:"depth"`
	Score       float64 `json:"score"`
}

// Engine matches queries to indexed pages and ranks them.
type Engine struct {
	store  *storage.Storage
	ranker *ranking.Ranker
}

// New creates a new Search Engine.
func New(store *storage.Storage, ranker *ranking.Ranker) *Engine {
	return &Engine{
		store:  store,
		ranker: ranker,
	}
}

// Search performs a query and returns sorted results.
func (e *Engine) Search(query string) []Result {
	queryTokens := indexer.Tokenize(query)
	if len(queryTokens) == 0 {
		return nil
	}

	// Match candidates
	seen := make(map[string]bool)
	var candidates []*storage.PageData

	for _, token := range queryTokens {
		pages := e.store.SearchIndex(token)
		for _, p := range pages {
			if !seen[p.URL] {
				seen[p.URL] = true
				candidates = append(candidates, p)
			}
		}
	}

	// Rank results
	results := make([]Result, 0, len(candidates))
	for _, p := range candidates {
		score := e.ranker.Score(p, queryTokens)
		results = append(results, Result{
			RelevantURL: p.URL,
			OriginURL:   p.OriginURL,
			Depth:       p.Depth,
			Score:       score,
		})
	}

	// Sort results descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}
